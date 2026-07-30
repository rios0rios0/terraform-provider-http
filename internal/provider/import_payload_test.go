package provider_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rios0rios0/terraform-provider-http/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeImportIDForms(t *testing.T) {
	t.Parallel()

	t.Run("should decode a bare path and default the method to GET", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics

		// when
		model, specified := provider.DecodeImportIDForTest("/posts/1", &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.Equal(t, "GET", model.Method.ValueString())
		assert.Equal(t, "/posts/1", model.Path.ValueString())
		assert.Equal(t, map[string]struct{}{"path": {}}, specified)
	})

	t.Run("should decode the method and path shorthand", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics

		// when
		model, specified := provider.DecodeImportIDForTest("post /posts", &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.Equal(t, "POST", model.Method.ValueString())
		assert.Equal(t, "/posts", model.Path.ValueString())
		assert.Equal(t, map[string]struct{}{"method": {}, "path": {}}, specified)
	})

	t.Run("should decode a raw JSON object", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		rawID := `{"method":"GET","path":"/posts/1","headers":{"Accept":"application/json"}}`

		// when
		model, specified := provider.DecodeImportIDForTest(rawID, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.Equal(t, "/posts/1", model.Path.ValueString())
		assert.Len(t, model.Headers.Elements(), 1)
		assert.Contains(t, specified, "headers")
	})

	t.Run("should decode a payload read from a file", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		name := filepath.Join(t.TempDir(), "payload.json")
		require.NoError(t, os.WriteFile(name, []byte(`{"method":"PUT","path":"/posts/9"}`), 0o600))

		// when
		model, _ := provider.DecodeImportIDForTest("@"+name, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.Equal(t, "PUT", model.Method.ValueString())
		assert.Equal(t, "/posts/9", model.Path.ValueString())
	})

	t.Run("should decode a bare base64 payload", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		rawID := base64.RawURLEncoding.EncodeToString([]byte(`{"method":"GET","path":"/posts/1"}`))

		// when
		model, _ := provider.DecodeImportIDForTest(rawID, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.Equal(t, "/posts/1", model.Path.ValueString())
	})

	t.Run("should decode the legacy identifier and base64 pair", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		encoded := base64.StdEncoding.EncodeToString([]byte(`{"method":"GET","path":"/posts/1"}`))

		// when
		model, _ := provider.DecodeImportIDForTest("unique/"+encoded, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.Equal(t, "unique", model.ID.ValueString())
		assert.Equal(t, "/posts/1", model.Path.ValueString())
	})

	t.Run("should decode a legacy payload whose base64 contains a slash", func(t *testing.T) {
		t.Parallel()

		// given
		// The standard base64 alphabet contains "/", so this payload used to be split into three
		// parts and rejected outright. A query string in the path is the usual trigger.
		var diagnostics diag.Diagnostics
		encoded := base64.StdEncoding.EncodeToString([]byte(`{"x":"","method":"GET","path":"/api/items?q=1"}`))
		require.Contains(t, encoded, "/", "the fixture must exercise a slash inside the payload")

		// when
		model, _ := provider.DecodeImportIDForTest("unique/"+encoded, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.Equal(t, "/api/items?q=1", model.Path.ValueString())
	})

	t.Run("should reject an identifier that matches no accepted form", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics

		// when
		model, _ := provider.DecodeImportIDForTest("posts/1", &diagnostics)

		// then
		require.True(t, diagnostics.HasError())
		assert.Nil(t, model)
		assert.Contains(t, diagnostics.Errors()[0].Detail(), "Accepted forms:")
	})
}

func TestDecodeImportIDNullCorrectness(t *testing.T) {
	t.Parallel()

	t.Run("should leave omitted optional arguments null so no replacement is planned", func(t *testing.T) {
		t.Parallel()

		// given
		// A concrete false for is_response_body_json would diff against a configuration that
		// leaves the argument out, and that attribute forces replacement.
		var diagnostics diag.Diagnostics

		// when
		model, _ := provider.DecodeImportIDForTest(`{"method":"GET","path":"/posts/1"}`, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.True(t, model.IsResponseBodyJSON.IsNull(), "is_response_body_json must stay null")
		assert.True(t, model.IgnoreTLS.IsNull(), "ignore_tls must stay null")
		assert.True(t, model.ResponseCode.IsNull(), "response_code must stay null")
		assert.True(t, model.Headers.IsNull(), "headers must stay null")
		assert.True(t, model.RequestBody.IsNull(), "request_body must stay null")
	})

	t.Run("should keep an explicit false distinguishable from an omitted argument", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics

		// when
		model, _ := provider.DecodeImportIDForTest(
			`{"method":"GET","path":"/posts/1","is_response_body_json":false,"ignore_tls":false}`,
			&diagnostics,
		)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.False(t, model.IsResponseBodyJSON.IsNull())
		assert.False(t, model.IsResponseBodyJSON.ValueBool())
		assert.False(t, model.IgnoreTLS.IsNull())
		assert.False(t, model.IgnoreTLS.ValueBool())
	})

	t.Run("should produce typed nulls for the nested retry and basic_auth values", func(t *testing.T) {
		t.Parallel()

		// given
		// An untyped null makes the very next plan fail with a value conversion error.
		var diagnostics diag.Diagnostics

		// when
		model, _ := provider.DecodeImportIDForTest(`{"method":"GET","path":"/posts/1"}`, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, model)
		assert.True(t, model.Retry.IsNull())
		assert.NotNil(t, model.Retry.AttributeTypes(t.Context()))
		assert.True(t, model.BasicAuth.IsNull())
		assert.Len(t, model.BasicAuth.AttributeTypes(t.Context()), 2)
	})
}

func TestBuildImportID(t *testing.T) {
	t.Parallel()

	t.Run("should round-trip through the decoder", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		original, _ := provider.DecodeImportIDForTest(
			`{"method":"POST","path":"/posts","request_body":"{}","is_response_body_json":true,`+
				`"response_body_id_filter":"$.id","response_body":"{\"id\":1}","response_code":201}`,
			&diagnostics,
		)
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		original.ID = types.StringValue("fixture-id")

		// when
		importID := provider.BuildImportIDForTest(t.Context(), *original, &diagnostics)
		decoded, _ := provider.DecodeImportIDForTest(importID.ValueString(), &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		require.NotNil(t, decoded)
		assert.Equal(t, "fixture-id", decoded.ID.ValueString())
		assert.Equal(t, original.Method, decoded.Method)
		assert.Equal(t, original.Path, decoded.Path)
		assert.Equal(t, original.RequestBody, decoded.RequestBody)
		assert.Equal(t, original.ResponseBodyIDFilter, decoded.ResponseBodyIDFilter)
		assert.Equal(t, original.ResponseCode, decoded.ResponseCode)
	})

	t.Run("should never encode the basic_auth credentials", func(t *testing.T) {
		t.Parallel()

		// given
		// The identifier lives in plain state and is meant to be pasted into shells and CI logs.
		var diagnostics diag.Diagnostics
		model, _ := provider.DecodeImportIDForTest(
			`{"method":"GET","path":"/posts/1","basic_auth":{"username":"fixture-user",`+
				`"password":"fixture-password-placeholder"}}`,
			&diagnostics,
		)
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		model.ID = types.StringValue("fixture-id")

		// when
		importID := provider.BuildImportIDForTest(t.Context(), *model, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.NotContains(t, importID.ValueString(), "basic_auth")

		encoded := importID.ValueString()[len("fixture-id/"):]
		decoded, err := base64.RawURLEncoding.DecodeString(encoded)
		require.NoError(t, err)
		assert.NotContains(t, string(decoded), "fixture-password-placeholder")
		assert.NotContains(t, string(decoded), "fixture-user")
	})

	t.Run("should emit URL-safe base64 so the identifier never contains a stray slash", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		model, _ := provider.DecodeImportIDForTest(
			`{"method":"GET","path":"/api/items?q=1&r=2","headers":{"X-A":"a/b/c","X-B":"d?e"}}`,
			&diagnostics,
		)
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		model.ID = types.StringValue("fixture-id")

		// when
		importID := provider.BuildImportIDForTest(t.Context(), *model, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		payload := importID.ValueString()[len("fixture-id/"):]
		assert.NotContains(t, payload, "/", "the payload must not contain the separator")
		assert.NotContains(t, payload, "=", "the payload must be unpadded")
	})
}

func TestPendingAdoptionFor(t *testing.T) {
	t.Parallel()

	t.Run("should list every adoptable argument the payload left out", func(t *testing.T) {
		t.Parallel()

		// given
		specified := map[string]struct{}{"method": {}, "path": {}}

		// when
		pending := provider.PendingAdoptionForTest(specified)

		// then
		assert.Equal(t, []string{
			"base_url",
			"headers",
			"ignore_tls",
			"is_response_body_json",
			"query_parameters",
			"request_body",
			"response_body_id_filter",
		}, pending)
	})

	t.Run("should report nothing pending when the payload spelled everything out", func(t *testing.T) {
		t.Parallel()

		// given
		specified := map[string]struct{}{
			"method": {}, "path": {}, "headers": {}, "request_body": {},
			"query_parameters": {}, "base_url": {}, "ignore_tls": {},
			"is_response_body_json": {}, "response_body_id_filter": {},
		}

		// when
		pending := provider.PendingAdoptionForTest(specified)

		// then
		assert.Empty(t, pending)
	})
}

func TestImportAdoptPrivateState(t *testing.T) {
	t.Parallel()

	t.Run("should round-trip the pending adoption", func(t *testing.T) {
		t.Parallel()

		// given
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics

		// when
		diagnostics.Append(provider.MarshalImportAdoptForTest(t.Context(), []string{"headers"}, private)...)
		attributes, diags := provider.UnmarshalImportAdoptForTest(t.Context(), private)
		diagnostics.Append(diags...)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.Equal(t, []string{"headers"}, attributes)
	})

	t.Run("should store nothing when there is no pending adoption", func(t *testing.T) {
		t.Parallel()

		// given
		// Writing an empty flag would leave behind a marker that no apply ever clears.
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics

		// when
		diagnostics.Append(provider.MarshalImportAdoptForTest(t.Context(), nil, private)...)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.Empty(t, private.data)
	})

	t.Run("should clear the pending adoption", func(t *testing.T) {
		t.Parallel()

		// given
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics
		diagnostics.Append(provider.MarshalImportAdoptForTest(t.Context(), []string{"headers"}, private)...)

		// when
		diagnostics.Append(provider.ClearImportAdoptForTest(t.Context(), private)...)
		attributes, diags := provider.UnmarshalImportAdoptForTest(t.Context(), private)
		diagnostics.Append(diags...)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.Nil(t, attributes)
	})
}

func TestDeleteParamsPrivateState(t *testing.T) {
	t.Parallel()

	t.Run("should round-trip the write-only destroy controls", func(t *testing.T) {
		t.Parallel()

		// given
		private := newPrivateStateDouble()
		var diagnostics diag.Diagnostics
		model, _ := provider.DecodeImportIDForTest(
			`{"method":"POST","path":"/posts","is_delete_enabled":true,"delete_method":"DELETE",`+
				`"delete_path":"/posts/1","delete_headers":{"X-A":"a"},"delete_request_body":"{}"}`,
			&diagnostics,
		)
		require.False(t, diagnostics.HasError(), diagnostics.Errors())

		// when
		diagnostics.Append(provider.MarshalDeleteParamsForTest(t.Context(), *model, private)...)
		var restored provider.HTTPRequestResourceModel
		diagnostics.Append(provider.ApplyDeleteParamsForTest(t.Context(), &restored, private)...)

		// then
		require.False(t, diagnostics.HasError(), diagnostics.Errors())
		assert.True(t, restored.IsDeleteEnabled.ValueBool())
		assert.Equal(t, "DELETE", restored.DeleteMethod.ValueString())
		assert.Equal(t, "/posts/1", restored.DeletePath.ValueString())
		assert.Equal(t, "{}", restored.DeleteRequestBody.ValueString())
		assert.Len(t, restored.DeleteHeaders.Elements(), 1)
	})
}

// privateStateDouble is a hand-rolled in-memory stand-in for the framework's provider private
// state. It records what was written so a test can assert on it directly, which the framework's
// own type does not allow from outside its package.
type privateStateDouble struct {
	data map[string][]byte
}

func newPrivateStateDouble() *privateStateDouble {
	return &privateStateDouble{data: make(map[string][]byte)}
}

func (p *privateStateDouble) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	var diagnostics diag.Diagnostics

	if len(value) == 0 {
		delete(p.data, key)

		return diagnostics
	}

	if !json.Valid(value) {
		diagnostics.AddError("Invalid private state", "the value must be valid JSON")

		return diagnostics
	}

	p.data[key] = value

	return diagnostics
}

func (p *privateStateDouble) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return p.data[key], nil
}
