//go:build integration

package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/checkpoint"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/rios0rios0/terraform-provider-http/v3/test/infrastructure/builders"
)

// The variables terraform-plugin-testing reads to locate the Terraform CLI. Its own constants for
// them live in an internal package.
const (
	envTerraformPath    = "TF_ACC_TERRAFORM_PATH"
	envTerraformVersion = "TF_ACC_TERRAFORM_VERSION"
)

// How hard TestMain tries to resolve a Terraform CLI: five attempts, waiting 2s, 4s, 8s and 16s
// in between, so a lookup outage shorter than half a minute costs time rather than the run.
const (
	terraformCLIAttempts   = 5
	terraformCLIFirstDelay = 2 * time.Second
)

// TestMain resolves ONE Terraform CLI for the whole test binary and hands it to every test case
// through TF_ACC_TERRAFORM_PATH.
//
// Left to itself, terraform-plugin-testing resolves the CLI per test case: every resource.Test
// and resource.UnitTest looks for `terraform` on PATH and, where there is none -- the CI runner --
// asks checkpoint.hashicorp.com for the latest version and downloads it into a fresh directory.
// That is one lookup and one download for each of this package's test cases, and the first lookup
// that fails (a 502, a timeout) makes the helper call os.Exit(1), which takes every other test in
// the binary down with it. On a bump merge it also takes the release down, because the tag is
// only created after the tests pass.
//
// An explicit TF_ACC_TERRAFORM_PATH or TF_ACC_TERRAFORM_VERSION is left alone: the helper already
// honours both, and either one says the caller has chosen the CLI.
func TestMain(m *testing.M) {
	os.Exit(runWithTerraformCLI(m))
}

// runWithTerraformCLI is TestMain's body, separated so its deferred cleanup runs before os.Exit.
func runWithTerraformCLI(m *testing.M) int {
	if os.Getenv(envTerraformPath) != "" || os.Getenv(envTerraformVersion) != "" {
		return m.Run()
	}

	installDir, err := os.MkdirTemp("", "terraform-cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create a directory for the Terraform CLI: %v\n", err)

		return 1
	}
	defer os.RemoveAll(installDir)

	execPath, err := withRetries(
		context.Background(), terraformCLIAttempts, terraformCLIFirstDelay, time.Sleep,
		findOrInstallTerraformCLI(installDir),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot resolve a Terraform CLI for the acceptance tests: %v\n", err)

		return 1
	}

	if err := os.Setenv(envTerraformPath, execPath); err != nil {
		fmt.Fprintf(os.Stderr, "cannot hand the Terraform CLI to the tests: %v\n", err)

		return 1
	}

	return m.Run()
}

// findOrInstallTerraformCLI looks for a Terraform CLI on PATH first, then installs the latest
// release as reported by checkpoint.hashicorp.com, and then the latest one listed on
// releases.hashicorp.com.
//
// Every source gets an Ensure call of its own on purpose. Ensure only moves on from a source whose
// error is skippable, and a failed checkpoint lookup is not, so with the sources in one list a
// checkpoint outage would end the call before the releases index is ever tried.
func findOrInstallTerraformCLI(installDir string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		sources := []src.Source{
			&fs.AnyVersion{Product: &product.Terraform},
			&checkpoint.LatestVersion{Product: product.Terraform, InstallDir: installDir},
			&releases.LatestVersion{Product: product.Terraform, InstallDir: installDir},
		}

		failures := make([]error, 0, len(sources))
		for _, source := range sources {
			execPath, err := install.NewInstaller().Ensure(ctx, []src.Source{source})
			if err == nil {
				return execPath, nil
			}

			failures = append(failures, err)
		}

		return "", errors.Join(failures...)
	}
}

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
