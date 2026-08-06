package entities

type Configuration struct {
	URL       string
	BasicAuth *BasicAuth
	// Headers are sent on every request this provider makes, before each resource's
	// own headers, so a resource naming the same header still wins. They exist for
	// credentials an API expects in a header rather than in basic auth: a bearer
	// token set here reaches requests a resource's own configuration cannot,
	// notably the read an import issues, which is built from the import identifier
	// alone and never sees the configuration. A nil or empty map adds nothing.
	Headers map[string]string
	// RequestTimeoutMs is the per-request timeout in milliseconds applied to the
	// underlying HTTP client. A value of 0 means no timeout (the client waits
	// indefinitely, which is the Go default).
	RequestTimeoutMs int64
	// Retry holds the retry configuration. A nil value means no retries.
	Retry *RetryConfig
}

type BasicAuth struct {
	Username string
	Password string
}

// RetryConfig describes how a failed HTTP request should be retried. It mirrors
// the semantics of the upstream hashicorp/http provider's `retry` block.
type RetryConfig struct {
	// Attempts is the maximum number of retries. For example, if 2 is specified,
	// the request is tried a maximum of 3 times (the initial attempt plus 2 retries).
	Attempts int64
	// MinDelayMs is the minimum delay between retries, in milliseconds.
	MinDelayMs int64
	// MaxDelayMs is the maximum delay between retries, in milliseconds.
	MaxDelayMs int64
}

func NewConfiguration(url string) *Configuration {
	return &Configuration{URL: url}
}

// HasAuthentication reports whether provider-level basic auth is usable. The nil-receiver check
// makes it safe to call on the configuration of a provider that has not been configured yet.
func (it *Configuration) HasAuthentication() bool {
	return it != nil && it.BasicAuth != nil && it.BasicAuth.Username != "" && it.BasicAuth.Password != ""
}
