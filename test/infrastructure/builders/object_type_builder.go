package builders

import "github.com/hashicorp/terraform-plugin-go/tftypes"

type ObjectTypeBuilder struct {
	attributeTypes map[string]tftypes.Type
}

func NewObjectTypeBuilder() *ObjectTypeBuilder {
	return &ObjectTypeBuilder{
		attributeTypes: make(map[string]tftypes.Type),
	}
}

func (b *ObjectTypeBuilder) WithURL() *ObjectTypeBuilder {
	b.attributeTypes["url"] = tftypes.String
	return b
}

func (b *ObjectTypeBuilder) WithUsername() *ObjectTypeBuilder {
	if basicAuth, ok := b.attributeTypes["basic_auth"]; ok {
		//nolint:errcheck // no need to check since it's covered by the test
		basicAuthType := basicAuth.(tftypes.Object)
		basicAuthType.AttributeTypes["username"] = tftypes.String
		b.attributeTypes["basic_auth"] = basicAuthType
	} else {
		b.attributeTypes["basic_auth"] = tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"username": tftypes.String,
			},
		}
	}
	return b
}

func (b *ObjectTypeBuilder) WithPassword() *ObjectTypeBuilder {
	if basicAuth, ok := b.attributeTypes["basic_auth"]; ok {
		//nolint:errcheck // no need to check since it's covered by the test
		basicAuthType := basicAuth.(tftypes.Object)
		basicAuthType.AttributeTypes["password"] = tftypes.String
		b.attributeTypes["basic_auth"] = basicAuthType
	} else {
		b.attributeTypes["basic_auth"] = tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"password": tftypes.String,
			},
		}
	}
	return b
}

func (b *ObjectTypeBuilder) WithIgnoreTLS() *ObjectTypeBuilder {
	b.attributeTypes["ignore_tls"] = tftypes.Bool
	return b
}

func (b *ObjectTypeBuilder) Build() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: b.attributeTypes,
	}
}
