package helpers_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/rios0rios0/terraform-provider-http/internal/infrastructure/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatJSONScalar(t *testing.T) {
	t.Parallel()

	t.Run("should render a whole number in positional notation when %g would use an exponent", func(t *testing.T) {
		t.Parallel()

		// given
		// the float64 encoding/json produces for the JSON number 803554429
		value := any(803554429.0)

		// when
		result := helpers.FormatJSONScalar(value)

		// then
		assert.Equal(t, "803554429", result)
	})

	t.Run("should drop no significant digit when the whole number ends in zero", func(t *testing.T) {
		t.Parallel()

		// given
		value := any(803554430.0)

		// when
		result := helpers.FormatJSONScalar(value)

		// then
		assert.Equal(t, "803554430", result)
	})

	t.Run("should keep a small whole number unchanged", func(t *testing.T) {
		t.Parallel()

		// given
		value := any(27164.0)

		// when
		result := helpers.FormatJSONScalar(value)

		// then
		assert.Equal(t, "27164", result)
	})

	t.Run("should keep a fractional number rendered as before", func(t *testing.T) {
		t.Parallel()

		// given
		value := any(1.5)

		// when
		result := helpers.FormatJSONScalar(value)

		// then
		assert.Equal(t, "1.5", result)
	})

	t.Run("should keep a whole number beyond exact float64 range rendered as before", func(t *testing.T) {
		t.Parallel()

		// given
		// past 2^53 consecutive integers are indistinguishable, so positional
		// notation would invent digits that were never in the payload
		value := any(math.Pow(2, 60))

		// when
		result := helpers.FormatJSONScalar(value)

		// then
		assert.Equal(t, "1.152921504606847e+18", result)
	})

	t.Run("should return the original literal when the number was decoded with UseNumber", func(t *testing.T) {
		t.Parallel()

		// given
		value := any(json.Number("803554429"))

		// when
		result := helpers.FormatJSONScalar(value)

		// then
		assert.Equal(t, "803554429", result)
	})

	t.Run("should keep non-numeric values rendered as before", func(t *testing.T) {
		t.Parallel()

		// given
		cases := map[string]struct {
			value    any
			expected string
		}{
			"string":    {value: "already-a-string", expected: "already-a-string"},
			"boolean":   {value: true, expected: "true"},
			"json null": {value: nil, expected: "<nil>"},
		}

		for name, testCase := range cases {
			// when
			result := helpers.FormatJSONScalar(testCase.value)

			// then
			assert.Equal(t, testCase.expected, result, name)
		}
	})
}

func TestConvertToStringMap(t *testing.T) {
	t.Parallel()

	t.Run("should render an identifier positionally when the response body is decoded", func(t *testing.T) {
		t.Parallel()

		// given
		var decoded map[string]any
		err := json.Unmarshal([]byte(`{"id":803554429,"title":"example"}`), &decoded)
		require.NoError(t, err)

		// when
		result := helpers.ConvertToStringMap(decoded)

		// then
		assert.Equal(t, "803554429", result["id"])
		assert.Equal(t, "example", result["title"])
	})

	t.Run("should flatten a nested object using dot notation", func(t *testing.T) {
		t.Parallel()

		// given
		var decoded map[string]any
		err := json.Unmarshal([]byte(`{"data":{"id":803554429,"nested":{"count":2}}}`), &decoded)
		require.NoError(t, err)

		// when
		result := helpers.ConvertToStringMap(decoded)

		// then
		assert.Equal(t, "803554429", result["data.id"])
		assert.Equal(t, "2", result["data.nested.count"])
	})
}
