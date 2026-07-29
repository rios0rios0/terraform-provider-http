package helpers

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// MaxExactJSONInteger is the largest magnitude (2^53) that a float64 represents exactly.
// Above it, consecutive integers are no longer distinguishable, so rendering such a value
// in positional notation would invent digits that were never in the payload.
const MaxExactJSONInteger = 1 << 53

func ConvertToStringMap(input map[string]any) map[string]string {
	stringMap := make(map[string]string)
	for key, value := range input {
		switch v := value.(type) {
		case map[string]any:
			nestedMap := ConvertToStringMap(v)
			for nestedKey, nestedValue := range nestedMap {
				stringMap[fmt.Sprintf("%s.%s", key, nestedKey)] = nestedValue
			}
		default:
			stringMap[key] = FormatJSONScalar(value)
		}
	}
	return stringMap
}

// FormatJSONScalar renders a value decoded from a JSON document as a string, keeping whole
// numbers in positional notation.
//
// `encoding/json` decodes every JSON number into a float64, and the `%v` verb formats a
// float64 with `%g`, which switches to scientific notation once the exponent grows. An
// identifier such as 803554429 is therefore rendered as "8.03554429e+08" -- a string that no
// longer works where the original number did: it cannot be interpolated into a URL path,
// compared against the upstream value, or parsed by a strict consumer.
//
// Whole numbers within the exactly-representable range are formatted with the 'f' verb and
// shortest-round-trip precision, which never emits an exponent. Every other value (including
// fractional numbers, JSON null, booleans and strings) keeps the previous `%v` rendering, so
// values already recorded in state stay byte-identical.
func FormatJSONScalar(value any) string {
	switch typed := value.(type) {
	case json.Number:
		// Only produced when the caller opts into json.Decoder.UseNumber; the original
		// literal is already exact, so hand it back untouched.
		return typed.String()
	case float64:
		return formatJSONFloat(typed)
	case float32:
		return formatJSONFloat(float64(typed))
	default:
		return fmt.Sprintf("%v", value)
	}
}

func formatJSONFloat(value float64) string {
	// NaN and both infinities fail the equality or the range check and fall through to %v,
	// which spells them out instead of pretending they are digits.
	if value == math.Trunc(value) && math.Abs(value) <= MaxExactJSONInteger {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", value)
}
