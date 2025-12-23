package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rios0rios0/terraform-provider-http/internal/provider"
	"github.com/stretchr/testify/require"
)

func TestApplyIgnoreEntriesHeadersKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	planHeaders, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"X-Correlation-Id": "plan-value",
		"X-Other":          "static",
	})
	require.False(t, diags.HasError())

	stateHeaders, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"X-Correlation-Id": "state-value",
		"X-Other":          "static",
	})
	require.False(t, diags.HasError())

	plan := provider.HTTPRequestResourceModel{
		Headers: planHeaders,
	}
	state := provider.HTTPRequestResourceModel{
		Headers: stateHeaders,
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("headers", []string{"X-Correlation-Id"}),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.Headers.Equal(state.Headers))
}

func TestApplyIgnoreEntriesFullHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	planHeaders, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"X-Correlation-Id": "plan-value",
		"X-Other":          "plan-other",
	})
	require.False(t, diags.HasError())

	stateHeaders, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"X-Correlation-Id": "state-value",
		"X-Other":          "state-other",
	})
	require.False(t, diags.HasError())

	plan := provider.HTTPRequestResourceModel{
		Headers: planHeaders,
	}
	state := provider.HTTPRequestResourceModel{
		Headers: stateHeaders,
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("headers", nil), // ignore entire headers map
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.Headers.Equal(state.Headers))
}

func TestApplyIgnoreEntriesQueryParameters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	planParams, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"page":  "2",
		"limit": "10",
	})
	require.False(t, diags.HasError())

	stateParams, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"page":  "1",
		"limit": "10",
	})
	require.False(t, diags.HasError())

	plan := provider.HTTPRequestResourceModel{
		QueryParameters: planParams,
	}
	state := provider.HTTPRequestResourceModel{
		QueryParameters: stateParams,
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("query_parameters", []string{"page"}),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.QueryParameters.Equal(state.QueryParameters))
}

func TestApplyIgnoreEntriesRequestBodyPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"demo","metadata":{"trace_id":"plan","other":"keep"}}`),
	}
	state := provider.HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"demo","metadata":{"trace_id":"state","other":"keep"}}`),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("request_body", []string{"metadata", "trace_id"}),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.RequestBody.Equal(state.RequestBody))
}

func TestApplyIgnoreEntriesRequestBodyWithOtherDiff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"changed","metadata":{"trace_id":"plan","other":"keep"}}`),
	}
	state := provider.HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"original","metadata":{"trace_id":"state","other":"keep"}}`),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("request_body", []string{"metadata", "trace_id"}),
	}, &plan, &state, &diagnostics)

	require.False(t, changed, "other differences should keep the plan change")
	require.False(t, diagnostics.HasError())
	require.False(t, plan.RequestBody.Equal(state.RequestBody))
}

func TestApplyIgnoreEntriesFullRequestBody(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"new","body":"new body"}`),
	}
	state := provider.HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"old","body":"old body"}`),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("request_body", nil), // ignore entire body
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.RequestBody.Equal(state.RequestBody))
}

func TestApplyIgnoreEntriesMethod(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		Method: types.StringValue("PUT"),
	}
	state := provider.HTTPRequestResourceModel{
		Method: types.StringValue("POST"),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("method", nil),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.Method.Equal(state.Method))
}

func TestApplyIgnoreEntriesPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		Path: types.StringValue("/api/v2/resource"),
	}
	state := provider.HTTPRequestResourceModel{
		Path: types.StringValue("/api/v1/resource"),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("path", nil),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.Path.Equal(state.Path))
}

func TestApplyIgnoreEntriesBaseURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		BaseURL: types.StringValue("https://api.new.com"),
	}
	state := provider.HTTPRequestResourceModel{
		BaseURL: types.StringValue("https://api.old.com"),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("base_url", nil),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.BaseURL.Equal(state.BaseURL))
}

func TestApplyIgnoreEntriesIgnoreTLS(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		IgnoreTLS: types.BoolValue(true),
	}
	state := provider.HTTPRequestResourceModel{
		IgnoreTLS: types.BoolValue(false),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("ignore_tls", nil),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.IgnoreTLS.Equal(state.IgnoreTLS))
}

func TestApplyIgnoreEntriesIsResponseBodyJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		IsResponseBodyJSON: types.BoolValue(true),
	}
	state := provider.HTTPRequestResourceModel{
		IsResponseBodyJSON: types.BoolValue(false),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("is_response_body_json", nil),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.IsResponseBodyJSON.Equal(state.IsResponseBodyJSON))
}

func TestApplyIgnoreEntriesResponseBodyIDFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := provider.HTTPRequestResourceModel{
		ResponseBodyIDFilter: types.StringValue("$.data.id"),
	}
	state := provider.HTTPRequestResourceModel{
		ResponseBodyIDFilter: types.StringValue("$.id"),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("response_body_id_filter", nil),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.ResponseBodyIDFilter.Equal(state.ResponseBodyIDFilter))
}

// TestDeleteFieldsNotInSupportedIgnoreAttributes verifies that delete fields
// are NOT in the supportedIgnoreAttributes map because they should never
// trigger replacement (they use NoReplace schema modifiers instead).
func TestDeleteFieldsNotInSupportedIgnoreAttributes(t *testing.T) {
	t.Parallel()

	deleteFields := []string{
		"is_delete_enabled",
		"delete_method",
		"delete_path",
		"delete_headers",
		"delete_request_body",
	}

	supportedAttrs := provider.GetSupportedIgnoreAttributes()

	for _, field := range deleteFields {
		// capture range variable
		t.Run(field+" should not be in supportedIgnoreAttributes", func(t *testing.T) {
			t.Parallel()
			_, exists := supportedAttrs[field]
			require.False(t, exists,
				"field %q should NOT be in supportedIgnoreAttributes because it uses NoReplace schema modifier", field)
		})
	}
}

// TestParseIgnoreEntriesWarnsOnDeleteFields verifies that when users try to add
// delete fields to ignore_changes, they receive a warning since those fields
// never trigger replacement anyway.
func TestParseIgnoreEntriesWarnsOnDeleteFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	deleteFields := []string{
		"is_delete_enabled",
		"delete_method",
		"delete_path",
		"delete_headers",
		"delete_request_body",
	}

	ignoreSet, diags := types.SetValueFrom(ctx, types.StringType, deleteFields)
	require.False(t, diags.HasError())

	var diagnostics diag.Diagnostics
	entries := provider.ParseIgnoreEntries(ctx, ignoreSet, &diagnostics)

	// All delete fields should be rejected with warnings
	require.Empty(t, entries, "no entries should be returned for delete fields")
	require.Len(t, diagnostics, len(deleteFields), "should have a warning for each delete field")

	for _, d := range diagnostics {
		require.Equal(t, "Unsupported ignore_changes entry", d.Summary())
	}
}

// TestApplyIgnoreEntriesNoChangeWhenEqual verifies that no change is reported
// when plan and state values are already equal.
func TestApplyIgnoreEntriesNoChangeWhenEqual(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	headers, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"Content-Type": "application/json",
	})
	require.False(t, diags.HasError())

	plan := provider.HTTPRequestResourceModel{
		Method:  types.StringValue("GET"),
		Path:    types.StringValue("/api/resource"),
		Headers: headers,
	}
	state := provider.HTTPRequestResourceModel{
		Method:  types.StringValue("GET"),
		Path:    types.StringValue("/api/resource"),
		Headers: headers,
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("method", nil),
		provider.NewIgnoreEntry("path", nil),
		provider.NewIgnoreEntry("headers", nil),
	}, &plan, &state, &diagnostics)

	require.False(t, changed, "no change should be reported when values are equal")
	require.False(t, diagnostics.HasError())
}

// TestApplyIgnoreEntriesMultipleAttributes verifies that multiple attributes
// can be ignored in a single call.
func TestApplyIgnoreEntriesMultipleAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	planHeaders, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"X-Request-Id": "plan-id",
	})
	require.False(t, diags.HasError())

	stateHeaders, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"X-Request-Id": "state-id",
	})
	require.False(t, diags.HasError())

	plan := provider.HTTPRequestResourceModel{
		Method:      types.StringValue("PUT"),
		Path:        types.StringValue("/api/v2"),
		Headers:     planHeaders,
		RequestBody: types.StringValue(`{"new":"data"}`),
	}
	state := provider.HTTPRequestResourceModel{
		Method:      types.StringValue("POST"),
		Path:        types.StringValue("/api/v1"),
		Headers:     stateHeaders,
		RequestBody: types.StringValue(`{"old":"data"}`),
	}

	var diagnostics diag.Diagnostics
	changed := provider.ApplyIgnoreEntries(ctx, []provider.IgnoreEntry{
		provider.NewIgnoreEntry("method", nil),
		provider.NewIgnoreEntry("path", nil),
		provider.NewIgnoreEntry("headers", nil),
		provider.NewIgnoreEntry("request_body", nil),
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.Method.Equal(state.Method))
	require.True(t, plan.Path.Equal(state.Path))
	require.True(t, plan.Headers.Equal(state.Headers))
	require.True(t, plan.RequestBody.Equal(state.RequestBody))
}
