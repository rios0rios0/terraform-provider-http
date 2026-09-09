package builders

import (
	"fmt"
	"sort"
)

const (
	baseProviderTF = `
		provider "http" {
		  %s
		}
	`
)

type ProviderTFBuilder struct {
	config string
}

func NewProviderTFBuilder() *ProviderTFBuilder {
	return &ProviderTFBuilder{}
}

func (b *ProviderTFBuilder) WithURL(url string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("url = \"%s\"\n", url)
	return b
}

func (b *ProviderTFBuilder) WithUsername(username string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("basic_auth = {\n  username = \"%s\"\n}\n", username)
	return b
}

func (b *ProviderTFBuilder) WithPassword(password string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("basic_auth = {\n  password = \"%s\"\n}\n", password)
	return b
}

func (b *ProviderTFBuilder) WithBasicAuth(username, password string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("basic_auth = {\n  username = \"%s\"\n  password = \"%s\"\n}\n", username, password)
	return b
}

func (b *ProviderTFBuilder) WithHeaders(headers map[string]string) *ProviderTFBuilder {
	// Sorted so the rendered configuration is identical between runs; Go randomises map order and
	// the terraform-plugin-testing harness compares configurations between steps as strings.
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)

	b.config += "headers = {\n"
	for _, name := range names {
		b.config += fmt.Sprintf("  %q = %q\n", name, headers[name])
	}
	b.config += "}\n"

	return b
}

func (b *ProviderTFBuilder) WithIgnoreTLS(ignoreTLS bool) *ProviderTFBuilder {
	b.config += fmt.Sprintf("ignore_tls = %t\n", ignoreTLS)
	return b
}

// WithRequestTimeoutMs bounds every request the provider makes to the given number of
// milliseconds.
func (b *ProviderTFBuilder) WithRequestTimeoutMs(timeoutMs int64) *ProviderTFBuilder {
	b.config += fmt.Sprintf("request_timeout_ms = %d\n", timeoutMs)
	return b
}

// WithRetry renders the provider-level `retry` block. The provider retries connection errors and
// 5xx responses (except 501) with an exponential backoff bounded by the two delays, which is what
// lets a suite that talks to a public endpoint survive a dropped connection.
func (b *ProviderTFBuilder) WithRetry(attempts, minDelayMs, maxDelayMs int64) *ProviderTFBuilder {
	b.config += fmt.Sprintf(
		"retry {\n  attempts = %d\n  min_delay_ms = %d\n  max_delay_ms = %d\n}\n",
		attempts, minDelayMs, maxDelayMs,
	)
	return b
}

func (b *ProviderTFBuilder) Build() string {
	return fmt.Sprintf(baseProviderTF, b.config)
}
