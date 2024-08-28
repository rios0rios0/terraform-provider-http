package examples

import _ "embed"

//go:embed providers.tf
var HTTPProviderExample string

//go:embed main.tf
var HTTPRequestResourceExample string

func GetHTTPProviderExample() string {
	return HTTPProviderExample
}

func GetHTTPRequestResourceExample() string {
	return HTTPRequestResourceExample
}
