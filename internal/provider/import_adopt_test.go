package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/rios0rios0/terraform-provider-http/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiresReplaceUnlessAdopted(t *testing.T) {
	t.Parallel()

	t.Run("should require replacement when nothing is pending", func(t *testing.T) {
		t.Parallel()

		// given
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics

		// when
		result := provider.RequiresReplaceUnlessAdoptedForTest(t.Context(), "path", private, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.True(t, result, "a normal change must still replace the resource")
	})

	t.Run("should suppress replacement for an attribute waiting to be adopted", func(t *testing.T) {
		t.Parallel()

		// given
		// This is the single window between `terraform import` and the apply that follows it.
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics
		diagnostics.Append(
			provider.MarshalImportAdoptForTest(t.Context(), []string{"headers", "query_parameters"}, private)...,
		)

		// when
		result := provider.RequiresReplaceUnlessAdoptedForTest(t.Context(), "headers", private, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.False(t, result, "an adopted attribute must not destroy and recreate the resource")
	})

	t.Run("should still require replacement for an attribute outside the pending set", func(t *testing.T) {
		t.Parallel()

		// given
		// A payload always names the path, so a later change to it is a genuine change.
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics
		diagnostics.Append(provider.MarshalImportAdoptForTest(t.Context(), []string{"headers"}, private)...)

		// when
		result := provider.RequiresReplaceUnlessAdoptedForTest(t.Context(), "path", private, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.True(t, result)
	})

	t.Run("should require replacement once the adoption has been cleared", func(t *testing.T) {
		t.Parallel()

		// given
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics
		diagnostics.Append(provider.MarshalImportAdoptForTest(t.Context(), []string{"headers"}, private)...)
		diagnostics.Append(provider.ClearImportAdoptForTest(t.Context(), private)...)

		// when
		result := provider.RequiresReplaceUnlessAdoptedForTest(t.Context(), "headers", private, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.True(t, result, "adoption is a one-shot window, not a permanent exemption")
	})

	t.Run("should fall back to replacing when the private state cannot be read", func(t *testing.T) {
		t.Parallel()

		// given
		// A corrupt flag must not silently disable replacement for a resource that needs it.
		private := newPrivateStateDouble()
		private.data["import_adopt"] = []byte(`{"attributes":`)
		var diagnostics diag.Diagnostics

		// when
		result := provider.RequiresReplaceUnlessAdoptedForTest(t.Context(), "headers", private, &diagnostics)

		// then
		assert.True(t, result)
		assert.True(t, diagnostics.HasError())
	})
}
