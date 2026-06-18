//go:build integration

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
)

// reissueTracker is a tiny in-memory HTTP endpoint used to detect whether the
// provider re-issued the outgoing request. Every non-DELETE call returns a fresh
// incrementing id, so a re-issue is observable as a changed response_body_id.
// DELETE calls are recorded so destroy behaviour can be asserted.
type reissueTracker struct {
	mu          sync.Mutex
	createCount int
	deletePaths []string
}

func newReissueServer(t *testing.T) (*httptest.Server, *reissueTracker) {
	t.Helper()

	tracker := &reissueTracker{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()

		if r.Method == http.MethodDelete {
			tracker.deletePaths = append(tracker.deletePaths, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}

		tracker.createCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"id":"%d"}`, tracker.createCount)
	}))
	t.Cleanup(srv.Close)

	return srv, tracker
}

func (rt *reissueTracker) creates() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.createCount
}

func (rt *reissueTracker) deleted(path string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for _, p := range rt.deletePaths {
		if p == path {
			return true
		}
	}
	return false
}

// TestHTTPRequestResource_ReissueRegression guards the "Provider produced inconsistent
// result after apply" bug: an in-place update that changes only a client-side attribute
// (for example tolerated_status_codes) must not re-issue the request, while a genuine
// request change still re-issues and stays consistent.
func TestHTTPRequestResource_ReissueRegression(t *testing.T) {
	t.Run("should not re-issue or fail apply when only a client-side attribute changes", func(t *testing.T) {
		// given
		srv, tracker := newReissueServer(t)
		providerConfig := builders.NewProviderTFBuilder().WithURL(srv.URL).Build()

		create := providerConfig + builders.NewResourceTFBuilder().
			WithName("reissue").
			WithMethod("GET").
			WithPath("/resource").
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			Build()

		// Adding tolerated_status_codes is a client-side-only change. Before the fix this
		// re-issued the request (changing the captured id) and could fail apply with
		// "Provider produced inconsistent result after apply".
		addTolerated := providerConfig + builders.NewResourceTFBuilder().
			WithName("reissue").
			WithMethod("GET").
			WithPath("/resource").
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			WithToleratedStatusCodes([]int{404}).
			Build()

		// when / then
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: create,
					Check:  resource.TestCheckResourceAttr("http_request.reissue", "response_body_id", "1"),
				},
				{
					Config: addTolerated,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.reissue", "response_body_id", "1"),
						resource.TestCheckTypeSetElemAttr("http_request.reissue", "tolerated_status_codes.*", "404"),
					),
				},
			},
		})

		// then
		if got := tracker.creates(); got != 1 {
			t.Fatalf("expected exactly 1 outgoing request (no re-issue), got %d", got)
		}
	})

	t.Run("should re-issue and stay consistent when a request attribute changes", func(t *testing.T) {
		// given
		srv, tracker := newReissueServer(t)
		providerConfig := builders.NewProviderTFBuilder().WithURL(srv.URL).Build()

		create := providerConfig + builders.NewResourceTFBuilder().
			WithName("reissue_real").
			WithMethod("GET").
			WithPath("/resource").
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			Build()

		// Adding a query parameter changes the outgoing request, so the provider must
		// re-issue it and accept the freshly captured response.
		changeRequest := providerConfig + builders.NewResourceTFBuilder().
			WithName("reissue_real").
			WithMethod("GET").
			WithPath("/resource").
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			WithQueryParameters(map[string]string{"x": "1"}).
			Build()

		// when / then
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: create,
					Check:  resource.TestCheckResourceAttr("http_request.reissue_real", "response_body_id", "1"),
				},
				{
					Config: changeRequest,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.reissue_real", "response_body_id", "2"),
						resource.TestCheckResourceAttr("http_request.reissue_real", "query_parameters.x", "1"),
					),
				},
			},
		})

		// then
		if got := tracker.creates(); got != 2 {
			t.Fatalf("expected 2 outgoing requests (initial + re-issue), got %d", got)
		}
	})

	t.Run("should refresh delete controls into private state during a client-side-only update", func(t *testing.T) {
		// given -- create with deletion disabled, so private state initially carries
		// is_delete_enabled = false.
		srv, tracker := newReissueServer(t)
		providerConfig := builders.NewProviderTFBuilder().WithURL(srv.URL).Build()

		create := providerConfig + builders.NewResourceTFBuilder().
			WithName("reissue_delete").
			WithMethod("GET").
			WithPath("/resource").
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			Build()

		// A client-side-only change (tolerated_status_codes) drives an in-place Update while
		// it simultaneously enables deletion. The write-only delete controls must be refreshed
		// from config into private state during the short-circuit, or the later Destroy would
		// skip the DELETE.
		enableDelete := providerConfig + builders.NewResourceTFBuilder().
			WithName("reissue_delete").
			WithMethod("GET").
			WithPath("/resource").
			WithIsResponseBodyJSON(true).
			WithResponseBodyIDFilter("$.id").
			WithToleratedStatusCodes([]int{404}).
			WithIsDeleteEnabled(true).
			WithDeletePath("/resource/$.id").
			Build()

		// when / then
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: create,
					Check:  resource.TestCheckResourceAttr("http_request.reissue_delete", "response_body_id", "1"),
				},
				{
					Config: enableDelete,
					Check:  resource.TestCheckResourceAttr("http_request.reissue_delete", "response_body_id", "1"),
				},
			},
		})

		// then -- the test case auto-destroys at the end; the DELETE must have used the
		// resolved path, proving the delete controls were refreshed from config.
		if got := tracker.creates(); got != 1 {
			t.Fatalf("expected exactly 1 outgoing create request (no re-issue), got %d", got)
		}
		if !tracker.deleted("/resource/1") {
			t.Fatalf("expected a DELETE to /resource/1 after destroy, got %v", tracker.deletePaths)
		}
	})
}
