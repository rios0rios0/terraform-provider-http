package provider

import (
	"testing"
)

func TestShadowedHeaderNames(t *testing.T) {
	t.Run("should report a delete header the provider also sets", func(t *testing.T) {
		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"Authorization"}

		// when
		shadowed := shadowedHeaderNames(providerHeaders, deleteHeaderNames)

		// then
		if len(shadowed) != 1 || shadowed[0] != "Authorization" {
			t.Fatalf("shadowed = %v, want [Authorization]", shadowed)
		}
	})

	t.Run("should match names case-insensitively, as http.Header.Set does", func(t *testing.T) {
		// given
		providerHeaders := map[string]string{"authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"AUTHORIZATION"}

		// when
		shadowed := shadowedHeaderNames(providerHeaders, deleteHeaderNames)

		// then
		if len(shadowed) != 1 {
			t.Fatalf("shadowed = %v, want one name; a differently-cased key still overwrites", shadowed)
		}
	})

	t.Run("should keep the spelling the configuration used", func(t *testing.T) {
		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"authorization"}

		// when
		shadowed := shadowedHeaderNames(providerHeaders, deleteHeaderNames)

		// then
		if len(shadowed) != 1 || shadowed[0] != "authorization" {
			t.Fatalf("shadowed = %v, want [authorization]; the diagnostic must name what was written", shadowed)
		}
	})

	t.Run("should report nothing when the maps do not overlap", func(t *testing.T) {
		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}
		deleteHeaderNames := []string{"Content-Type"}

		// when
		shadowed := shadowedHeaderNames(providerHeaders, deleteHeaderNames)

		// then
		if shadowed != nil {
			t.Fatalf("shadowed = %v, want nil; a header the provider does not set is not shadowed", shadowed)
		}
	})

	t.Run("should report every overlapping name, sorted", func(t *testing.T) {
		// given
		providerHeaders := map[string]string{
			"Authorization": "Bearer fixture-token-placeholder",
			"X-Api-Key":     "fixture-key-placeholder",
			"Accept":        "application/json",
		}
		deleteHeaderNames := []string{"X-Api-Key", "Authorization", "Content-Type"}

		// when
		shadowed := shadowedHeaderNames(providerHeaders, deleteHeaderNames)

		// then
		if len(shadowed) != 2 || shadowed[0] != "Authorization" || shadowed[1] != "X-Api-Key" {
			t.Fatalf("shadowed = %v, want [Authorization X-Api-Key] in that order", shadowed)
		}
	})

	t.Run("should report nothing when the provider sets no headers", func(t *testing.T) {
		// given
		var providerHeaders map[string]string
		deleteHeaderNames := []string{"Authorization"}

		// when
		shadowed := shadowedHeaderNames(providerHeaders, deleteHeaderNames)

		// then
		if shadowed != nil {
			t.Fatalf("shadowed = %v, want nil; there is nothing to shadow", shadowed)
		}
	})

	t.Run("should report nothing when the resource sets no delete headers", func(t *testing.T) {
		// given
		providerHeaders := map[string]string{"Authorization": "Bearer fixture-token-placeholder"}

		// when
		shadowed := shadowedHeaderNames(providerHeaders, nil)

		// then
		if shadowed != nil {
			t.Fatalf("shadowed = %v, want nil; this is the shape the warning exists to encourage", shadowed)
		}
	})
}

func TestFormatHeaderList(t *testing.T) {
	t.Run("should render one name", func(t *testing.T) {
		// given
		names := []string{"Authorization"}

		// when
		rendered := formatHeaderList(names)

		// then
		if rendered != "`Authorization`" {
			t.Fatalf("rendered = %q", rendered)
		}
	})

	t.Run("should join two names with and", func(t *testing.T) {
		// given
		names := []string{"Authorization", "X-Api-Key"}

		// when
		rendered := formatHeaderList(names)

		// then
		if rendered != "`Authorization` and `X-Api-Key`" {
			t.Fatalf("rendered = %q", rendered)
		}
	})

	t.Run("should comma-separate three names and join the last with and", func(t *testing.T) {
		// given
		names := []string{"Authorization", "Proxy-Authorization", "X-Api-Key"}

		// when
		rendered := formatHeaderList(names)

		// then
		want := "`Authorization`, `Proxy-Authorization` and `X-Api-Key`"
		if rendered != want {
			t.Fatalf("rendered = %q, want %q", rendered, want)
		}
	})

	t.Run("should render an empty list as an empty string", func(t *testing.T) {
		// given
		var names []string

		// when
		rendered := formatHeaderList(names)

		// then
		if rendered != "" {
			t.Fatalf("rendered = %q, want empty", rendered)
		}
	})
}
