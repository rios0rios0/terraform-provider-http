//go:build unit || integration

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
)

func retryObject(attempts, minDelayMs, maxDelayMs types.Int64) types.Object {
	return types.ObjectValueMust(retryObjectAttrTypes(), map[string]attr.Value{
		attrAttempts:   attempts,
		attrMinDelayMs: minDelayMs,
		attrMaxDelayMs: maxDelayMs,
	})
}

func TestRetryConfigFromObject(t *testing.T) {
	t.Parallel()

	t.Run("should return nil when the object is null", func(t *testing.T) {
		t.Parallel()

		// given
		obj := types.ObjectNull(retryObjectAttrTypes())

		// when
		cfg := retryConfigFromObject(obj)

		// then
		assert.Nil(t, cfg, "a null retry block means no retries")
	})

	t.Run("should apply default delays when only attempts is set", func(t *testing.T) {
		t.Parallel()

		// given
		obj := retryObject(types.Int64Value(5), types.Int64Null(), types.Int64Null())

		// when
		cfg := retryConfigFromObject(obj)

		// then
		require.NotNil(t, cfg)
		assert.Equal(t, int64(5), cfg.Attempts, "attempts is taken from the block")
		assert.Equal(t, defaultRetryMinDelayMs, cfg.MinDelayMs, "min delay defaults to 1000ms")
		assert.Equal(t, defaultRetryMaxDelayMs, cfg.MaxDelayMs, "max delay defaults to 30000ms")
	})

	t.Run("should honor explicit delays", func(t *testing.T) {
		t.Parallel()

		// given
		obj := retryObject(types.Int64Value(3), types.Int64Value(250), types.Int64Value(2000))

		// when
		cfg := retryConfigFromObject(obj)

		// then
		require.NotNil(t, cfg)
		assert.Equal(t, int64(3), cfg.Attempts)
		assert.Equal(t, int64(250), cfg.MinDelayMs)
		assert.Equal(t, int64(2000), cfg.MaxDelayMs)
	})

	t.Run("should clamp a max delay that is below the min delay", func(t *testing.T) {
		t.Parallel()

		// given
		obj := retryObject(types.Int64Value(2), types.Int64Value(5000), types.Int64Value(1000))

		// when
		cfg := retryConfigFromObject(obj)

		// then
		require.NotNil(t, cfg)
		assert.Equal(t, int64(5000), cfg.MinDelayMs)
		assert.Equal(t, int64(5000), cfg.MaxDelayMs, "max is raised to min when configured lower")
	})
}

func TestGetHTTPClientRetry(t *testing.T) {
	t.Parallel()

	t.Run("should retry on 5xx and eventually succeed", func(t *testing.T) {
		t.Parallel()

		// given: the endpoint fails twice with 503 before returning 200
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		it := &HTTPRequestResource{}
		model := HTTPRequestResourceModel{
			IgnoreTLS:        types.BoolNull(),
			RequestTimeoutMs: types.Int64Null(),
			Retry:            retryObject(types.Int64Value(5), types.Int64Value(1), types.Int64Value(2)),
		}
		client := it.getHTTPClient(context.Background(), model)

		// when
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)

		// then
		require.NoError(t, err, "the request should succeed after the transient failures are retried")
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.GreaterOrEqual(t, calls.Load(), int32(3), "the endpoint should have been retried")
	})

	t.Run("should not retry when no retry block is configured", func(t *testing.T) {
		t.Parallel()

		// given: the endpoint always returns 503
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		it := &HTTPRequestResource{}
		model := HTTPRequestResourceModel{
			IgnoreTLS:        types.BoolNull(),
			RequestTimeoutMs: types.Int64Null(),
			Retry:            types.ObjectNull(retryObjectAttrTypes()),
		}
		client := it.getHTTPClient(context.Background(), model)

		// when
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)

		// then
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, int32(1), calls.Load(),
			"the endpoint should be called exactly once without retries")
	})
}

func TestGetHTTPClientTimeout(t *testing.T) {
	t.Parallel()

	t.Run("should fail fast when the response exceeds the timeout", func(t *testing.T) {
		t.Parallel()

		// given: the endpoint is slower than the configured timeout
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		it := &HTTPRequestResource{}
		model := HTTPRequestResourceModel{
			IgnoreTLS:        types.BoolNull(),
			RequestTimeoutMs: types.Int64Value(50),
			Retry:            types.ObjectNull(retryObjectAttrTypes()),
		}
		client := it.getHTTPClient(context.Background(), model)

		// when
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		start := time.Now()
		_, err = client.Do(req)

		// then
		require.Error(t, err, "the request must not hang; it should time out")
		assert.Less(t, time.Since(start), 200*time.Millisecond,
			"the client should give up well before the server responds")
	})

	t.Run("should not set a timeout when request_timeout_ms is unset", func(t *testing.T) {
		t.Parallel()

		// given
		it := &HTTPRequestResource{}
		model := HTTPRequestResourceModel{
			IgnoreTLS:        types.BoolNull(),
			RequestTimeoutMs: types.Int64Null(),
			Retry:            types.ObjectNull(retryObjectAttrTypes()),
		}

		// when
		client := it.getHTTPClient(context.Background(), model)

		// then
		assert.Equal(t, time.Duration(0), client.Timeout,
			"an unset timeout preserves the historical no-timeout behavior")
	})
}

func TestGetHTTPClientTransportReuse(t *testing.T) {
	t.Parallel()

	t.Run("should reuse the provider transport when ignore_tls is set at the provider level", func(t *testing.T) {
		t.Parallel()

		// given: a provider configured with ignore_tls owns an insecure transport
		internal := entities.NewInternalContext(true, entities.NewConfiguration(""))
		providerTransport := internal.Client.Transport
		require.NotNil(t, providerTransport, "the provider should own a transport when ignore_tls is enabled")
		it := &HTTPRequestResource{internal: internal}
		model := HTTPRequestResourceModel{
			IgnoreTLS:        types.BoolNull(),
			RequestTimeoutMs: types.Int64Null(),
			Retry:            types.ObjectNull(retryObjectAttrTypes()),
		}

		// when
		client := it.getHTTPClient(context.Background(), model)

		// then
		assert.Same(t, providerTransport, client.Transport,
			"the provider transport must be reused so the connection pool is shared across requests")
	})

	t.Run("should allocate a fresh transport when a resource overrides ignore_tls from false to true", func(t *testing.T) {
		t.Parallel()

		// given: a provider that verifies TLS has no insecure transport to reuse
		internal := entities.NewInternalContext(false, entities.NewConfiguration(""))
		it := &HTTPRequestResource{internal: internal}
		model := HTTPRequestResourceModel{
			IgnoreTLS:        types.BoolValue(true),
			RequestTimeoutMs: types.Int64Null(),
			Retry:            types.ObjectNull(retryObjectAttrTypes()),
		}

		// when
		client := it.getHTTPClient(context.Background(), model)

		// then
		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok, "a fresh *http.Transport must be created for the resource-level override")
		require.NotNil(t, transport.TLSClientConfig)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify,
			"the new transport must skip verification per the resource override")
	})
}
