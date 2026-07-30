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

// driftServer serves a value that a test can change behind Terraform's back, and can be told to
// report the object as gone.
type driftServer struct {
	mu      sync.Mutex
	name    string
	missing bool
	gets    int
}

func newDriftServer(t *testing.T) (*httptest.Server, *driftServer) {
	t.Helper()

	state := &driftServer{name: "original"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		if r.Method == http.MethodGet {
			state.gets++
		}

		if state.missing {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"gone"}`))

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"id":1,"name":%q}`, state.name)
	}))
	t.Cleanup(srv.Close)

	return srv, state
}

func (d *driftServer) rename(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.name = name
}

func (d *driftServer) vanish() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.missing = true
}

func (d *driftServer) reads() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.gets
}

func TestHTTPRequestResourceRefresh(t *testing.T) {
	t.Run("should leave the captured response alone when refresh is disabled", func(t *testing.T) {
		// given
		// This is the default, and it is what every existing configuration relies on.
		srv, remote := newDriftServer(t)
		address := "http_request.static"
		config := builders.NewProviderTFBuilder().WithURL(srv.URL).Build() +
			builders.NewResourceTFBuilder().
				WithName("static").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.TestCheckResourceAttr(
						address, "response_body_json.name", "original",
					),
				},
				{
					PreConfig: func() { remote.rename("changed-behind-terraform") },
					Config:    config,
					PlanOnly:  true,
				},
			},
		})

		// then
		// The plan stayed empty: nothing re-read the endpoint.
	})

	t.Run("should detect drift when refresh is enabled", func(t *testing.T) {
		// given
		srv, remote := newDriftServer(t)
		address := "http_request.watched"
		config := builders.NewProviderTFBuilder().WithURL(srv.URL).Build() +
			builders.NewResourceTFBuilder().
				WithName("watched").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				WithIsRefreshEnabled(true).
				Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.TestCheckResourceAttr(
						address, "response_body_json.name", "original",
					),
				},
				{
					PreConfig: func() { remote.rename("changed-behind-terraform") },
					Config:    config,
					Check: resource.TestCheckResourceAttr(
						// then
						address, "response_body_json.name", "changed-behind-terraform",
					),
				},
			},
		})
	})

	t.Run("should plan a create when refresh finds the resource gone", func(t *testing.T) {
		// given
		srv, remote := newDriftServer(t)
		config := builders.NewProviderTFBuilder().WithURL(srv.URL).Build() +
			builders.NewResourceTFBuilder().
				WithName("vanishing").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsRefreshEnabled(true).
				Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{Config: config},
				{
					PreConfig:    func() { remote.vanish() },
					RefreshState: true,
					// then
					// The refresh drops it from state, so the following plan is non-empty: the
					// resource is proposed for creation rather than left pointing at something
					// that no longer exists.
					ExpectNonEmptyPlan: true,
				},
			},
		})

		// then
		if remote.reads() == 0 {
			t.Fatal("the refresh should have read the endpoint")
		}
	})
}
