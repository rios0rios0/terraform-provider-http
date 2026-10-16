//go:build integration

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/rios0rios0/terraform-provider-http/v3/test/infrastructure/builders"
)

// keyServer mints a key on POST /keys, serves it back on GET /keys/7 and deletes it on
// DELETE /keys/7, counting every call so a test can prove a request was or was not re-sent.
type keyServer struct {
	mu      sync.Mutex
	byRoute map[string]int
}

func newKeyServer(t *testing.T) (*httptest.Server, *keyServer) {
	t.Helper()

	state := &keyServer{byRoute: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		route := r.Method + " " + r.URL.Path
		state.byRoute[route]++

		switch route {
		case "POST /keys", "GET /keys/7":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, sensitiveFixtureBody)
		case "DELETE /keys/7":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	return srv, state
}

func (k *keyServer) calls(route string) int {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.byRoute[route]
}

// keyResourceConfig renders a resource that mints a key; extra lines are applied on top.
func keyResourceConfig(srv *httptest.Server, extra func(*builders.ResourceTFBuilder)) string {
	resourceBuilder := builders.NewResourceTFBuilder().
		WithName("key").
		WithMethod("POST").
		WithPath("/keys").
		WithIsResponseBodyJSON(true).
		WithResponseBodyIDFilter("$.id")
	extra(resourceBuilder)

	return builders.NewProviderTFBuilder().WithURL(srv.URL).Build() + resourceBuilder.Build()
}

// capturedInPlainPair checks the response sits only in `response_body` / `response_body_json`.
func capturedInPlainPair(address string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(address, tfjsonpath.New(attrResponseBodyJSON).AtMapKey("api_key"),
			knownvalue.StringExact(sensitiveFixtureKey)),
		statecheck.ExpectKnownValue(address, tfjsonpath.New(attrSensitiveResponseBody), knownvalue.Null()),
		statecheck.ExpectKnownValue(address, tfjsonpath.New(attrSensitiveResponseBodyJSON), knownvalue.Null()),
	}
}

// capturedInSensitivePair checks the response sits only in the sensitive pair.
func capturedInSensitivePair(address string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(address, tfjsonpath.New(attrSensitiveResponseBodyJSON).AtMapKey("api_key"),
			knownvalue.StringExact(sensitiveFixtureKey)),
		statecheck.ExpectSensitiveValue(address, tfjsonpath.New(attrSensitiveResponseBody)),
		statecheck.ExpectSensitiveValue(address, tfjsonpath.New(attrSensitiveResponseBodyJSON)),
		statecheck.ExpectKnownValue(address, tfjsonpath.New(attrResponseBody), knownvalue.Null()),
		statecheck.ExpectKnownValue(address, tfjsonpath.New(attrResponseBodyJSON), knownvalue.Null()),
	}
}

func TestHTTPRequestResourceSensitiveResponse(t *testing.T) {
	t.Run("should capture the response only into the sensitive attributes when the flag is set", func(t *testing.T) {
		// given
		srv, remote := newKeyServer(t)
		address := "http_request.key"
		sensitiveKey := func(queryParameters map[string]string) string {
			return keyResourceConfig(srv, func(b *builders.ResourceTFBuilder) {
				b.WithIsResponseBodySensitive(true).
					WithIsDeleteEnabled(true).
					WithDeletePath("/keys/$.id").
					WithQueryParameters(queryParameters)
			})
		}
		// A create, and the create half of a replacement, plan only the pair that will be filled.
		plansOnlySensitivePair := []plancheck.PlanCheck{
			plancheck.ExpectUnknownValue(address, tfjsonpath.New(attrSensitiveResponseBody)),
			plancheck.ExpectKnownValue(address, tfjsonpath.New(attrResponseBody), knownvalue.Null()),
			plancheck.ExpectKnownValue(address, tfjsonpath.New(attrResponseBodyJSON), knownvalue.Null()),
		}

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			CheckDestroy: func(*terraform.State) error {
				// One for the replacement, one for the final destroy.
				if calls := remote.calls("DELETE /keys/7"); calls != 2 {
					return fmt.Errorf("/keys/7 should have been deleted twice, got %d", calls)
				}

				return nil
			},
			Steps: []resource.TestStep{
				{
					Config: sensitiveKey(map[string]string{"generation": "1"}),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: append([]plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionCreate),
						}, plansOnlySensitivePair...),
					},
					// then
					ConfigStateChecks: append(capturedInSensitivePair(address),
						statecheck.ExpectKnownValue(address, tfjsonpath.New(attrResponseBodyID),
							knownvalue.StringExact("7")),
						statecheck.ExpectKnownValue(address, tfjsonpath.New(attrDeleteResolvedPath),
							knownvalue.StringExact("/keys/7")),
					),
				},
				{
					Config: sensitiveKey(map[string]string{"generation": "2"}),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: append([]plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionReplace),
						}, plansOnlySensitivePair...),
					},
					ConfigStateChecks: capturedInSensitivePair(address),
				},
			},
		})
	})

	t.Run("should move the captured response between the pairs without re-sending the request", func(t *testing.T) {
		// given
		srv, remote := newKeyServer(t)
		address := "http_request.key"
		plain := keyResourceConfig(srv, func(*builders.ResourceTFBuilder) {})
		sensitive := keyResourceConfig(srv, func(b *builders.ResourceTFBuilder) { b.WithIsResponseBodySensitive(true) })
		explicitlyPlain := keyResourceConfig(srv, func(b *builders.ResourceTFBuilder) { b.WithIsResponseBodySensitive(false) })

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: plain,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							// A resource that never asked for the sensitive pair does not plan it.
							plancheck.ExpectKnownValue(address, tfjsonpath.New(attrSensitiveResponseBody), knownvalue.Null()),
							plancheck.ExpectKnownValue(address, tfjsonpath.New(attrSensitiveResponseBodyJSON), knownvalue.Null()),
							plancheck.ExpectUnknownValue(address, tfjsonpath.New(attrResponseBody)),
						},
					},
					ConfigStateChecks: capturedInPlainPair(address),
				},
				{
					Config: sensitive,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
							plancheck.ExpectSensitiveValue(address, tfjsonpath.New(attrSensitiveResponseBody)),
							plancheck.ExpectKnownValue(address, tfjsonpath.New(attrResponseBody), knownvalue.Null()),
						},
					},
					ConfigStateChecks: capturedInSensitivePair(address),
				},
				{
					Config: explicitlyPlain,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: capturedInPlainPair(address),
				},
			},
		})

		// then
		// Re-sending would have minted a new key, rotating the very secret being protected.
		if calls := remote.calls("POST /keys"); calls != 1 {
			t.Fatalf("the key should have been minted exactly once, got %d", calls)
		}
	})

	t.Run("should settle a flag that stays unknown until apply", func(t *testing.T) {
		// given
		// `terraform_data.output` is unknown while the instance is being replaced, so the plan
		// cannot know which pair will hold the response.
		srv, remote := newKeyServer(t)
		address := "http_request.key"
		withFlag := func(value bool) string {
			return fmt.Sprintf("resource \"terraform_data\" \"flag\" {\n  input = %t\n}\n", value) +
				keyResourceConfig(srv, func(b *builders.ResourceTFBuilder) {
					b.WithIsResponseBodySensitiveExpression("terraform_data.flag.output")
				})
		}

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:            withFlag(false),
					ConfigStateChecks: capturedInPlainPair(address),
				},
				{
					Config: withFlag(true),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(address, tfjsonpath.New(attrSensitiveResponseBody)),
						},
					},
					ConfigStateChecks: capturedInSensitivePair(address),
				},
			},
		})

		// then
		if calls := remote.calls("POST /keys"); calls != 1 {
			t.Fatalf("the key should have been minted exactly once, got %d", calls)
		}
	})

	t.Run("should refresh through a path resolved from the sensitive response", func(t *testing.T) {
		// given
		srv, remote := newKeyServer(t)
		address := "http_request.key"
		config := keyResourceConfig(srv, func(b *builders.ResourceTFBuilder) {
			b.WithIsResponseBodySensitive(true).
				WithIsRefreshEnabled(true).
				WithRefreshPath("/keys/$.id")
		})

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:            config,
					ConfigStateChecks: capturedInSensitivePair(address),
				},
				{
					// A refresh that resolved the wrong path would read a 404, drop the resource
					// and leave a create in this plan.
					Config:   config,
					PlanOnly: true,
				},
			},
		})

		// then
		if remote.calls("GET /keys/7") == 0 {
			t.Fatal("the refresh should have read /keys/7")
		}
	})
}
