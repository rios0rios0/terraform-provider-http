package builders

import (
	"fmt"
)

const (
	baseResourceTF = `
		%s
	}
	`
)

type ResourceTFBuilder struct {
	config string
}

func NewResourceTFBuilder() *ResourceTFBuilder {
	return &ResourceTFBuilder{}
}

func (b *ResourceTFBuilder) WithName(name string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("resource \"http_request\" \"%s\" {\n", name)
	return b
}

func (b *ResourceTFBuilder) WithMethod(method string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("method = \"%s\"\n", method)
	return b
}

func (b *ResourceTFBuilder) WithPath(path string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("path = \"%s\"\n", path)
	return b
}

func (b *ResourceTFBuilder) WithHeaders(headers map[string]string) *ResourceTFBuilder {
	b.config += "headers = {\n"
	for key, value := range headers {
		b.config += fmt.Sprintf("  %s = \"%s\"\n", key, value)
	}
	b.config += "}\n"
	return b
}

func (b *ResourceTFBuilder) WithRequestBody(requestBody string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("request_body = %s \n", requestBody)
	return b
}

func (b *ResourceTFBuilder) WithResponseBodyIDFilter(responseBodyIDFilter string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("response_body_id_filter = \"%s\"\n", responseBodyIDFilter)
	return b
}

func (b *ResourceTFBuilder) WithIsResponseBodyJSON(isResponseBodyJSON bool) *ResourceTFBuilder {
	b.config += fmt.Sprintf("is_response_body_json = %t\n", isResponseBodyJSON)
	return b
}

func (b *ResourceTFBuilder) WithQueryParameters(queryParameters map[string]string) *ResourceTFBuilder {
	b.config += "query_parameters = {\n"
	for key, value := range queryParameters {
		b.config += fmt.Sprintf("  %s = \"%s\"\n", key, value)
	}
	b.config += "}\n"
	return b
}

// WithBaseURL adds a base URL to the resource configuration.
func (b *ResourceTFBuilder) WithBaseURL(baseURL string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("base_url = \"%s\"\n", baseURL)
	return b
}

func (b *ResourceTFBuilder) WithBasicAuth(username, password string) *ResourceTFBuilder {
	b.config += "basic_auth = {\n"
	b.config += fmt.Sprintf("  username = \"%s\"\n", username)
	b.config += fmt.Sprintf("  password = \"%s\"\n", password)
	b.config += "}\n"
	return b
}

func (b *ResourceTFBuilder) WithIgnoreTLS(ignoreTLS bool) *ResourceTFBuilder {
	b.config += fmt.Sprintf("ignore_tls = %t\n", ignoreTLS)
	return b
}

func (b *ResourceTFBuilder) WithIsDeleteEnabled(isDeleteEnabled bool) *ResourceTFBuilder {
	b.config += fmt.Sprintf("is_delete_enabled = %t\n", isDeleteEnabled)
	return b
}

func (b *ResourceTFBuilder) WithDeleteMethod(deleteMethod string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("delete_method = \"%s\"\n", deleteMethod)
	return b
}

func (b *ResourceTFBuilder) WithDeletePath(deletePath string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("delete_path = \"%s\"\n", deletePath)
	return b
}

func (b *ResourceTFBuilder) WithDeleteHeaders(deleteHeaders map[string]string) *ResourceTFBuilder {
	b.config += "delete_headers = {\n"
	for key, value := range deleteHeaders {
		b.config += fmt.Sprintf("  \"%s\" = \"%s\"\n", key, value)
	}
	b.config += "}\n"
	return b
}

func (b *ResourceTFBuilder) WithDeleteRequestBody(deleteRequestBody string) *ResourceTFBuilder {
	b.config += fmt.Sprintf("delete_request_body = %s\n", deleteRequestBody)
	return b
}

func (b *ResourceTFBuilder) Build() string {
	return fmt.Sprintf(baseResourceTF, b.config)
}
