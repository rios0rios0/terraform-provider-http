package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rios0rios0/terraform-provider-http/internal/provider"
	"github.com/stretchr/testify/assert"
)

func basicAuthAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"username": types.StringType,
		"password": types.StringType,
	}
}

// baselineReissueModel returns a fully-populated model used as the prior state in
// the RequestAttributesChanged tests. Each test copies it and mutates a single
// attribute, so the predicate is exercised one attribute at a time.
func baselineReissueModel() provider.HTTPRequestResourceModel {
	return provider.HTTPRequestResourceModel{
		Method: types.StringValue("POST"),
		Path:   types.StringValue("/things"),
		Headers: types.MapValueMust(types.StringType, map[string]attr.Value{
			"Content-Type": types.StringValue("application/json"),
		}),
		RequestBody: types.StringValue(`{"a":1}`),
		QueryParameters: types.MapValueMust(types.StringType, map[string]attr.Value{
			"page": types.StringValue("1"),
		}),
		BaseURL:   types.StringValue("https://api.example.com"),
		BasicAuth: types.ObjectNull(basicAuthAttrTypes()),
		IgnoreTLS: types.BoolValue(false),

		ToleratedStatusCodes: types.SetNull(types.Int32Type),
		ResponseBodyIDFilter: types.StringValue("$.id"),
		IsResponseBodyJSON:   types.BoolValue(true),
		IgnoreChanges:        types.SetNull(types.StringType),

		ID:                 types.StringValue("captured-id"),
		ResponseCode:       types.Int32Value(200),
		ResponseBody:       types.StringValue(`{"id":"1"}`),
		ResponseBodyID:     types.StringValue("1"),
		DeleteResolvedPath: types.StringNull(),
	}
}

func TestRequestAttributesChanged(t *testing.T) {
	t.Parallel()

	t.Run("should report no change when the plan equals the prior state", func(t *testing.T) {
		t.Parallel()

		// given
		state := baselineReissueModel()
		plan := baselineReissueModel()

		// when
		changed := provider.RequestAttributesChanged(plan, state)

		// then
		assert.False(t, changed, "identical models must not trigger a re-issue")
	})

	t.Run("should report a change when a request-defining attribute differs", func(t *testing.T) {
		t.Parallel()

		// given
		state := baselineReissueModel()
		mutations := map[string]func(m *provider.HTTPRequestResourceModel){
			"method": func(m *provider.HTTPRequestResourceModel) {
				m.Method = types.StringValue("PUT")
			},
			"path": func(m *provider.HTTPRequestResourceModel) {
				m.Path = types.StringValue("/other")
			},
			"headers": func(m *provider.HTTPRequestResourceModel) {
				m.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{
					"X-New": types.StringValue("y"),
				})
			},
			"request_body": func(m *provider.HTTPRequestResourceModel) {
				m.RequestBody = types.StringValue(`{"a":2}`)
			},
			"query_parameters": func(m *provider.HTTPRequestResourceModel) {
				m.QueryParameters = types.MapValueMust(types.StringType, map[string]attr.Value{
					"page": types.StringValue("2"),
				})
			},
			"base_url": func(m *provider.HTTPRequestResourceModel) {
				m.BaseURL = types.StringValue("https://api.other.com")
			},
			"basic_auth": func(m *provider.HTTPRequestResourceModel) {
				m.BasicAuth = types.ObjectValueMust(basicAuthAttrTypes(), map[string]attr.Value{
					"username": types.StringValue("u"),
					"password": types.StringValue("p"),
				})
			},
			"ignore_tls": func(m *provider.HTTPRequestResourceModel) {
				m.IgnoreTLS = types.BoolValue(true)
			},
		}

		for name, mutate := range mutations {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				// given
				plan := baselineReissueModel()
				mutate(&plan)

				// when
				changed := provider.RequestAttributesChanged(plan, state)

				// then
				assert.True(t, changed, "changing %s must trigger a re-issue", name)
			})
		}
	})

	t.Run("should report no change when only a client-side or computed attribute differs", func(t *testing.T) {
		t.Parallel()

		// given
		state := baselineReissueModel()
		mutations := map[string]func(m *provider.HTTPRequestResourceModel){
			"tolerated_status_codes": func(m *provider.HTTPRequestResourceModel) {
				m.ToleratedStatusCodes = types.SetValueMust(types.Int32Type, []attr.Value{
					types.Int32Value(404),
				})
			},
			"response_body_id_filter": func(m *provider.HTTPRequestResourceModel) {
				m.ResponseBodyIDFilter = types.StringValue("$.data.id")
			},
			"is_response_body_json": func(m *provider.HTTPRequestResourceModel) {
				m.IsResponseBodyJSON = types.BoolValue(false)
			},
			"ignore_changes": func(m *provider.HTTPRequestResourceModel) {
				m.IgnoreChanges = types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("headers"),
				})
			},
			"id": func(m *provider.HTTPRequestResourceModel) {
				m.ID = types.StringValue("different")
			},
			"response_body": func(m *provider.HTTPRequestResourceModel) {
				m.ResponseBody = types.StringValue(`{"id":"2"}`)
			},
			"response_body_id": func(m *provider.HTTPRequestResourceModel) {
				m.ResponseBodyID = types.StringValue("2")
			},
			"delete_resolved_path": func(m *provider.HTTPRequestResourceModel) {
				m.DeleteResolvedPath = types.StringValue("/things/2")
			},
		}

		for name, mutate := range mutations {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				// given
				plan := baselineReissueModel()
				mutate(&plan)

				// when
				changed := provider.RequestAttributesChanged(plan, state)

				// then
				assert.False(t, changed, "changing %s must NOT trigger a re-issue", name)
			})
		}
	})
}
