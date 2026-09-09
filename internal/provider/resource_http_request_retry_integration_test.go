//go:build integration

package provider

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
)

// newResettingServer answers every request before the given one by dropping the TCP connection
// with a reset -- the `read: connection reset by peer` a CDN produces when it sheds a connection
// mid-request -- and serves the ones after it normally. The counter reports how many requests
// reached the server at all.
func newResettingServer(t *testing.T, succeedFrom int32) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) >= succeedFrom {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id":1,"name":"fixture"}`)

			return
		}

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("the test server must support hijacking to reset a connection")

			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijacking the connection: %v", err)

			return
		}
		// A zero linger turns the close into a reset instead of an orderly shutdown.
		if tcp, isTCP := conn.(*net.TCPConn); isTCP {
			_ = tcp.SetLinger(0)
		}
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)

	return srv, &calls
}

// retryTestResource is the resource both cases below apply: one GET against the resetting server.
func retryTestResource() string {
	return builders.NewResourceTFBuilder().
		WithName("retried").
		WithMethod("GET").
		WithPath("/posts/1").
		Build()
}

// TestProviderRetryBlockReachesTheRequest pins the mechanism the live acceptance suite leans on to
// survive a dropped connection: a `retry` block rendered into the provider configuration must be
// what the resource's requests actually run under. The unit tests cover the client built from a
// resource-level block; this is the provider-level block, end to end, through Configure.
func TestProviderRetryBlockReachesTheRequest(t *testing.T) {
	t.Run("should apply when the connection is reset and the provider block configures retries", func(t *testing.T) {
		// given: the first two requests are reset, the third is served
		srv, calls := newResettingServer(t, 3)
		config := builders.NewProviderTFBuilder().
			WithURL(srv.URL).
			WithRequestTimeoutMs(liveRequestTimeoutMs).
			// The live suite's attempts with a backoff short enough for a test.
			WithRetry(liveRetryAttempts, 1, 2).
			Build() + retryTestResource()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						// then
						resource.TestCheckResourceAttr("http_request.retried", "response_code", "200"),
						resource.TestCheckResourceAttr("http_request.retried", "response_body", `{"id":1,"name":"fixture"}`),
					),
				},
			},
		})

		// then
		if got := calls.Load(); got < 3 {
			t.Fatalf("requests reaching the server = %d, want at least 3: the provider-level retries "+
				"must have replayed the request after each reset", got)
		}
	})

	t.Run("should fail on a reset connection when the provider block configures no retries", func(t *testing.T) {
		// given: the very first request is reset and no retry block is rendered
		srv, calls := newResettingServer(t, 2)
		config := builders.NewProviderTFBuilder().
			WithURL(srv.URL).
			WithRequestTimeoutMs(liveRequestTimeoutMs).
			Build() + retryTestResource()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`Error executing request using HTTP client`),
				},
			},
		})

		// then
		if got := calls.Load(); got != 1 {
			t.Fatalf("requests reaching the server = %d, want exactly 1: without a retry block the "+
				"reset must surface as the apply error instead of being replayed", got)
		}
	})
}
