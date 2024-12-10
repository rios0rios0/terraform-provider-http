package builders

import "fmt"

const (
	baseProviderTF = `
		provider "http" {
		  %s
		}
	`
)

type ProviderTFBuilder struct {
	config string
}

func NewProviderTFBuilder() *ProviderTFBuilder {
	return &ProviderTFBuilder{}
}

func (b *ProviderTFBuilder) WithURL(url string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("url = \"%s\"\n", url)
	return b
}

func (b *ProviderTFBuilder) WithUsername(username string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("basic_auth = {\n  username = \"%s\"\n}\n", username)
	return b
}

func (b *ProviderTFBuilder) WithPassword(password string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("basic_auth = {\n  password = \"%s\"\n}\n", password)
	return b
}

func (b *ProviderTFBuilder) WithBasicAuth(username, password string) *ProviderTFBuilder {
	b.config += fmt.Sprintf("basic_auth = {\n  username = \"%s\"\n  password = \"%s\"\n}\n", username, password)
	return b
}

func (b *ProviderTFBuilder) WithIgnoreTLS(ignoreTLS bool) *ProviderTFBuilder {
	b.config += fmt.Sprintf("ignore_tls = %t\n", ignoreTLS)
	return b
}

func (b *ProviderTFBuilder) Build() string {
	return fmt.Sprintf(baseProviderTF, b.config)
}
