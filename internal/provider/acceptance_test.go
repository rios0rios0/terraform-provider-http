//go:build integration

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
)

// The acceptance harness shared by every test in this package that drives a real Terraform binary
// through terraform-plugin-testing. Those tests sit behind the `integration` build tag because
// they need that binary -- and the ones pointed at liveEndpoint, the network as well. The unit
// tests of this package carry no build tag at all.

// testAccProtoV6ProviderFactories is barely used to create the block "terraform.required_providers"
// in the Terraform configuration.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"http": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck runs before every acceptance case. The namespace is set with os.Setenv rather
// than t.Setenv because the callers run in parallel, which t.Setenv refuses; the value is the same
// for every caller, so the order they set it in does not matter.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if err := os.Setenv("TF_ACC_PROVIDER_NAMESPACE", "rios0rios0"); err != nil {
		t.Fatalf("setting TF_ACC_PROVIDER_NAMESPACE, without which the provider cannot be resolved: %v", err)
	}
}

// liveEndpoint is the public API the live acceptance tests exercise for real.
const liveEndpoint = "https://jsonplaceholder.typicode.com"

// The hardening every provider block pointed at liveEndpoint carries. The endpoint sits behind a
// CDN that now and then resets a connection mid-request, and one such reset used to fail the whole
// run: the provider only retries when asked to. With these settings a transient error is retried
// after 1s, 2s and 4s -- at most 7s on top of a request that would otherwise have failed, and
// nothing at all on the happy path -- and a stalled connection gives up after 30s instead of
// holding the suite until `go test` itself times out.
const (
	liveRequestTimeoutMs int64 = 30000
	liveRetryAttempts    int64 = 3
	liveRetryMinDelayMs  int64 = 1000
	liveRetryMaxDelayMs  int64 = 5000
)

// liveProvider returns a provider block builder hardened for the public endpoint. The URL is left
// to the caller so a test can leave it out and exercise a resource-level `base_url` instead.
func liveProvider() *builders.ProviderTFBuilder {
	return builders.NewProviderTFBuilder().
		WithRequestTimeoutMs(liveRequestTimeoutMs).
		WithRetry(liveRetryAttempts, liveRetryMinDelayMs, liveRetryMaxDelayMs)
}

// liveProviderWithURL is liveProvider pointed at the public endpoint.
func liveProviderWithURL() *builders.ProviderTFBuilder {
	return liveProvider().WithURL(liveEndpoint)
}

// liveProviderConfigs renders the four provider blocks the apply/destroy/import cases run against:
// with and without basic auth, with and without `ignore_tls`.
func liveProviderConfigs() []string {
	return []string{
		liveProviderWithURL().WithBasicAuth("***", "***").WithIgnoreTLS(true).Build(),
		liveProviderWithURL().WithIgnoreTLS(true).Build(),
		liveProviderWithURL().WithBasicAuth("***", "***").Build(),
		liveProviderWithURL().Build(),
	}
}
