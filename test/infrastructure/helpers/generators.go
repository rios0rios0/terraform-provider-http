package helpers

import (
	"encoding/json"
	"fmt"
)

const (
	baseProviderConfig = `
	provider "http" {
	  url = "%s"
	  %s
	}`

	baseResourceConfig = `
	resource "http_request" "%s" {
	  method = "%s"
	  path   = "%s"
	  %s
	}`
)

func GenerateProviderConfig(url string, basicAuth, ignoreTLS bool) string {
	authConfig := ""
	if basicAuth {
		authConfig = `
		basic_auth = {
		  username = "anything"
		  password = "anything"
		}`
	}
	tlsConfig := ""
	if ignoreTLS {
		tlsConfig = `
		ignore_tls = true
		`
	}
	return fmt.Sprintf(baseProviderConfig, url, authConfig+tlsConfig)
}

func GenerateInvalidProviderConfig(url string, basicAuth bool, missingUsername, missingPassword bool) string {
	authConfig := ""
	if basicAuth {
		authConfig = `
		basic_auth = {`
		if !missingUsername {
			authConfig += `
		  username = "anything"`
		}
		if !missingPassword {
			authConfig += `
		  password = "anything"`
		}
		authConfig += `
		}`
	}
	return fmt.Sprintf(baseProviderConfig, url, authConfig)
}

func GenerateResourceConfig(
	name, method, path, requestBody, responseBodyIDFilter string, isResponseBodyJSON bool,
) string {
	bodyConfig := ""
	if requestBody != "" {
		body, _ := json.Marshal(requestBody)

		bodyConfig = fmt.Sprintf(`
		request_body = %s
		`, body)
	}
	filterConfig := ""
	if responseBodyIDFilter != "" {
		filterConfig = fmt.Sprintf(`
		response_body_id_filter = "%s"
		`, responseBodyIDFilter)
	}
	jsonConfig := ""
	if isResponseBodyJSON {
		jsonConfig = `
		is_response_body_json = true
		`
	}
	return fmt.Sprintf(baseResourceConfig, name, method, path, bodyConfig+filterConfig+jsonConfig)
}
