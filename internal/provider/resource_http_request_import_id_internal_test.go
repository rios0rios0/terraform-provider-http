package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// importIDBaselineModel sets every argument buildImportID encodes to a non-null value, so that
// changing any one of them is visible in the rendered identifier.
func importIDBaselineModel() HTTPRequestResourceModel {
	model := sensitiveCaptureModel()
	model.ID = types.StringValue("fixture-id")
	model.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{
		"Accept": types.StringValue("application/json"),
	})
	model.RequestBody = types.StringValue(`{"name":"fixture"}`)
	model.QueryParameters = types.MapValueMust(types.StringType, map[string]attr.Value{
		"page": types.StringValue("1"),
	})
	model.ToleratedStatusCodes = types.SetValueMust(types.Int32Type, []attr.Value{types.Int32Value(409)})
	model.IgnoreChanges = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("headers.Accept")})
	model.BaseURL = types.StringValue("https://api.example.com")
	model.IgnoreTLS = types.BoolValue(false)
	model.RequestTimeoutMs = types.Int64Value(1000)
	model.Retry = types.ObjectValueMust(retryObjectAttrTypes(), map[string]attr.Value{
		attrAttempts:   types.Int64Value(1),
		attrMinDelayMs: types.Int64Value(10),
		attrMaxDelayMs: types.Int64Value(20),
	})
	model.IsRefreshEnabled = types.BoolValue(true)
	model.RefreshPath = types.StringValue("/keys/$.id")
	model.IsResponseBodySensitive = types.BoolValue(false)
	model.ResponseBody = types.StringValue(sensitiveFixtureBody)

	return model
}

// importIDArgumentMutations changes one encoded argument each, keyed by its attribute name.
func importIDArgumentMutations() map[string]func(*HTTPRequestResourceModel) {
	return map[string]func(*HTTPRequestResourceModel){
		attrMethod: func(m *HTTPRequestResourceModel) { m.Method = types.StringValue("PUT") },
		attrPath:   func(m *HTTPRequestResourceModel) { m.Path = types.StringValue("/other") },
		attrHeaders: func(m *HTTPRequestResourceModel) {
			m.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{
				"Accept": types.StringValue("text/plain"),
			})
		},
		attrRequestBody:          func(m *HTTPRequestResourceModel) { m.RequestBody = types.StringValue(`{}`) },
		attrIsResponseBodyJSON:   func(m *HTTPRequestResourceModel) { m.IsResponseBodyJSON = types.BoolValue(false) },
		attrResponseBodyIDFilter: func(m *HTTPRequestResourceModel) { m.ResponseBodyIDFilter = types.StringValue("$.key") },
		attrQueryParameters: func(m *HTTPRequestResourceModel) {
			m.QueryParameters = types.MapNull(types.StringType)
		},
		attrToleratedStatusCodes: func(m *HTTPRequestResourceModel) {
			m.ToleratedStatusCodes = types.SetValueMust(types.Int32Type, []attr.Value{types.Int32Value(404)})
		},
		attrIgnoreChanges: func(m *HTTPRequestResourceModel) { m.IgnoreChanges = types.SetNull(types.StringType) },
		attrBaseURL:       func(m *HTTPRequestResourceModel) { m.BaseURL = types.StringValue("https://other.example.com") },
		attrIgnoreTLS:     func(m *HTTPRequestResourceModel) { m.IgnoreTLS = types.BoolValue(true) },
		attrRequestTimeoutMs: func(m *HTTPRequestResourceModel) {
			m.RequestTimeoutMs = types.Int64Value(2000)
		},
		attrRetry: func(m *HTTPRequestResourceModel) {
			m.Retry = types.ObjectNull(retryObjectAttrTypes())
		},
		attrIsRefreshEnabled: func(m *HTTPRequestResourceModel) { m.IsRefreshEnabled = types.BoolValue(false) },
		attrRefreshPath:      func(m *HTTPRequestResourceModel) { m.RefreshPath = types.StringValue("/keys/latest") },
		attrIsResponseBodySensitive: func(m *HTTPRequestResourceModel) {
			m.IsResponseBodySensitive = types.BoolValue(true)
		},
	}
}

// encodedImportIDKeys returns the JSON keys an import identifier carries.
func encodedImportIDKeys(t *testing.T, importID types.String) []string {
	t.Helper()

	_, encoded, found := strings.Cut(importID.ValueString(), importPathSigil)
	require.True(t, found)

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)

	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(decoded, &payload))

	return slices.Sorted(maps.Keys(payload))
}

func TestHTTPRequestResource_ImportIDArgumentsChanged(t *testing.T) {
	t.Parallel()

	for name, mutate := range importIDArgumentMutations() {
		t.Run("should report a change of "+name, func(t *testing.T) {
			t.Parallel()

			// given
			state := importIDBaselineModel()
			plan := importIDBaselineModel()
			mutate(&plan)
			var diagnostics diag.Diagnostics

			// when
			changed := importIDArgumentsChanged(plan, state)

			// then
			assert.True(t, changed)
			assert.NotEqual(t,
				buildImportID(context.Background(), state, &diagnostics).ValueString(),
				buildImportID(context.Background(), plan, &diagnostics).ValueString(),
				"the mutation must change what import_id encodes, or it is not testing an encoded argument")
			assert.False(t, diagnostics.HasError(), "%v", diagnostics)
		})
	}

	t.Run("should cover every argument the identifier encodes", func(t *testing.T) {
		t.Parallel()

		// given
		// A key buildImportID gains without a matching comparison would leave import_id stale again.
		var diagnostics diag.Diagnostics
		importID := buildImportID(context.Background(), importIDBaselineModel(), &diagnostics)
		require.False(t, diagnostics.HasError(), "%v", diagnostics)

		// when
		encoded := encodedImportIDKeys(t, importID)

		// then
		// `import_read_path` is derived from `method`, `refresh_path` and the captured response.
		covered := slices.Sorted(maps.Keys(importIDArgumentMutations()))
		assert.Equal(t, covered, slices.DeleteFunc(encoded, func(key string) bool { return key == "import_read_path" }))
	})

	t.Run("should report no change when the plan carries the same arguments", func(t *testing.T) {
		t.Parallel()

		// given
		state := importIDBaselineModel()
		plan := importIDBaselineModel()
		plan.ResponseBody = types.StringValue(`{"id":8}`)

		// when
		changed := importIDArgumentsChanged(plan, state)

		// then
		assert.False(t, changed, "the captured response is not an argument")
	})

	t.Run("should count an argument the plan does not know yet as a change", func(t *testing.T) {
		t.Parallel()

		// given
		state := importIDBaselineModel()
		plan := importIDBaselineModel()
		plan.RefreshPath = types.StringUnknown()

		// when
		changed := importIDArgumentsChanged(plan, state)

		// then
		assert.True(t, changed)
	})
}

func TestHTTPRequestResource_PlanKeptResponse(t *testing.T) {
	t.Parallel()

	t.Run("should leave import_id for apply to render when an encoded argument changes", func(t *testing.T) {
		t.Parallel()

		// given
		state := importIDBaselineModel()
		state.ImportID = types.StringValue("fixture-id/stored")
		plan := state
		plan.IsResponseBodySensitive = types.BoolValue(true)

		// when
		planKeptResponse(state, &plan)

		// then
		assert.True(t, plan.ImportID.IsUnknown())
		assert.JSONEq(t, sensitiveFixtureBody, plan.SensitiveResponseBody.ValueString())
	})

	t.Run("should keep the stored import_id when nothing it encodes changes", func(t *testing.T) {
		t.Parallel()

		// given
		// Even an identifier that is out of date stays put: a plan must not propose a change to it
		// on its own, or upgrading the provider would make every such resource show a diff.
		state := importIDBaselineModel()
		state.ImportID = types.StringValue("fixture-id/stored")
		plan := state

		// when
		planKeptResponse(state, &plan)

		// then
		assert.Equal(t, "fixture-id/stored", plan.ImportID.ValueString())
	})
}

func TestHTTPRequestResource_SettleKeptResponse(t *testing.T) {
	t.Parallel()

	t.Run("should render the import_id the plan left unknown", func(t *testing.T) {
		t.Parallel()

		// given
		state := importIDBaselineModel()
		plan := state
		plan.IsResponseBodySensitive = types.BoolValue(true)
		plan.ImportID = types.StringUnknown()
		var diagnostics diag.Diagnostics

		// when
		settleKeptResponse(context.Background(), state, &plan, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		payload := decodeImportID(plan.ImportID.ValueString(), &diagnostics)
		require.NotNil(t, payload, "%v", diagnostics)
		require.NotNil(t, payload.native.IsResponseBodySensitive)
		assert.True(t, *payload.native.IsResponseBodySensitive)
		assert.Equal(t, "/keys/7", payload.native.ImportReadPath, "resolved from the moved response")
	})

	t.Run("should reproduce a planned import_id exactly", func(t *testing.T) {
		t.Parallel()

		// given
		state := importIDBaselineModel()
		plan := state
		plan.ImportID = types.StringValue("fixture-id/stored")
		var diagnostics diag.Diagnostics

		// when
		settleKeptResponse(context.Background(), state, &plan, &diagnostics)

		// then
		assert.Equal(t, "fixture-id/stored", plan.ImportID.ValueString())
	})
}
