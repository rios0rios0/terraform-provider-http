package helpers

import "net/http"

func IsResponseSuccessful(response *http.Response) bool {
	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}
