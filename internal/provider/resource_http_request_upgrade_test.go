package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rios0rios0/terraform-provider-http/internal/provider"
	"github.com/stretchr/testify/assert"
)

// baselineUpgradeModel returns a schema v1 state as the provider used to write it: the
// create response carried the numeric id 803554429, and every attribute derived from it was
// rendered with an exponent.
func baselineUpgradeModel() provider.HTTPRequestResourceModel {
	return provider.HTTPRequestResourceModel{
		Method:               types.StringValue("POST"),
		Path:                 types.StringValue("/posts"),
		IsResponseBodyJSON:   types.BoolValue(true),
		ResponseBodyIDFilter: types.StringValue("$.id"),
		ResponseBody:         types.StringValue(`{"id":803554429,"title":"example","views":60}`),
		ResponseBodyID:       types.StringValue("8.03554429e+08"),
		DeleteResolvedPath:   types.StringValue("/posts/8.03554429e+08"),
		ResponseBodyJSON: types.MapValueMust(types.StringType, map[string]attr.Value{
			"id":    types.StringValue("8.03554429e+08"),
			"title": types.StringValue("example"),
			"views": types.StringValue("60"),
		}),
	}
}

func TestRepairExponentNotation(t *testing.T) {
	t.Parallel()

	t.Run("should rewrite the captured id positionally when state holds an exponent", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "803554429", model.ResponseBodyID.ValueString())
	})

	t.Run("should rewrite the resolved delete path so destroy addresses a real id", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "/posts/803554429", model.DeleteResolvedPath.ValueString())
	})

	t.Run("should rebuild the response body map when the body is flagged as JSON", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		rebuilt := model.ResponseBodyJSON.Elements()
		assert.Equal(t, types.StringValue("803554429"), rebuilt["id"])
		assert.Equal(t, types.StringValue("example"), rebuilt["title"])
		assert.Equal(t, types.StringValue("60"), rebuilt["views"])
	})

	t.Run("should leave state untouched when no captured number changed rendering", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		model.ResponseBody = types.StringValue(`{"id":27164,"title":"group"}`)
		model.ResponseBodyID = types.StringValue("27164")
		model.DeleteResolvedPath = types.StringValue("/groups/27164")
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "27164", model.ResponseBodyID.ValueString())
		assert.Equal(t, "/groups/27164", model.DeleteResolvedPath.ValueString())
	})

	t.Run("should leave state untouched when the recorded response body is not JSON", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		model.ResponseBody = types.StringValue("Created")
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "8.03554429e+08", model.ResponseBodyID.ValueString())
		assert.Equal(t, "/posts/8.03554429e+08", model.DeleteResolvedPath.ValueString())
	})

	t.Run("should leave state untouched when no response body was recorded", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		model.ResponseBody = types.StringNull()
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "8.03554429e+08", model.ResponseBodyID.ValueString())
	})

	t.Run("should not substitute an id that only contains the old rendering as a substring", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		model.ResponseBodyID = types.StringValue("prefix-8.03554429e+08")
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "prefix-8.03554429e+08", model.ResponseBodyID.ValueString())
	})

	t.Run("should rewrite an id nested inside the recorded response body", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		model.ResponseBody = types.StringValue(`{"data":{"post":{"id":803554429}}}`)
		model.ResponseBodyIDFilter = types.StringValue("$.data.post.id")
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "803554429", model.ResponseBodyID.ValueString())
	})

	t.Run("should rewrite an id carried inside a response body array", func(t *testing.T) {
		t.Parallel()

		// given
		model := baselineUpgradeModel()
		model.ResponseBody = types.StringValue(`{"data":[{"id":803554429}]}`)
		var diagnostics diag.Diagnostics

		// when
		provider.RepairExponentNotation(context.Background(), &model, &diagnostics)

		// then
		assert.False(t, diagnostics.HasError())
		assert.Equal(t, "803554429", model.ResponseBodyID.ValueString())
	})
}

func TestGetHTTPRequestResourceSchemaVersion(t *testing.T) {
	t.Parallel()

	t.Run("should declare version 2 so the captured-number repair runs once", func(t *testing.T) {
		t.Parallel()

		// given
		// the schema as the provider advertises it

		// when
		result := provider.GetHTTPRequestResourceSchema()

		// then
		assert.Equal(t, int64(2), result.Version)
	})
}
