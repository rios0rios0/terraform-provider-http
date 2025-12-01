package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	plan := HTTPRequestResourceModel{
		Headers: planHeaders,
	}
	state := HTTPRequestResourceModel{
		Headers: stateHeaders,
	}

	var diagnostics diag.Diagnostics
	changed := applyIgnoreEntries(ctx, []ignoreEntry{
		{attribute: "headers", subPath: []string{"X-Correlation-Id"}},
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.Headers.Equal(state.Headers))
}

func TestApplyIgnoreEntriesRequestBodyPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"demo","metadata":{"trace_id":"plan","other":"keep"}}`),
	}
	state := HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"demo","metadata":{"trace_id":"state","other":"keep"}}`),
	}

	var diagnostics diag.Diagnostics
	changed := applyIgnoreEntries(ctx, []ignoreEntry{
		{attribute: "request_body", subPath: []string{"metadata", "trace_id"}},
	}, &plan, &state, &diagnostics)

	require.True(t, changed)
	require.False(t, diagnostics.HasError())
	require.True(t, plan.RequestBody.Equal(state.RequestBody))
}

func TestApplyIgnoreEntriesRequestBodyWithOtherDiff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"changed","metadata":{"trace_id":"plan","other":"keep"}}`),
	}
	state := HTTPRequestResourceModel{
		RequestBody: types.StringValue(`{"title":"original","metadata":{"trace_id":"state","other":"keep"}}`),
	}

	var diagnostics diag.Diagnostics
	changed := applyIgnoreEntries(ctx, []ignoreEntry{
		{attribute: "request_body", subPath: []string{"metadata", "trace_id"}},
	}, &plan, &state, &diagnostics)

	require.False(t, changed, "other differences should keep the plan change")
	require.False(t, diagnostics.HasError())
	require.False(t, plan.RequestBody.Equal(state.RequestBody))
}
