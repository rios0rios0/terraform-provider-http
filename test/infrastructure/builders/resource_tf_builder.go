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

func (b *ResourceTFBuilder) Build() string {
	return fmt.Sprintf(baseResourceTF, b.config)
}
