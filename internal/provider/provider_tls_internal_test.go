package provider

import (
	"bytes"
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// warnEntries keeps the entries a tflogtest root logger wrote at the WARN level.
func warnEntries(entries []map[string]any) []map[string]any {
	var warns []map[string]any
	for _, entry := range entries {
		if entry["@level"] == "warn" {
			warns = append(warns, entry)
		}
	}

	return warns
}

// configureWithIgnoreTLS runs Configure over a provider configuration whose `ignore_tls` is the given
// value and returns the log entries the call wrote together with its diagnostics.
func configureWithIgnoreTLS(t *testing.T, ignoreTLS tftypes.Value) ([]map[string]any, diag.Diagnostics) {
	t.Helper()

	values := fullProviderValues(
		tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
		nullBasicAuthValue(),
	)
	values["ignore_tls"] = ignoreTLS
	raw := tftypes.NewValue(fullProviderType(), values)

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)
	req := provider.ConfigureRequest{Config: tfsdk.Config{Raw: raw, Schema: GetHTTPProviderSchema()}}
	resp := provider.ConfigureResponse{Diagnostics: make(diag.Diagnostics, 0)}

	it := &HTTPProvider{}
	it.Configure(ctx, req, &resp)

	entries, err := tflogtest.MultilineJSONDecode(&output)
	require.NoError(t, err, "the root logger writes one JSON document per entry")

	return entries, resp.Diagnostics
}

func TestHTTPProviderConfigureIgnoreTLS(t *testing.T) {
	t.Parallel()

	t.Run("should log a warning when ignore_tls is enabled", func(t *testing.T) {
		t.Parallel()

		// given
		ignoreTLS := tftypes.NewValue(tftypes.Bool, true)

		// when
		entries, diagnostics := configureWithIgnoreTLS(t, ignoreTLS)

		// then
		require.False(t, diagnostics.HasError(), "opting out of verification is a valid configuration")
		warns := warnEntries(entries)
		require.Len(t, warns, 1, "exactly one warning announces the disabled verification")
		assert.Equal(t, warnIgnoreTLSProvider, warns[0]["@message"])
		assert.Equal(t, true, warns[0]["http_ignore_tls"], "the structured field mirrors the flag")
	})

	t.Run("should not log a warning when ignore_tls is left unset", func(t *testing.T) {
		t.Parallel()

		// given
		ignoreTLS := tftypes.NewValue(tftypes.Bool, nil)

		// when
		entries, diagnostics := configureWithIgnoreTLS(t, ignoreTLS)

		// then
		require.False(t, diagnostics.HasError())
		assert.Empty(t, warnEntries(entries), "the default keeps verification on, so there is nothing to warn about")
	})

	t.Run("should not log a warning when ignore_tls is explicitly false", func(t *testing.T) {
		t.Parallel()

		// given
		ignoreTLS := tftypes.NewValue(tftypes.Bool, false)

		// when
		entries, diagnostics := configureWithIgnoreTLS(t, ignoreTLS)

		// then
		require.False(t, diagnostics.HasError())
		assert.Empty(t, warnEntries(entries), "an explicit false keeps verification on")
	})
}
