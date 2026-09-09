package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
	"github.com/stretchr/testify/assert"
)

func TestHTTPRequestResource_ValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("should not throw any error when the 'method' and 'path' are set", func(t *testing.T) {
		t.Parallel()

		// given
		req := fresource.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewResourceTypeBuilder().
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
						Build(),
					map[string]tftypes.Value{
						"method":                  tftypes.NewValue(tftypes.String, "GET"),
						"path":                    tftypes.NewValue(tftypes.String, "/posts/1"),
						"headers":                 tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"request_body":            tftypes.NewValue(tftypes.String, nil),
						"is_response_body_json":   tftypes.NewValue(tftypes.Bool, false),
						"response_body_id_filter": tftypes.NewValue(tftypes.String, nil),
						"query_parameters":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"ignore_changes":          tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),

						// Destroy controls
						"is_delete_enabled":    tftypes.NewValue(tftypes.Bool, nil),
						"delete_method":        tftypes.NewValue(tftypes.String, nil),
						"delete_path":          tftypes.NewValue(tftypes.String, nil),
						"delete_headers":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"delete_request_body":  tftypes.NewValue(tftypes.String, nil),
						"delete_resolved_path": tftypes.NewValue(tftypes.String, nil),

						// Computed fields
						"id":                 tftypes.NewValue(tftypes.String, nil),
						"response_code":      tftypes.NewValue(tftypes.Number, nil),
						"response_body":      tftypes.NewValue(tftypes.String, nil),
						"response_body_id":   tftypes.NewValue(tftypes.String, nil),
						"response_body_json": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
					},
				),
				Schema: GetHTTPRequestResourceSchema(),
			},
		}
		resp := fresource.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPRequestResource{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Empty(t, resp.Diagnostics, "there should be no errors when required parameters are set")
	})
}

func TestHTTPRequestResource_DestroyValidation(t *testing.T) {
	t.Parallel()

	t.Run("should validate destroy configuration with custom delete_path and JSONPath token", func(t *testing.T) {
		t.Parallel()

		// given
		req := fresource.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewResourceTypeBuilder().
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
						Build(),
					map[string]tftypes.Value{
						"method":                  tftypes.NewValue(tftypes.String, "POST"),
						"path":                    tftypes.NewValue(tftypes.String, "/posts"),
						"headers":                 tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"request_body":            tftypes.NewValue(tftypes.String, `{"title":"test"}`),
						"is_response_body_json":   tftypes.NewValue(tftypes.Bool, true),
						"response_body_id_filter": tftypes.NewValue(tftypes.String, "$.id"),
						"query_parameters":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"ignore_changes":          tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),

						// Destroy controls
						"is_delete_enabled":    tftypes.NewValue(tftypes.Bool, true),
						"delete_method":        tftypes.NewValue(tftypes.String, "DELETE"),
						"delete_path":          tftypes.NewValue(tftypes.String, "/posts/$.id"),
						"delete_headers":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"delete_request_body":  tftypes.NewValue(tftypes.String, nil),
						"delete_resolved_path": tftypes.NewValue(tftypes.String, nil),

						// Computed fields
						"id":                 tftypes.NewValue(tftypes.String, nil),
						"response_code":      tftypes.NewValue(tftypes.Number, nil),
						"response_body":      tftypes.NewValue(tftypes.String, nil),
						"response_body_id":   tftypes.NewValue(tftypes.String, nil),
						"response_body_json": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
					},
				),
				Schema: GetHTTPRequestResourceSchema(),
			},
		}
		resp := fresource.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPRequestResource{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Empty(t, resp.Diagnostics, "there should be no errors when all parameters are correctly set for destroy")
	})

	t.Run("should validate destroy configuration with custom delete_method POST", func(t *testing.T) {
		t.Parallel()

		// given
		req := fresource.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewResourceTypeBuilder().
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
						Build(),
					map[string]tftypes.Value{
						"method":                  tftypes.NewValue(tftypes.String, "POST"),
						"path":                    tftypes.NewValue(tftypes.String, "/posts"),
						"headers":                 tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"request_body":            tftypes.NewValue(tftypes.String, `{"title":"test"}`),
						"is_response_body_json":   tftypes.NewValue(tftypes.Bool, true),
						"response_body_id_filter": tftypes.NewValue(tftypes.String, "$.id"),
						"query_parameters":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"ignore_changes":          tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),

						// Destroy controls - soft delete example
						"is_delete_enabled":    tftypes.NewValue(tftypes.Bool, true),
						"delete_method":        tftypes.NewValue(tftypes.String, "POST"),
						"delete_path":          tftypes.NewValue(tftypes.String, "/posts/$.id/archive"),
						"delete_headers":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"delete_request_body":  tftypes.NewValue(tftypes.String, `{"reason":"terraform destroy"}`),
						"delete_resolved_path": tftypes.NewValue(tftypes.String, nil),

						// Computed fields
						"id":                 tftypes.NewValue(tftypes.String, nil),
						"response_code":      tftypes.NewValue(tftypes.Number, nil),
						"response_body":      tftypes.NewValue(tftypes.String, nil),
						"response_body_id":   tftypes.NewValue(tftypes.String, nil),
						"response_body_json": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
					},
				),
				Schema: GetHTTPRequestResourceSchema(),
			},
		}
		resp := fresource.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPRequestResource{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Empty(t, resp.Diagnostics, "there should be no errors when soft delete configuration is valid")
	})
}

func TestHTTPRequestResource_JSONPathTokenResolution(t *testing.T) {
	t.Parallel()

	t.Run("should resolve single JSONPath token in delete_path", func(t *testing.T) {
		t.Parallel()

		// given
		rawPath := "/posts/$.id"
		responseBody := `{"id": 123, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully resolve token")
		assert.Equal(t, "/posts/123", resolved, "should replace $.id with 123")
		assert.Empty(t, diagnostics, "should have no errors")
	})

	t.Run("should resolve multiple JSONPath tokens in delete_path", func(t *testing.T) {
		t.Parallel()

		// given
		rawPath := "/users/$.userId/posts/$.id"
		responseBody := `{"id": 456, "userId": 789, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully resolve tokens")
		assert.Equal(t, "/users/789/posts/456", resolved, "should replace both tokens")
		assert.Empty(t, diagnostics, "should have no errors")
	})

	t.Run("should return original path when no JSONPath tokens present", func(t *testing.T) {
		t.Parallel()

		// given
		rawPath := "/posts/123"
		responseBody := `{"id": 123, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully process")
		assert.Equal(t, "/posts/123", resolved, "should return original path unchanged")
		assert.Empty(t, diagnostics, "should have no errors")
	})

	t.Run("should handle error when JSONPath token not found in response", func(t *testing.T) {
		t.Parallel()

		// given
		rawPath := "/posts/$.nonexistent"
		responseBody := `{"id": 123, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.False(t, ok, "should fail to resolve")
		assert.Empty(t, resolved, "should return empty string on error")
		assert.NotEmpty(t, diagnostics, "should have error diagnostics")
		assert.Contains(
			t,
			diagnostics[0].Summary(),
			"JSONPath token not found",
			"should have appropriate error message",
		)
	})

	t.Run("should handle error when response body is invalid JSON", func(t *testing.T) {
		t.Parallel()

		// given
		rawPath := "/posts/$.id"
		responseBody := `invalid json`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.False(t, ok, "should fail to resolve")
		assert.Empty(t, resolved, "should return empty string on error")
		assert.NotEmpty(t, diagnostics, "should have error diagnostics")
		assert.Contains(
			t,
			diagnostics[0].Summary(),
			"unmarshall response body",
			"should have appropriate error message",
		)
	})

	t.Run("should resolve nested JSONPath token", func(t *testing.T) {
		t.Parallel()

		// given
		rawPath := "/posts/$.data.id"
		responseBody := `{"data": {"id": 999}, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully resolve nested token")
		assert.Equal(t, "/posts/999", resolved, "should replace $.data.id with 999")
		assert.Empty(t, diagnostics, "should have no errors")
	})
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
