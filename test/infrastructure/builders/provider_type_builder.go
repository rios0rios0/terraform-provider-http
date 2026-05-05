package builders

import "github.com/hashicorp/terraform-plugin-go/tftypes"

const (
	attrBasicAuth = "basic_auth"
	attrIgnoreTLS = "ignore_tls"
	attrUsername  = "username"
	attrPassword  = "password"
)

type ProviderTypeBuilder struct {
	attributeTypes map[string]tftypes.Type
}

func NewProviderTypeBuilder() *ProviderTypeBuilder {
	return &ProviderTypeBuilder{
		attributeTypes: make(map[string]tftypes.Type),
	}
}

func (b *ProviderTypeBuilder) WithURL() *ProviderTypeBuilder {
	b.attributeTypes["url"] = tftypes.String
	return b
}

func (b *ProviderTypeBuilder) WithUsername() *ProviderTypeBuilder {
	if basicAuth, ok := b.attributeTypes[attrBasicAuth]; ok {
		//nolint:errcheck // no need to check since it's covered by the test
		basicAuthType := basicAuth.(tftypes.Object)
		basicAuthType.AttributeTypes[attrUsername] = tftypes.String
		b.attributeTypes[attrBasicAuth] = basicAuthType
	} else {
		b.attributeTypes[attrBasicAuth] = tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				attrUsername: tftypes.String,
			},
		}
	}
	return b
}

func (b *ProviderTypeBuilder) WithPassword() *ProviderTypeBuilder {
	if basicAuth, ok := b.attributeTypes[attrBasicAuth]; ok {
		//nolint:errcheck // no need to check since it's covered by the test
		basicAuthType := basicAuth.(tftypes.Object)
		basicAuthType.AttributeTypes[attrPassword] = tftypes.String
		b.attributeTypes[attrBasicAuth] = basicAuthType
	} else {
		b.attributeTypes[attrBasicAuth] = tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				attrPassword: tftypes.String,
			},
		}
	}
	return b
}

func (b *ProviderTypeBuilder) WithIgnoreTLS() *ProviderTypeBuilder {
	b.attributeTypes[attrIgnoreTLS] = tftypes.Bool
	return b
}

func (b *ProviderTypeBuilder) Build() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: b.attributeTypes,
	}
}
