//go:build unit || integration

package provider

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
)

// clientWithLogs builds the client for the model against a provider whose ignore_tls is the given
// value and returns it together with the log entries getHTTPClient wrote.
func clientWithLogs(
	t *testing.T,
	providerIgnoreTLS bool,
	resourceIgnoreTLS types.Bool,
) (*http.Client, []map[string]any) {
	t.Helper()

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)
	it := &HTTPRequestResource{
		internal: entities.NewInternalContext(providerIgnoreTLS, entities.NewConfiguration("")),
	}
	model := HTTPRequestResourceModel{
		IgnoreTLS:        resourceIgnoreTLS,
		RequestTimeoutMs: types.Int64Null(),
		Retry:            types.ObjectNull(retryObjectAttrTypes()),
	}

	client := it.getHTTPClient(ctx, model)

	entries, err := tflogtest.MultilineJSONDecode(&output)
	require.NoError(t, err, "the root logger writes one JSON document per entry")

	return client, entries
}

func TestGetHTTPClientIgnoreTLSWarning(t *testing.T) {
	t.Parallel()

	t.Run("should log a warning when the resource opts out of TLS verification", func(t *testing.T) {
		t.Parallel()

		// given: a provider that verifies TLS and a resource that overrides it for this request
		providerIgnoreTLS := false
		resourceIgnoreTLS := types.BoolValue(true)

		// when
		client, entries := clientWithLogs(t, providerIgnoreTLS, resourceIgnoreTLS)

		// then
		warns := warnEntries(entries)
		require.Len(t, warns, 1, "exactly one warning announces the skipped verification")
		assert.Equal(t, warnIgnoreTLSRequest, warns[0]["@message"])
		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok, "the request must still get the insecure transport it asked for")
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("should log a warning when ignore_tls is inherited from the provider", func(t *testing.T) {
		t.Parallel()

		// given: the resource says nothing, so the provider-level opt-out applies
		providerIgnoreTLS := true
		resourceIgnoreTLS := types.BoolNull()

		// when
		_, entries := clientWithLogs(t, providerIgnoreTLS, resourceIgnoreTLS)

		// then
		warns := warnEntries(entries)
		require.Len(t, warns, 1, "the inherited opt-out is announced for the request as well")
		assert.Equal(t, warnIgnoreTLSRequest, warns[0]["@message"])
	})

	t.Run("should not log a warning when TLS verification stays enabled", func(t *testing.T) {
		t.Parallel()

		// given: neither the provider nor the resource opts out
		providerIgnoreTLS := false
		resourceIgnoreTLS := types.BoolNull()

		// when
		client, entries := clientWithLogs(t, providerIgnoreTLS, resourceIgnoreTLS)

		// then
		assert.Empty(t, warnEntries(entries), "the secure default is silent")
		assert.Nil(t, client.Transport, "the default transport verifies certificates")
	})
}
