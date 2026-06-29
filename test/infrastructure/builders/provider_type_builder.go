package builders

import "github.com/hashicorp/terraform-plugin-go/tftypes"

const (
	attrBasicAuth        = "basic_auth"
	attrIgnoreTLS        = "ignore_tls"
	attrUsername         = "username"
	attrPassword         = "password"
	attrRequestTimeoutMs = "request_timeout_ms"
	attrRetry            = "retry"
	attrAttempts         = "attempts"
	attrMinDelayMs       = "min_delay_ms"
	attrMaxDelayMs       = "max_delay_ms"
)

// retryObjectType is the tftypes shape of the `retry` nested block, shared by the
// provider and resource type builders.
func retryObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			attrAttempts:   tftypes.Number,
			attrMinDelayMs: tftypes.Number,
			attrMaxDelayMs: tftypes.Number,
		},
	}
}

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

func (b *ProviderTypeBuilder) WithRequestTimeoutMs() *ProviderTypeBuilder {
	b.attributeTypes[attrRequestTimeoutMs] = tftypes.Number
	return b
}

func (b *ProviderTypeBuilder) WithRetry() *ProviderTypeBuilder {
	b.attributeTypes[attrRetry] = retryObjectType()
	return b
}

func (b *ProviderTypeBuilder) Build() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: b.attributeTypes,
	}
}
