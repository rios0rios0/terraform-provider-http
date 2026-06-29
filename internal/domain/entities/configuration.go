package entities

type Configuration struct {
	URL       string
	BasicAuth *BasicAuth
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

func (it *Configuration) HasAuthentication() bool {
	return it.BasicAuth != nil && it.BasicAuth.Username != "" && it.BasicAuth.Password != ""
}
