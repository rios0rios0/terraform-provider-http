//go:build integration

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
)

// fixtureBearer is a placeholder, not a credential shape any scanner recognises.
const fixtureBearer = "Bearer fixture-token-placeholder"

// authTracker records what a guarded server saw, so a test can assert on the request that was
// actually sent rather than on the resource's configuration.
type authTracker struct {
	mu           sync.Mutex
	authorized   map[string]int
	unauthorized map[string]int
}

// newGuardedServer answers 401 to any request without the expected bearer, which is what an API
// that authenticates through a header does. A test that regresses therefore fails on the provider's
// own error rather than on a subtle assertion.
func newGuardedServer(t *testing.T) (*httptest.Server, *authTracker) {
	t.Helper()

	tracker := &authTracker{authorized: map[string]int{}, unauthorized: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()

		if r.Header.Get("Authorization") != fixtureBearer {
			tracker.unauthorized[r.Method]++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprint(w, `{"message":"Invalid token."}`)

			return
		}

		tracker.authorized[r.Method]++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"id":1,"name":"fixture"}`)
	}))
	t.Cleanup(srv.Close)

	return srv, tracker
}

func (it *authTracker) counts(verb string) (authorized, unauthorized int) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return it.authorized[verb], it.unauthorized[verb]
}

// TestProviderHeadersReachTheImportRead is the load-bearing test for provider-level headers, and the
// reason the feature exists. A resource created with POST cannot have its import read replay that
// method, so an identifier names `import_read_path` and the provider force-GETs it instead. That GET
// is built from the identifier ALONE -- `ImportState` never sees the configuration -- so a bearer
// token kept in the resource's own `headers` cannot reach it, and the read comes back 401. Spelling
// the token into the identifier would work and would print it wherever plan output goes. A
// provider-level header is the only place it can live and still be sent.
func TestProviderHeadersReachTheImportRead(t *testing.T) {
	t.Run("should authenticate the read of an identifier that carries no headers", func(t *testing.T) {
		// given: the ONLY place the credential exists is the provider block
		srv, tracker := newGuardedServer(t)
		address := "http_request.guarded"
		providerConfig := builders.NewProviderTFBuilder().
			WithURL(srv.URL).
			WithHeaders(map[string]string{"Authorization": fixtureBearer}).
			Build()

		config := providerConfig + builders.NewResourceTFBuilder().
			WithName("guarded").
			WithMethod("POST").
			WithPath("/widgets").
			WithRequestBody(strconv.Quote(`{"name":"fixture"}`)).
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				// The create itself already depends on the provider header: without it the POST
				// would be rejected and this step would fail.
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(address, "response_body_id", "1"),
					),
				},

				// Forget the object without deleting it: `is_delete_enabled` is unset, so Destroy
				// only drops it from state. The remote object survives, which is the situation an
				// adopting import exists for.
				{Destroy: true, Config: config},

				// Re-import it. The identifier spells out no headers -- deliberately, so they stay
				// adoptable -- so the read it triggers can only be authenticated by the provider.
				{
					ResourceName:       address,
					ImportState:        true,
					ImportStatePersist: true,
					ImportStateId: `{"method":"POST","path":"/widgets",` +
						`"is_response_body_json":true,"response_body_id_filter":"$.id",` +
						`"import_read_path":"/widgets/1"}`,
					ImportStateCheck: func(states []*terraform.InstanceState) error {
						// then
						if got := states[0].Attributes["response_body_id"]; got != "1" {
							return fmt.Errorf("response_body_id = %q, want 1: the read must have "+
								"been authenticated and its body captured", got)
						}
						if states[0].Attributes["response_body"] == "" {
							return fmt.Errorf("response_body should have been captured by the read")
						}

						return nil
					},
				},

				// Adoption settles in place, and the plan after it is empty.
				{Config: config},
				{Config: config, PlanOnly: true},
			},
		})

		// then
		authorizedGets, unauthorizedGets := tracker.counts(http.MethodGet)
		if unauthorizedGets != 0 {
			t.Fatalf("the server rejected %d GET(s) as unauthenticated; the import read must carry "+
				"the provider-level header", unauthorizedGets)
		}
		if authorizedGets == 0 {
			t.Fatal("no authenticated GET reached the server, so the import read never happened")
		}

		authorizedPosts, unauthorizedPosts := tracker.counts(http.MethodPost)
		if unauthorizedPosts != 0 {
			t.Fatalf("the server rejected %d POST(s); provider headers must apply to create too",
				unauthorizedPosts)
		}
		if authorizedPosts != 1 {
			t.Fatalf("POST calls = %d, want exactly 1: adoption must not re-issue the create",
				authorizedPosts)
		}
	})
}

// TestProviderHeadersReachTheDestroyRequest covers the other request a resource's own `headers`
// cannot reach on its own: the destroy sends `delete_headers`, so a resource that sets none has no
// credential for it at all.
func TestProviderHeadersReachTheDestroyRequest(t *testing.T) {
	t.Run("should authenticate a destroy that declares no delete_headers", func(t *testing.T) {
		// given
		srv, tracker := newGuardedServer(t)
		providerConfig := builders.NewProviderTFBuilder().
			WithURL(srv.URL).
			WithHeaders(map[string]string{"Authorization": fixtureBearer}).
			Build()

		config := providerConfig + builders.NewResourceTFBuilder().
			WithName("guarded").
			WithMethod("POST").
			WithPath("/widgets").
			WithRequestBody(strconv.Quote(`{"name":"fixture"}`)).
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			WithIsDeleteEnabled(true).
			WithDeleteMethod("DELETE").
			WithDeletePath("/widgets/$.id").
			Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{Config: config},
				{Destroy: true, Config: config},
			},
		})

		// then
		authorizedDeletes, unauthorizedDeletes := tracker.counts(http.MethodDelete)
		if unauthorizedDeletes != 0 {
			t.Fatalf("the server rejected %d DELETE(s) as unauthenticated; provider headers must "+
				"apply to the destroy request", unauthorizedDeletes)
		}
		if authorizedDeletes != 1 {
			t.Fatalf("DELETE calls = %d, want exactly 1", authorizedDeletes)
		}
	})
}

// TestShadowedDeleteHeadersStayNonFatal pins the severity of the shadowing diagnostic. Warning it
// is: a great many existing configurations repeat the provider's credential in `delete_headers`,
// and that is a hazard rather than a defect -- it only bites once the credential is rotated. If it
// were ever raised to an error, every one of those configurations would stop planning at all.
func TestShadowedDeleteHeadersStayNonFatal(t *testing.T) {
	t.Run("should still plan, apply and destroy when delete_headers shadow the provider", func(t *testing.T) {
		// given
		srv, tracker := newGuardedServer(t)
		providerConfig := builders.NewProviderTFBuilder().
			WithURL(srv.URL).
			WithHeaders(map[string]string{"Authorization": fixtureBearer}).
			Build()

		config := providerConfig + builders.NewResourceTFBuilder().
			WithName("shadowed").
			WithMethod("POST").
			WithPath("/widgets").
			WithRequestBody(strconv.Quote(`{"name":"fixture"}`)).
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			WithIsDeleteEnabled(true).
			WithDeleteMethod("DELETE").
			WithDeletePath("/widgets/$.id").
			WithDeleteHeaders(map[string]string{"Authorization": fixtureBearer}).
			Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{Config: config},
				{Destroy: true, Config: config},
			},
		})

		// then
		authorizedDeletes, unauthorizedDeletes := tracker.counts(http.MethodDelete)
		if unauthorizedDeletes != 0 {
			t.Fatalf("the server rejected %d DELETE(s); the shadowing warning must not change "+
				"which credential is sent", unauthorizedDeletes)
		}
		if authorizedDeletes != 1 {
			t.Fatalf("DELETE calls = %d, want exactly 1", authorizedDeletes)
		}
	})
}
