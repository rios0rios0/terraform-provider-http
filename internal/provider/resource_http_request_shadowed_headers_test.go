package provider_test

import (
	"testing"

	"github.com/rios0rios0/terraform-provider-http/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShadowedHeaderNames(t *testing.T) {
	t.Parallel()

	t.Run("should report a delete header the provider also sets", func(t *testing.T) {
		t.Parallel()

		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"Authorization"}

		// when
		shadowed := provider.ShadowedHeaderNamesForTest(providerHeaders, deleteHeaderNames)

		// then
		require.Equal(t, []string{"Authorization"}, shadowed)
	})

	t.Run("should match names case-insensitively, as http.Header.Set does", func(t *testing.T) {
		t.Parallel()

		// given
		providerHeaders := map[string]string{"authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"AUTHORIZATION"}

		// when
		shadowed := provider.ShadowedHeaderNamesForTest(providerHeaders, deleteHeaderNames)

		// then
		require.Len(t, shadowed, 1, "a differently-cased key still overwrites the provider's")
	})

	t.Run("should keep the spelling the configuration used", func(t *testing.T) {
		t.Parallel()

		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"authorization"}

		// when
		shadowed := provider.ShadowedHeaderNamesForTest(providerHeaders, deleteHeaderNames)

		// then
		require.Equal(t, []string{"authorization"}, shadowed,
			"the diagnostic must name the header the way it was written")
	})

	t.Run("should report nothing when the maps do not overlap", func(t *testing.T) {
		t.Parallel()

		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"Content-Type"}

		// when
		shadowed := provider.ShadowedHeaderNamesForTest(providerHeaders, deleteHeaderNames)

		// then
		assert.Nil(t, shadowed, "a header the provider does not set is not shadowed")
	})

	t.Run("should report every overlapping name, sorted", func(t *testing.T) {
		t.Parallel()

		// given
		providerHeaders := map[string]string{
			"Authorization": "Bearer fixture-token-placeholder",
			"X-Api-Key":     "fixture-key-placeholder",
			"Accept":        "application/json",
		}
		deleteHeaderNames := []string{"X-Api-Key", "Authorization", "Content-Type"}

		// when
		shadowed := provider.ShadowedHeaderNamesForTest(providerHeaders, deleteHeaderNames)

		// then
		require.Equal(t, []string{"Authorization", "X-Api-Key"}, shadowed)
	})

	t.Run("should report nothing when the provider sets no headers", func(t *testing.T) {
		t.Parallel()

		// given
		var providerHeaders map[string]string
		deleteHeaderNames := []string{"Authorization"}

		// when
		shadowed := provider.ShadowedHeaderNamesForTest(providerHeaders, deleteHeaderNames)

		// then
		assert.Nil(t, shadowed, "there is nothing to shadow")
	})

	t.Run("should report nothing when the resource sets no delete headers", func(t *testing.T) {
		t.Parallel()

		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}

		// when
		shadowed := provider.ShadowedHeaderNamesForTest(providerHeaders, nil)

		// then
		assert.Nil(t, shadowed, "this is the shape the warning exists to encourage")
	})
}

func TestFormatHeaderList(t *testing.T) {
	t.Parallel()

	t.Run("should render one name", func(t *testing.T) {
		t.Parallel()

		// given
		names := []string{"Authorization"}

		// when
		rendered := provider.FormatHeaderListForTest(names)

		// then
		require.Equal(t, "`Authorization`", rendered)
	})

	t.Run("should join two names with and", func(t *testing.T) {
		t.Parallel()

		// given
		names := []string{"Authorization", "X-Api-Key"}

		// when
		rendered := provider.FormatHeaderListForTest(names)

		// then
		require.Equal(t, "`Authorization` and `X-Api-Key`", rendered)
	})

	t.Run("should comma-separate three names and join the last with and", func(t *testing.T) {
		t.Parallel()

		// given
		names := []string{"Authorization", "Proxy-Authorization", "X-Api-Key"}

		// when
		rendered := provider.FormatHeaderListForTest(names)

		// then
		require.Equal(t, "`Authorization`, `Proxy-Authorization` and `X-Api-Key`", rendered)
	})

	t.Run("should render an empty list as an empty string", func(t *testing.T) {
		t.Parallel()

		// given
		var names []string

		// when
		rendered := provider.FormatHeaderListForTest(names)

		// then
		require.Empty(t, rendered)
	})
}
