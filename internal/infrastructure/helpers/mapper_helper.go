package helpers

import "fmt"

func ConvertToStringMap(input map[string]interface{}) map[string]string {
	stringMap := make(map[string]string)
	for key, value := range input {
		switch v := value.(type) {
		case map[string]interface{}:
			nestedMap := ConvertToStringMap(v)
			for nestedKey, nestedValue := range nestedMap {
				stringMap[fmt.Sprintf("%s.%s", key, nestedKey)] = nestedValue
			}
		default:
			stringMap[key] = fmt.Sprintf("%v", value)
		}
	}
	return stringMap
}
