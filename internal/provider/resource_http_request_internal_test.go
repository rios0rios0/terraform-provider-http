package provider

import (
	"context"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullResourceType names every attribute of the resource schema, which is what a raw value handed
// to ValidateConfig has to match.
func fullResourceType() tftypes.Object {
	return builders.NewResourceTypeBuilder().
		WithMethod().
		WithPath().
		WithHeaders().
		WithRequestBody().
		WithIsResponseBodyJSON().
		WithResponseBodyIDFilter().
		WithQueryParameters().
		WithIgnoreChanges().
		WithIsDeleteEnabled().
		WithDeleteMethod().
		WithDeletePath().
		WithDeleteHeaders().
		WithDeleteRequestBody().
		WithDeleteResolvedPath().
		WithID().
		WithResponseCode().
		WithResponseBody().
		WithResponseBodyID().
		WithResponseBodyJSON().
		Build()
}

// nullResourceValues returns every attribute of fullResourceType as a typed null. A case sets the
// handful of attributes it is about on top of it instead of spelling all nineteen out each time.
func nullResourceValues() map[string]tftypes.Value {
	stringMap := tftypes.Map{ElementType: tftypes.String}

	return map[string]tftypes.Value{
		"method":                  tftypes.NewValue(tftypes.String, nil),
		"path":                    tftypes.NewValue(tftypes.String, nil),
		"headers":                 tftypes.NewValue(stringMap, nil),
		"request_body":            tftypes.NewValue(tftypes.String, nil),
		"is_response_body_json":   tftypes.NewValue(tftypes.Bool, nil),
		"response_body_id_filter": tftypes.NewValue(tftypes.String, nil),
		"query_parameters":        tftypes.NewValue(stringMap, nil),
		"ignore_changes":          tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),

		// Destroy controls
		"is_delete_enabled":    tftypes.NewValue(tftypes.Bool, nil),
		"delete_method":        tftypes.NewValue(tftypes.String, nil),
		"delete_path":          tftypes.NewValue(tftypes.String, nil),
		"delete_headers":       tftypes.NewValue(stringMap, nil),
		"delete_request_body":  tftypes.NewValue(tftypes.String, nil),
		"delete_resolved_path": tftypes.NewValue(tftypes.String, nil),

		// Computed fields
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"response_code":      tftypes.NewValue(tftypes.Number, nil),
		"response_body":      tftypes.NewValue(tftypes.String, nil),
		"response_body_id":   tftypes.NewValue(tftypes.String, nil),
		"response_body_json": tftypes.NewValue(stringMap, nil),
	}
}

// resourceConfigOf builds the raw resource value ValidateConfig receives from the attributes a case
// sets; every other attribute stays null.
func resourceConfigOf(set map[string]tftypes.Value) tftypes.Value {
	values := nullResourceValues()
	maps.Copy(values, set)

	return tftypes.NewValue(fullResourceType(), values)
}

// validateResourceConfigOf runs the resource's ValidateConfig over a raw value and returns the
// diagnostics, the counterpart of validateConfigOf for the provider.
func validateResourceConfigOf(raw tftypes.Value) diag.Diagnostics {
	req := fresource.ValidateConfigRequest{
		Config: tfsdk.Config{Raw: raw, Schema: GetHTTPRequestResourceSchema()},
	}
	resp := fresource.ValidateConfigResponse{Diagnostics: make(diag.Diagnostics, 0)}

	it := &HTTPRequestResource{}
	it.ValidateConfig(context.Background(), req, &resp)

	return resp.Diagnostics
}

// stringValue keeps the attribute tables below to one line per attribute.
func stringValue(value string) tftypes.Value {
	return tftypes.NewValue(tftypes.String, value)
}

// boolValue is stringValue for booleans.
func boolValue(value bool) tftypes.Value {
	return tftypes.NewValue(tftypes.Bool, value)
}

func TestHTTPRequestResource_ValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("should not throw any error when the 'method' and 'path' are set", func(t *testing.T) {
		t.Parallel()

		// given
		raw := resourceConfigOf(map[string]tftypes.Value{
			"method":                stringValue("GET"),
			"path":                  stringValue("/posts/1"),
			"is_response_body_json": boolValue(false),
		})

		// when
		diagnostics := validateResourceConfigOf(raw)

		// then
		assert.Empty(t, diagnostics, "there should be no errors when required parameters are set")
	})
}

func TestHTTPRequestResource_DestroyValidation(t *testing.T) {
	t.Parallel()

	// Every case is a POST that captures the created id and enables the destroy; the cases differ
	// only in how that destroy is issued.
	cases := []struct {
		name    string
		destroy map[string]tftypes.Value
	}{
		{
			name: "should validate destroy configuration with custom delete_path and JSONPath token",
			destroy: map[string]tftypes.Value{
				"delete_method": stringValue("DELETE"),
				"delete_path":   stringValue("/posts/$.id"),
			},
		},
		{
			name: "should validate destroy configuration with custom delete_method POST",
			destroy: map[string]tftypes.Value{
				"delete_method":       stringValue("POST"),
				"delete_path":         stringValue("/posts/$.id/archive"),
				"delete_request_body": stringValue(`{"reason":"terraform destroy"}`),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			set := map[string]tftypes.Value{
				"method":                  stringValue("POST"),
				"path":                    stringValue("/posts"),
				"request_body":            stringValue(`{"title":"test"}`),
				"is_response_body_json":   boolValue(true),
				"response_body_id_filter": stringValue("$.id"),
				"is_delete_enabled":       boolValue(true),
			}
			maps.Copy(set, tc.destroy)
			raw := resourceConfigOf(set)

			// when
			diagnostics := validateResourceConfigOf(raw)

			// then
			assert.Empty(t, diagnostics, "there should be no errors when the destroy configuration is valid")
		})
	}
}

func TestHTTPRequestResource_JSONPathTokenResolution(t *testing.T) {
	t.Parallel()

	successes := []struct {
		name         string
		rawPath      string
		responseBody string
		want         string
	}{
		{
			name:         "should resolve single JSONPath token in delete_path",
			rawPath:      "/posts/$.id",
			responseBody: `{"id": 123, "title": "test post"}`,
			want:         "/posts/123",
		},
		{
			name:         "should resolve multiple JSONPath tokens in delete_path",
			rawPath:      "/users/$.userId/posts/$.id",
			responseBody: `{"id": 456, "userId": 789, "title": "test post"}`,
			want:         "/users/789/posts/456",
		},
		{
			name:         "should return original path when no JSONPath tokens present",
			rawPath:      "/posts/123",
			responseBody: `{"id": 123, "title": "test post"}`,
			want:         "/posts/123",
		},
		{
			name:         "should resolve nested JSONPath token",
			rawPath:      "/posts/$.data.id",
			responseBody: `{"data": {"id": 999}, "title": "test post"}`,
			want:         "/posts/999",
		},
	}

	for _, tc := range successes {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var diagnostics diag.Diagnostics

			// when
			resolved, ok := resolveDeletePathTokens(tc.rawPath, tc.responseBody, &diagnostics)

			// then
			assert.True(t, ok, "should successfully resolve the path")
			assert.Equal(t, tc.want, resolved, "every token should be replaced by its value")
			assert.Empty(t, diagnostics, "should have no errors")
		})
	}

	failures := []struct {
		name         string
		rawPath      string
		responseBody string
		wantSummary  string
	}{
		{
			name:         "should handle error when JSONPath token not found in response",
			rawPath:      "/posts/$.nonexistent",
			responseBody: `{"id": 123, "title": "test post"}`,
			wantSummary:  "JSONPath token not found",
		},
		{
			name:         "should handle error when response body is invalid JSON",
			rawPath:      "/posts/$.id",
			responseBody: `invalid json`,
			wantSummary:  "unmarshall response body",
		},
	}

	for _, tc := range failures {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var diagnostics diag.Diagnostics

			// when
			resolved, ok := resolveDeletePathTokens(tc.rawPath, tc.responseBody, &diagnostics)

			// then
			assert.False(t, ok, "should fail to resolve")
			assert.Empty(t, resolved, "should return empty string on error")
			require.NotEmpty(t, diagnostics, "should have error diagnostics")
			assert.Contains(t, diagnostics[0].Summary(), tc.wantSummary, "should have appropriate error message")
		})
	}
}

func TestHTTPRequestResource_DestroyHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("isBoolTrue should correctly identify true boolean values", func(t *testing.T) {
		t.Parallel()

		// given
		trueValue := types.BoolValue(true)
		falseValue := types.BoolValue(false)
		nullValue := types.BoolNull()

		// then
		assert.True(t, isBoolTrue(trueValue), "should return true for true boolean")
		assert.False(t, isBoolTrue(falseValue), "should return false for false boolean")
		assert.False(t, isBoolTrue(nullValue), "should return false for null boolean")
	})

	t.Run("isNonEmptyString should correctly identify non-empty strings", func(t *testing.T) {
		t.Parallel()

		// given
		nonEmptyValue := types.StringValue("test")
		emptyValue := types.StringValue("")
		whitespaceValue := types.StringValue("   ")
		nullValue := types.StringNull()

		// then
		assert.True(t, isNonEmptyString(nonEmptyValue), "should return true for non-empty string")
		assert.False(t, isNonEmptyString(emptyValue), "should return false for empty string")
		assert.False(t, isNonEmptyString(whitespaceValue), "should return false for whitespace-only string")
		assert.False(t, isNonEmptyString(nullValue), "should return false for null string")
	})

	t.Run("pickDeleteMethod should return correct HTTP method", func(t *testing.T) {
		t.Parallel()

		// given
		model1 := HTTPRequestResourceModel{
			DeleteMethod: types.StringValue("POST"),
		}
		model2 := HTTPRequestResourceModel{
			DeleteMethod: types.StringValue("  put  "),
		}
		model3 := HTTPRequestResourceModel{
			DeleteMethod: types.StringNull(),
		}
		model4 := HTTPRequestResourceModel{
			DeleteMethod: types.StringValue(""),
		}

		// then
		assert.Equal(t, "POST", pickDeleteMethod(model1), "should return POST for POST")
		assert.Equal(t, "PUT", pickDeleteMethod(model2), "should return PUT and trim whitespace")
		assert.Equal(t, "DELETE", pickDeleteMethod(model3), "should return DELETE for null")
		assert.Equal(t, "DELETE", pickDeleteMethod(model4), "should return DELETE for empty string")
	})

	t.Run("resolveDeleteTargetPath should handle different path scenarios", func(t *testing.T) {
		t.Parallel()

		// Test case 1: No delete_path specified, should use original path
		model1 := HTTPRequestResourceModel{
			Path:       types.StringValue("/posts"),
			DeletePath: types.StringNull(),
		}
		var diag1 diag.Diagnostics
		path1, ok1 := resolveDeleteTargetPath(model1, &diag1)
		assert.True(t, ok1, "should succeed with no delete_path")
		assert.Equal(t, "/posts", path1, "should return original path")
		assert.Empty(t, diag1, "should have no errors")

		// Test case 2: delete_path without JSONPath tokens
		model2 := HTTPRequestResourceModel{
			Path:         types.StringValue("/posts"),
			DeletePath:   types.StringValue("/posts/123"),
			ResponseBody: types.StringValue(`{"id": 123}`), // Need some response body for the function
		}
		var diag2 diag.Diagnostics
		path2, ok2 := resolveDeleteTargetPath(model2, &diag2)
		assert.True(t, ok2, "should succeed with simple delete_path")
		assert.Equal(t, "/posts/123", path2, "should return delete_path")
		assert.Empty(t, diag2, "should have no errors")

		// Test case 3: delete_path already resolved
		model3 := HTTPRequestResourceModel{
			Path:               types.StringValue("/posts"),
			DeletePath:         types.StringValue("/posts/$.id"),
			DeleteResolvedPath: types.StringValue("/posts/456"),
		}
		var diag3 diag.Diagnostics
		path3, ok3 := resolveDeleteTargetPath(model3, &diag3)
		assert.True(t, ok3, "should succeed with resolved path")
		assert.Equal(t, "/posts/456", path3, "should return resolved path")
		assert.Empty(t, diag3, "should have no errors")
	})

	t.Run("makeDeleteModel should create correct delete model", func(t *testing.T) {
		t.Parallel()

		// given
		baseHeaders, _ := types.MapValue(types.StringType, map[string]attr.Value{
			"Content-Type": types.StringValue("application/json"),
		})
		deleteHeaders, _ := types.MapValue(types.StringType, map[string]attr.Value{
			"X-Delete-Reason": types.StringValue("terraform-destroy"),
		})

		baseModel := HTTPRequestResourceModel{
			Method:            types.StringValue("POST"),
			Path:              types.StringValue("/posts"),
			Headers:           baseHeaders,
			RequestBody:       types.StringValue(`{"title":"test"}`),
			DeleteHeaders:     deleteHeaders,
			DeleteRequestBody: types.StringValue(`{"reason":"destroy"}`),
		}

		// when
		deleteModel := makeDeleteModel(baseModel, "DELETE", "/posts/123")

		// then
		assert.Equal(t, "DELETE", deleteModel.Method.ValueString(), "should set delete method")
		assert.Equal(t, "/posts/123", deleteModel.Path.ValueString(), "should set target path")
		assert.JSONEq(
			t,
			`{"reason":"destroy"}`,
			deleteModel.RequestBody.ValueString(),
			"should use delete request body",
		)
		assert.Equal(
			t,
			"terraform-destroy",
			deleteModel.Headers.Elements()["X-Delete-Reason"].(types.String).ValueString(),
			"should use delete headers",
		)
	})
}
