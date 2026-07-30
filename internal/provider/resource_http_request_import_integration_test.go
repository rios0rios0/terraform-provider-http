//go:build integration

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
)

// importTracker counts the calls a test server receives, so a test can prove that adopting a
// configuration after an import sends nothing at all.
type importTracker struct {
	mu     sync.Mutex
	byVerb map[string]int
	posts  int
}

func newImportServer(t *testing.T) (*httptest.Server, *importTracker) {
	t.Helper()

	tracker := &importTracker{byVerb: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()

		tracker.byVerb[r.Method]++
		if r.Method == http.MethodPost {
			tracker.posts++
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"id":%d,"name":"fixture"}`, tracker.posts+1)
	}))
	t.Cleanup(srv.Close)

	return srv, tracker
}

func (it *importTracker) calls(verb string) int {
	it.mu.Lock()
	defer it.mu.Unlock()

	return it.byVerb[verb]
}

// TestHTTPRequestResourceImportAdoption is the load-bearing test for import adoption: importing
// with an identifier that omits arguments and then planning against the full configuration must
// settle in place. A regression here reintroduces exactly the destroy-and-recreate that importing,
// rather than recreating, exists to avoid.
func TestHTTPRequestResourceImportAdoption(t *testing.T) {
	t.Run("should update in place instead of replacing when a short identifier is imported", func(t *testing.T) {
		// given
		srv, tracker := newImportServer(t)
		address := "http_request.imported"
		providerConfig := builders.NewProviderTFBuilder().WithURL(srv.URL).Build()

		// The configuration the resource is applied under, and the fuller one it must converge on
		// after the import without being replaced.
		minimal := providerConfig + builders.NewResourceTFBuilder().
			WithName("imported").
			WithMethod("GET").
			WithPath("/posts/1").
			Build()
		full := providerConfig + builders.NewResourceTFBuilder().
			WithName("imported").
			WithMethod("GET").
			WithPath("/posts/1").
			WithHeaders(map[string]string{"Accept": "application/json"}).
			WithQueryParameters(map[string]string{"expand": "true"}).
			Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				// Establish the resource.
				{Config: minimal},

				// Drop it from state without touching the remote object: `is_delete_enabled` is
				// unset, so Delete only forgets it. This reproduces the situation the feature is
				// for -- the object still exists remotely but Terraform no longer knows about it.
				{Destroy: true, Config: minimal},

				// Re-import it knowing only its path. This is the "the state was lost and the
				// resource cannot be recreated" case.
				{
					ResourceName:       address,
					ImportState:        true,
					ImportStatePersist: true,
					ImportStateId:      "/posts/1",
					ImportStateCheck: func(states []*terraform.InstanceState) error {
						// then
						if got := states[0].Attributes["method"]; got != "GET" {
							return fmt.Errorf("method = %q, want GET", got)
						}
						if got := states[0].Attributes["path"]; got != "/posts/1" {
							return fmt.Errorf("path = %q, want /posts/1", got)
						}
						if states[0].Attributes["response_body"] == "" {
							return fmt.Errorf("response_body should have been captured by the live read")
						}
						if states[0].Attributes["import_id"] == "" {
							return fmt.Errorf("import_id should have been rendered")
						}

						return nil
					},
				},

				// The first plan against the full configuration must be an in-place update.
				{
					Config: full,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
						},
					},
					Check: resource.ComposeAggregateTestCheckFunc(
						// then
						resource.TestCheckResourceAttr(address, "headers.Accept", "application/json"),
						resource.TestCheckResourceAttr(address, "query_parameters.expand", "true"),
						resource.TestCheckResourceAttrSet(address, "import_id"),
					),
				},

				// And the plan after that must be empty: adoption is a one-shot settlement.
				// PlanOnly already fails the step on a non-empty plan, and it cannot be combined
				// with PreApply plan checks.
				{Config: full, PlanOnly: true},
			},
		})

		// then
		// One GET for the apply and one for the import read. The adoption itself sent nothing,
		// and nothing was ever created or destroyed.
		if got := tracker.calls(http.MethodGet); got != 2 {
			t.Fatalf("GET calls = %d, want 2 (the apply and the import read)", got)
		}
		if got := tracker.calls(http.MethodPost); got != 0 {
			t.Fatalf("POST calls = %d, want 0", got)
		}
		if got := tracker.calls(http.MethodDelete); got != 0 {
			t.Fatalf("DELETE calls = %d, want 0 -- the resource must never be replaced", got)
		}
	})

	t.Run("should not replay an unsafe method when importing", func(t *testing.T) {
		// given
		// Replaying a POST would create a second remote object, which is the opposite of import.
		srv, tracker := newImportServer(t)
		address := "http_request.created"
		config := builders.NewProviderTFBuilder().WithURL(srv.URL).Build() +
			builders.NewResourceTFBuilder().
				WithName("created").
				WithMethod("POST").
				WithPath("/posts").
				Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{Config: config},
				{Destroy: true, Config: config},
				{
					ResourceName:       address,
					ImportState:        true,
					ImportStatePersist: true,
					ImportStateId:      "POST /posts",
					ImportStateCheck: func(states []*terraform.InstanceState) error {
						// then
						if got := states[0].Attributes["response_body"]; got != "" {
							return fmt.Errorf("response_body = %q, want empty for an unsafe method", got)
						}

						return nil
					},
				},
			},
		})

		// then
		if got := tracker.calls(http.MethodPost); got != 1 {
			t.Fatalf("POST calls = %d, want 1 -- the import must not replay the creation", got)
		}
	})

	t.Run("should read the object named by import_read_path for an unsafe method", func(t *testing.T) {
		// given
		srv, tracker := newImportServer(t)
		address := "http_request.created_with_read"
		config := builders.NewProviderTFBuilder().WithURL(srv.URL).Build() +
			builders.NewResourceTFBuilder().
				WithName("created_with_read").
				WithMethod("POST").
				WithPath("/posts").
				Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{Config: config},
				{Destroy: true, Config: config},
				{
					ResourceName:       address,
					ImportState:        true,
					ImportStatePersist: true,
					ImportStateId:      `{"method":"POST","path":"/posts","import_read_path":"/posts/7"}`,
					ImportStateCheck: func(states []*terraform.InstanceState) error {
						// then
						if states[0].Attributes["response_body"] == "" {
							return fmt.Errorf("response_body should have been captured from import_read_path")
						}

						return nil
					},
				},
			},
		})

		// then
		if got := tracker.calls(http.MethodGet); got != 1 {
			t.Fatalf("GET calls = %d, want 1 against the import_read_path", got)
		}
		if got := tracker.calls(http.MethodPost); got != 1 {
			t.Fatalf("POST calls = %d, want 1 -- only the initial apply", got)
		}
	})
}
