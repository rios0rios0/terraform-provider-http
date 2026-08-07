//go:build unit || integration

package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
)

// resourceWithProviderHeaders builds a resource whose provider configuration carries the given
// default headers, which is the only state this behaviour depends on.
func resourceWithProviderHeaders(headers map[string]string) *HTTPRequestResource {
	config := entities.NewConfiguration("https://example.test")
	config.Headers = headers

	return &HTTPRequestResource{internal: entities.NewInternalContext(false, config)}
}

// requestModel is the smallest model buildRequest accepts. Every field left out keeps its zero
// value, which the framework reads as null -- including `basic_auth`, so no request is authenticated
// by anything other than the headers under test.
func requestModel(headers types.Map) HTTPRequestResourceModel {
	return HTTPRequestResourceModel{
		Method:             types.StringValue("GET"),
		Path:               types.StringValue("/resource"),
		Headers:            headers,
		RequestBody:        types.StringNull(),
		IsResponseBodyJSON: types.BoolNull(),
	}
}

// resourceHeaderMap converts a plain map into the framework map a resource's `headers` holds.
func resourceHeaderMap(t *testing.T, headers map[string]string) types.Map {
	t.Helper()

	value, diags := types.MapValueFrom(context.Background(), types.StringType, headers)
	require.False(t, diags.HasError(), "the fixture headers must convert cleanly")

	return value
}

// buildTestRequest performs the build under test and fails the test if it errors, so each case is
// left holding only its own given and then.
func buildTestRequest(t *testing.T, it *HTTPRequestResource, model HTTPRequestResourceModel) *http.Request {
	t.Helper()

	request, err := it.buildRequest(context.Background(), model, "https://example.test/resource")
	require.NoError(t, err, "building the request must not fail")

	return request
}

func TestBuildRequestProviderHeaders(t *testing.T) {
	t.Parallel()

	t.Run("should send a provider header when the resource declares none", func(t *testing.T) {
		t.Parallel()

		// given
		it := resourceWithProviderHeaders(map[string]string{"Authorization": "Bearer provider-value"})
		model := requestModel(types.MapNull(types.StringType))

		// when
		request := buildTestRequest(t, it, model)

		// then
		assert.Equal(t, "Bearer provider-value", request.Header.Get("Authorization"),
			"a provider-level header must reach a request the resource did not add headers to")
	})

	t.Run("should let the resource header win when both name the same header", func(t *testing.T) {
		t.Parallel()

		// given
		it := resourceWithProviderHeaders(map[string]string{"Authorization": "Bearer provider-value"})
		model := requestModel(resourceHeaderMap(t, map[string]string{"Authorization": "Bearer resource-value"}))

		// when
		request := buildTestRequest(t, it, model)

		// then
		assert.Equal(t, "Bearer resource-value", request.Header.Get("Authorization"),
			"the resource is the more specific configuration and must override the provider")
		assert.Len(t, request.Header.Values("Authorization"), 1,
			"the override must replace the provider value, not append a second one")
	})

	t.Run("should override case-insensitively because header names are", func(t *testing.T) {
		t.Parallel()

		// given: the two sides spell the same header differently, as RFC 9110 permits
		it := resourceWithProviderHeaders(map[string]string{"authorization": "Bearer provider-value"})
		model := requestModel(resourceHeaderMap(t, map[string]string{"AUTHORIZATION": "Bearer resource-value"}))

		// when
		request := buildTestRequest(t, it, model)

		// then
		assert.Len(t, request.Header.Values("Authorization"), 1,
			"differing casing must not produce two values for one header")
		assert.Equal(t, "Bearer resource-value", request.Header.Get("Authorization"),
			"the resource must win whatever casing either side used")
	})

	t.Run("should merge per header rather than replacing the whole set", func(t *testing.T) {
		t.Parallel()

		// given
		it := resourceWithProviderHeaders(map[string]string{
			"Authorization":   "Bearer provider-value",
			"X-Provider-Only": "kept",
		})
		model := requestModel(resourceHeaderMap(t, map[string]string{"X-Resource-Only": "added"}))

		// when
		request := buildTestRequest(t, it, model)

		// then
		assert.Equal(t, "Bearer provider-value", request.Header.Get("Authorization"))
		assert.Equal(t, "kept", request.Header.Get("X-Provider-Only"),
			"declaring resource headers must not discard the provider's")
		assert.Equal(t, "added", request.Header.Get("X-Resource-Only"))
	})

	t.Run("should add nothing when the provider declares no headers", func(t *testing.T) {
		t.Parallel()

		// given
		it := resourceWithProviderHeaders(nil)
		model := requestModel(types.MapNull(types.StringType))

		// when
		request := buildTestRequest(t, it, model)

		// then
		assert.Empty(t, request.Header.Get("Authorization"),
			"an unset provider headers map must leave the request exactly as it was before")
	})

	t.Run("should keep a provider Content-Type instead of the JSON default", func(t *testing.T) {
		t.Parallel()

		// given: the JSON default only fills a Content-Type nothing else set
		it := resourceWithProviderHeaders(map[string]string{"Content-Type": "application/vnd.api+json"})
		model := requestModel(types.MapNull(types.StringType))
		model.IsResponseBodyJSON = types.BoolValue(true)

		// when
		request := buildTestRequest(t, it, model)

		// then
		assert.Equal(t, "application/vnd.api+json", request.Header.Get("Content-Type"),
			"provider headers are applied before the JSON defaults so an explicit value survives")
	})

	t.Run("should not panic when the provider has not been configured yet", func(t *testing.T) {
		t.Parallel()

		// given: Terraform sets provider data after the ConfigureProvider RPC, so nil is reachable
		it := &HTTPRequestResource{}
		model := requestModel(types.MapNull(types.StringType))

		// when
		request := buildTestRequest(t, it, model)

		// then
		assert.NotNil(t, request)
	})
}
