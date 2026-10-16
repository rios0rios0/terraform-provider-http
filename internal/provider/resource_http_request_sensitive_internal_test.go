package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The secret a sensitive endpoint returns in these tests. It is a placeholder on purpose: a
// realistic key shape would trip the secret scanners.
const sensitiveFixtureKey = "fixture-token-placeholder"

// sensitiveFixtureBody is the compacted JSON response of an endpoint that mints a key.
const sensitiveFixtureBody = `{"api_key":"` + sensitiveFixtureKey + `","id":7}`

// capturedFixtureJSON is sensitiveFixtureBody parsed the way `response_body_json` holds it.
func capturedFixtureJSON() types.Map {
	return types.MapValueMust(types.StringType, map[string]attr.Value{
		"api_key": types.StringValue(sensitiveFixtureKey),
		"id":      types.StringValue("7"),
	})
}

// modelCapturedPlain is a state written without the flag: the response sits in the plain pair.
func modelCapturedPlain() HTTPRequestResourceModel {
	return HTTPRequestResourceModel{
		IsResponseBodySensitive:   types.BoolNull(),
		ResponseBody:              types.StringValue(sensitiveFixtureBody),
		ResponseBodyJSON:          capturedFixtureJSON(),
		SensitiveResponseBody:     types.StringNull(),
		SensitiveResponseBodyJSON: types.MapNull(types.StringType),
	}
}

// modelCapturedSensitive is a state written with the flag: the response sits in the sensitive pair.
func modelCapturedSensitive() HTTPRequestResourceModel {
	return HTTPRequestResourceModel{
		IsResponseBodySensitive:   types.BoolValue(true),
		ResponseBody:              types.StringNull(),
		ResponseBodyJSON:          types.MapNull(types.StringType),
		SensitiveResponseBody:     types.StringValue(sensitiveFixtureBody),
		SensitiveResponseBodyJSON: capturedFixtureJSON(),
	}
}

func TestHTTPRequestResource_SensitiveResponseSchema(t *testing.T) {
	t.Parallel()

	t.Run("should declare both captured-response attributes as computed and sensitive", func(t *testing.T) {
		t.Parallel()

		// given
		attributes := GetHTTPRequestResourceSchema().Attributes

		// when
		body, bodyOK := attributes[attrSensitiveResponseBody].(schema.StringAttribute)
		bodyJSON, bodyJSONOK := attributes[attrSensitiveResponseBodyJSON].(schema.MapAttribute)

		// then
		require.True(t, bodyOK)
		require.True(t, bodyJSONOK)
		assert.True(t, body.Computed && body.Sensitive)
		assert.True(t, bodyJSON.Computed && bodyJSON.Sensitive)
		assert.Equal(t, types.StringType, bodyJSON.ElementType)
	})

	t.Run("should keep the flag optional and free of replacement", func(t *testing.T) {
		t.Parallel()

		// given
		attributes := GetHTTPRequestResourceSchema().Attributes

		// when
		flag, ok := attributes[attrIsResponseBodySensitive].(schema.BoolAttribute)

		// then
		require.True(t, ok)
		assert.True(t, flag.Optional)
		assert.False(t, flag.Computed)
		assert.Empty(t, flag.PlanModifiers, "toggling the flag must never replace the resource")
	})

	t.Run("should leave the plain response attributes non-sensitive", func(t *testing.T) {
		t.Parallel()

		// given
		attributes := GetHTTPRequestResourceSchema().Attributes

		// when
		body, bodyOK := attributes[attrResponseBody].(schema.StringAttribute)
		bodyJSON, bodyJSONOK := attributes[attrResponseBodyJSON].(schema.MapAttribute)

		// then
		require.True(t, bodyOK)
		require.True(t, bodyJSONOK)
		assert.False(t, body.Sensitive, "outputs that read response_body without sensitive = true must keep working")
		assert.False(t, bodyJSON.Sensitive)
	})

	t.Run("should keep the new attributes out of the prior schemas the upgraders decode with", func(t *testing.T) {
		t.Parallel()

		// given
		prior := []schema.Schema{
			getHTTPRequestResourceSchemaPreV3(schemaVersionV1),
			getHTTPRequestResourceSchemaPreV3(schemaVersionV2),
		}

		for _, priorSchema := range prior {
			// when
			_, hasFlag := priorSchema.Attributes[attrIsResponseBodySensitive]
			_, hasBody := priorSchema.Attributes[attrSensitiveResponseBody]
			_, hasJSON := priorSchema.Attributes[attrSensitiveResponseBodyJSON]

			// then
			assert.False(t, hasFlag || hasBody || hasJSON,
				"a prior schema with them would fail to decode into httpRequestResourceModelPreV3")
		}
	})
}

func TestHTTPRequestResource_PlaceCapturedResponse(t *testing.T) {
	t.Parallel()

	t.Run("should hold the response only in the sensitive pair when the flag is true", func(t *testing.T) {
		t.Parallel()

		// given
		model := modelCapturedPlain()
		model.IsResponseBodySensitive = types.BoolValue(true)

		// when
		placeCapturedResponse(&model, types.StringValue(sensitiveFixtureBody), capturedFixtureJSON())

		// then
		assert.JSONEq(t, sensitiveFixtureBody, model.SensitiveResponseBody.ValueString())
		assert.True(t, model.SensitiveResponseBodyJSON.Equal(capturedFixtureJSON()))
		assert.True(t, model.ResponseBody.IsNull())
		assert.True(t, model.ResponseBodyJSON.IsNull())
		assert.Equal(t, types.StringType, model.ResponseBodyJSON.ElementType(context.Background()),
			"the null map must stay typed, or the next plan fails with MISSING TYPE")
	})

	for _, flag := range []types.Bool{types.BoolValue(false), types.BoolNull()} {
		t.Run("should hold the response only in the plain pair when the flag is "+flag.String(), func(t *testing.T) {
			t.Parallel()

			// given
			model := modelCapturedSensitive()
			model.IsResponseBodySensitive = flag

			// when
			placeCapturedResponse(&model, types.StringValue(sensitiveFixtureBody), capturedFixtureJSON())

			// then
			assert.JSONEq(t, sensitiveFixtureBody, model.ResponseBody.ValueString())
			assert.True(t, model.ResponseBodyJSON.Equal(capturedFixtureJSON()))
			assert.True(t, model.SensitiveResponseBody.IsNull())
			assert.True(t, model.SensitiveResponseBodyJSON.IsNull())
			assert.Equal(t, types.StringType, model.SensitiveResponseBodyJSON.ElementType(context.Background()))
		})
	}
}

func TestHTTPRequestResource_CapturedResponse(t *testing.T) {
	t.Parallel()

	t.Run("should read the sensitive pair when the flag is true", func(t *testing.T) {
		t.Parallel()

		// given
		model := modelCapturedSensitive()

		// when
		body, bodyJSON := capturedResponse(model)

		// then
		assert.JSONEq(t, sensitiveFixtureBody, body.ValueString())
		assert.True(t, bodyJSON.Equal(capturedFixtureJSON()))
	})

	t.Run("should read the plain pair when the flag is unset", func(t *testing.T) {
		t.Parallel()

		// given
		model := modelCapturedPlain()

		// when
		body, bodyJSON := capturedResponse(model)

		// then
		assert.JSONEq(t, sensitiveFixtureBody, body.ValueString())
		assert.True(t, bodyJSON.Equal(capturedFixtureJSON()))
	})
}

func TestHTTPRequestResource_PlanCapturedResponse(t *testing.T) {
	t.Parallel()

	t.Run("should move the captured response into the sensitive pair when the flag is turned on", func(t *testing.T) {
		t.Parallel()

		// given
		state := modelCapturedPlain()
		plan := modelCapturedPlain()
		plan.IsResponseBodySensitive = types.BoolValue(true)

		// when
		planCapturedResponse(state, &plan)

		// then
		assert.JSONEq(t, sensitiveFixtureBody, plan.SensitiveResponseBody.ValueString())
		assert.True(t, plan.SensitiveResponseBodyJSON.Equal(capturedFixtureJSON()))
		assert.True(t, plan.ResponseBody.IsNull())
		assert.True(t, plan.ResponseBodyJSON.IsNull())
	})

	t.Run("should move the captured response back into the plain pair when the flag is turned off", func(t *testing.T) {
		t.Parallel()

		// given
		state := modelCapturedSensitive()
		plan := modelCapturedSensitive()
		plan.IsResponseBodySensitive = types.BoolValue(false)

		// when
		planCapturedResponse(state, &plan)

		// then
		assert.JSONEq(t, sensitiveFixtureBody, plan.ResponseBody.ValueString())
		assert.True(t, plan.ResponseBodyJSON.Equal(capturedFixtureJSON()))
		assert.True(t, plan.SensitiveResponseBody.IsNull())
		assert.True(t, plan.SensitiveResponseBodyJSON.IsNull())
	})

	t.Run("should leave both pairs unknown when the flag is unknown until apply", func(t *testing.T) {
		t.Parallel()

		// given
		state := modelCapturedPlain()
		plan := modelCapturedPlain()
		plan.IsResponseBodySensitive = types.BoolUnknown()

		// when
		planCapturedResponse(state, &plan)

		// then
		assert.True(t, plan.ResponseBody.IsUnknown())
		assert.True(t, plan.ResponseBodyJSON.IsUnknown())
		assert.True(t, plan.SensitiveResponseBody.IsUnknown())
		assert.True(t, plan.SensitiveResponseBodyJSON.IsUnknown())
	})
}

func TestHTTPRequestResource_MarkCapturedResponseUnknown(t *testing.T) {
	t.Parallel()

	t.Run("should mark only the sensitive pair unknown when the flag is true", func(t *testing.T) {
		t.Parallel()

		// given
		plan := modelCapturedSensitive()

		// when
		markCapturedResponseUnknown(&plan)

		// then
		assert.True(t, plan.SensitiveResponseBody.IsUnknown())
		assert.True(t, plan.SensitiveResponseBodyJSON.IsUnknown())
		assert.True(t, plan.ResponseBody.IsNull())
		assert.True(t, plan.ResponseBodyJSON.IsNull())
	})

	t.Run("should mark only the plain pair unknown when the flag is unset", func(t *testing.T) {
		t.Parallel()

		// given
		plan := modelCapturedPlain()

		// when
		markCapturedResponseUnknown(&plan)

		// then
		assert.True(t, plan.ResponseBody.IsUnknown())
		assert.True(t, plan.ResponseBodyJSON.IsUnknown())
		assert.True(t, plan.SensitiveResponseBody.IsNull())
		assert.True(t, plan.SensitiveResponseBodyJSON.IsNull())
	})
}

// sensitiveCaptureModel is a resource that mints a key, parses it and can delete it again.
func sensitiveCaptureModel() HTTPRequestResourceModel {
	return HTTPRequestResourceModel{
		Method:                    types.StringValue("POST"),
		Path:                      types.StringValue("/keys"),
		Headers:                   types.MapNull(types.StringType),
		QueryParameters:           types.MapNull(types.StringType),
		ToleratedStatusCodes:      types.SetNull(types.Int32Type),
		IgnoreChanges:             types.SetNull(types.StringType),
		BasicAuth:                 types.ObjectNull(basicAuthObjectAttrTypesForTest()),
		Retry:                     types.ObjectNull(retryObjectAttrTypes()),
		IsResponseBodyJSON:        types.BoolValue(true),
		ResponseBodyIDFilter:      types.StringValue("$.id"),
		DeletePath:                types.StringValue("/keys/$.id"),
		IsResponseBodySensitive:   types.BoolValue(true),
		ResponseBody:              types.StringNull(),
		ResponseBodyJSON:          types.MapNull(types.StringType),
		SensitiveResponseBody:     types.StringNull(),
		SensitiveResponseBodyJSON: types.MapNull(types.StringType),
	}
}

func basicAuthObjectAttrTypesForTest() map[string]attr.Type {
	return map[string]attr.Type{attrUsername: types.StringType, attrPassword: types.StringType}
}

func TestHTTPRequestResource_PopulateResponseStateSensitive(t *testing.T) {
	t.Parallel()

	t.Run("should capture the response only into the sensitive pair when the flag is true", func(t *testing.T) {
		t.Parallel()

		// given
		model := sensitiveCaptureModel()
		exchange := &httpExchange{
			statusCode: 201,
			body:       []byte(`{ "api_key": "` + sensitiveFixtureKey + `", "id": 7 }`),
		}
		var diagnostics diag.Diagnostics

		// when
		populateResponseState(context.Background(), &model, exchange, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		assert.True(t, model.SensitiveResponseBody.Equal(types.StringValue(sensitiveFixtureBody)),
			"compacted exactly like response_body, got %s", model.SensitiveResponseBody)
		assert.True(t, model.SensitiveResponseBodyJSON.Equal(capturedFixtureJSON()))
		assert.True(t, model.ResponseBody.IsNull())
		assert.True(t, model.ResponseBodyJSON.IsNull())
		assert.Equal(t, "7", model.ResponseBodyID.ValueString(), "the id is still extracted from the body")
		assert.Equal(t, "/keys/7", model.DeleteResolvedPath.ValueString(), "delete_path tokens still resolve")
		assert.NotContains(t, model.ImportID.ValueString(), sensitiveFixtureKey)
	})

	t.Run("should replace the response an earlier capture left in the sensitive pair", func(t *testing.T) {
		t.Parallel()

		// given
		// A refresh re-captures into a model that already holds the previous response.
		model := sensitiveCaptureModel()
		model.SensitiveResponseBody = types.StringValue(`{"api_key":"fixture-stale-placeholder","id":7}`)
		model.SensitiveResponseBodyJSON = types.MapValueMust(types.StringType, map[string]attr.Value{
			"api_key": types.StringValue("fixture-stale-placeholder"),
			"id":      types.StringValue("7"),
		})
		exchange := &httpExchange{statusCode: 200, body: []byte(sensitiveFixtureBody)}
		var diagnostics diag.Diagnostics

		// when
		populateResponseState(context.Background(), &model, exchange, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		assert.JSONEq(t, sensitiveFixtureBody, model.SensitiveResponseBody.ValueString())
		assert.True(t, model.SensitiveResponseBodyJSON.Equal(capturedFixtureJSON()))
	})

	t.Run("should keep capturing into the plain pair when the flag is unset", func(t *testing.T) {
		t.Parallel()

		// given
		model := sensitiveCaptureModel()
		model.IsResponseBodySensitive = types.BoolNull()
		exchange := &httpExchange{statusCode: 200, body: []byte(sensitiveFixtureBody)}
		var diagnostics diag.Diagnostics

		// when
		populateResponseState(context.Background(), &model, exchange, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		assert.JSONEq(t, sensitiveFixtureBody, model.ResponseBody.ValueString())
		assert.True(t, model.ResponseBodyJSON.Equal(capturedFixtureJSON()))
		assert.True(t, model.SensitiveResponseBody.IsNull())
		assert.True(t, model.SensitiveResponseBodyJSON.IsNull())
	})
}

func TestHTTPRequestResource_PathsFromSensitiveResponse(t *testing.T) {
	t.Parallel()

	t.Run("should resolve refresh_path tokens against the sensitive pair", func(t *testing.T) {
		t.Parallel()

		// given
		model := modelCapturedSensitive()
		model.Path = types.StringValue("/keys")
		model.RefreshPath = types.StringValue("/keys/$.id")
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveRefreshPath(model, &diagnostics)

		// then
		require.True(t, ok, "%v", diagnostics)
		assert.Equal(t, "/keys/7", resolved)
	})

	t.Run("should resolve delete_path tokens against the sensitive pair", func(t *testing.T) {
		t.Parallel()

		// given
		// A state without delete_resolved_path, so destroy has to resolve the tokens itself.
		model := modelCapturedSensitive()
		model.Path = types.StringValue("/keys")
		model.DeletePath = types.StringValue("/keys/$.id")
		model.DeleteResolvedPath = types.StringNull()
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeleteTargetPath(model, &diagnostics)

		// then
		require.True(t, ok, "%v", diagnostics)
		assert.Equal(t, "/keys/7", resolved)
	})

	t.Run("should resolve the import read path against the sensitive pair", func(t *testing.T) {
		t.Parallel()

		// given
		model := modelCapturedSensitive()
		model.Method = types.StringValue("POST")
		model.RefreshPath = types.StringValue("/keys/$.id")

		// when
		resolved := importReadPathForIdentifier(model)

		// then
		assert.Equal(t, "/keys/7", resolved)
	})
}

func TestHTTPRequestResource_ImportSensitiveResponse(t *testing.T) {
	t.Parallel()

	t.Run("should decode the flag from an import payload", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		payload := decodeImportID(`{"method":"GET","path":"/keys/7","is_response_body_sensitive":true}`, &diagnostics)
		require.NotNil(t, payload, "%v", diagnostics)

		// when
		model := buildModelFromNativeData(payload.native, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		assert.True(t, model.IsResponseBodySensitive.ValueBool())
		assert.True(t, model.SensitiveResponseBodyJSON.IsNull())
		assert.Equal(t, types.StringType, model.SensitiveResponseBodyJSON.ElementType(context.Background()))
	})

	t.Run("should render the flag into import_id so a re-import keeps the response sensitive", func(t *testing.T) {
		t.Parallel()

		// given
		model := sensitiveCaptureModel()
		model.ID = types.StringValue("fixture-id")
		model.SensitiveResponseBody = types.StringValue(sensitiveFixtureBody)
		var diagnostics diag.Diagnostics

		// when
		importID := buildImportID(context.Background(), model, &diagnostics)
		payload := decodeImportID(importID.ValueString(), &diagnostics)
		decoded := buildModelFromNativeData(payload.native, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		assert.True(t, decoded.IsResponseBodySensitive.ValueBool())
		assert.True(t, decoded.ResponseBody.IsNull(), "the captured response is never encoded into import_id")
	})

	t.Run("should capture a payload's response into the sensitive pair", func(t *testing.T) {
		t.Parallel()

		// given
		var diagnostics diag.Diagnostics
		payload := decodeImportID(
			`{"method":"POST","path":"/keys","is_response_body_json":true,"response_body_id_filter":"$.id",`+
				`"is_response_body_sensitive":true,"response_body":"{\"api_key\":\"`+sensitiveFixtureKey+`\",\"id\":7}"}`,
			&diagnostics,
		)
		require.NotNil(t, payload, "%v", diagnostics)
		model := buildModelFromNativeData(payload.native, &diagnostics)
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		it := &HTTPRequestResource{}

		// when
		it.captureImportedResponse(context.Background(), model, payload.native, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		assert.JSONEq(t, sensitiveFixtureBody, model.SensitiveResponseBody.ValueString())
		assert.True(t, model.SensitiveResponseBodyJSON.Equal(capturedFixtureJSON()))
		assert.True(t, model.ResponseBody.IsNull())
		assert.True(t, model.ResponseBodyJSON.IsNull())
	})

	t.Run("should move a payload's parsed map into the sensitive pair when none was captured", func(t *testing.T) {
		t.Parallel()

		// given
		// A POST is never replayed, and this payload names no import_read_path.
		var diagnostics diag.Diagnostics
		payload := decodeImportID(
			`{"method":"POST","path":"/keys","is_response_body_sensitive":true,`+
				`"response_body_json":{"api_key":"`+sensitiveFixtureKey+`","id":"7"}}`,
			&diagnostics,
		)
		require.NotNil(t, payload, "%v", diagnostics)
		model := buildModelFromNativeData(payload.native, &diagnostics)
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		it := &HTTPRequestResource{}

		// when
		it.captureImportedResponse(context.Background(), model, payload.native, &diagnostics)

		// then
		require.False(t, diagnostics.HasError(), "%v", diagnostics)
		assert.True(t, model.SensitiveResponseBodyJSON.Equal(capturedFixtureJSON()))
		assert.True(t, model.ResponseBodyJSON.IsNull())
		assert.True(t, model.ResponseBody.IsNull())
	})
}

func TestHTTPRequestResource_UpgradeStateSensitiveResponse(t *testing.T) {
	t.Parallel()

	// Every stored state shape this provider has ever written, as Terraform hands it back. Version 3
	// is the shape providers before the sensitive-response attributes wrote: it reaches no
	// upgrader at all, the framework decodes it against the current schema directly.
	states := []struct {
		name    string
		version int64
		raw     string
	}{
		{
			name:    "version 0",
			version: 0,
			raw: `{"method":"POST","path":"/keys","id":"fixture-id","response_body":"{\"id\":7}",` +
				`"is_delete_enabled":true,"delete_headers":{"X-Trace":"fixture"}}`,
		},
		{
			name:    "version 1",
			version: schemaVersionV1,
			raw:     `{"method":"POST","path":"/keys","id":"fixture-id","response_body":"{\"id\":7}"}`,
		},
		{
			name:    "version 2",
			version: schemaVersionV2,
			raw:     `{"method":"POST","path":"/keys","id":"fixture-id","response_body":"{\"id\":7}"}`,
		},
		{
			name:    "version 3 without the sensitive-response attributes",
			version: schemaVersionV3,
			raw: `{"method":"POST","path":"/keys","id":"fixture-id","response_body":"{\"id\":7}",` +
				`"is_refresh_enabled":true,"import_id":"fixture-id/e30"}`,
		},
	}

	for _, state := range states {
		t.Run("should upgrade a "+state.name+" state with the new attributes as typed nulls", func(t *testing.T) {
			t.Parallel()

			// given
			ctx := context.Background()
			server, err := providerserver.NewProtocol6WithError(New("test")())()
			require.NoError(t, err)
			currentSchema := GetHTTPRequestResourceSchema()

			// when
			resp, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "http_request",
				Version:  state.version,
				RawState: &tfprotov6.RawState{JSON: []byte(state.raw)},
			})

			// then
			require.NoError(t, err)
			for _, diagnostic := range resp.Diagnostics {
				assert.NotEqual(t, tfprotov6.DiagnosticSeverityError, diagnostic.Severity,
					"%s: %s", diagnostic.Summary, diagnostic.Detail)
			}
			require.NotNil(t, resp.UpgradedState)

			raw, err := resp.UpgradedState.Unmarshal(currentSchema.Type().TerraformType(ctx))
			require.NoError(t, err)

			var model HTTPRequestResourceModel
			getDiagnostics := tfsdk.State{Raw: raw, Schema: currentSchema}.Get(ctx, &model)
			require.False(t, getDiagnostics.HasError(), "%v", getDiagnostics)
			assert.True(t, model.IsResponseBodySensitive.IsNull())
			assert.True(t, model.SensitiveResponseBody.IsNull())
			assert.True(t, model.SensitiveResponseBodyJSON.IsNull())
			assert.Equal(t, types.StringType, model.SensitiveResponseBodyJSON.ElementType(ctx))
			assert.Equal(t, `{"id":7}`, model.ResponseBody.ValueString(), "the captured response stays where it was")
		})
	}
}
