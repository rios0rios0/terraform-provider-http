package builders

import "github.com/hashicorp/terraform-plugin-go/tftypes"

type ProviderValueBuilder struct {
	values map[string]tftypes.Value
}

func NewProviderValueBuilder() *ProviderValueBuilder {
	return &ProviderValueBuilder{
		values: make(map[string]tftypes.Value),
	}
}

func (b *ProviderValueBuilder) WithURL(url string) *ProviderValueBuilder {
	b.values["url"] = tftypes.NewValue(tftypes.String, url)
	return b
}

func (b *ProviderValueBuilder) WithUsername(username string) *ProviderValueBuilder {
	if basicAuth, ok := b.values["basic_auth"]; ok {
		steps := tftypes.NewAttributePath().
			WithAttributeName("username").
			WithElementKeyValue(tftypes.NewValue(tftypes.String, username)).LastStep()
		value, _ := basicAuth.ApplyTerraform5AttributePathStep(steps)
		b.values["basic_auth"] = tftypes.NewValue(basicAuth.Type(), value)
	} else {
		b.values["basic_auth"] = tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"username": tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"username": tftypes.NewValue(tftypes.String, username),
			},
		)
	}
	return b
}

func (b *ProviderValueBuilder) WithPassword(password string) *ProviderValueBuilder {
	if basicAuth, ok := b.values["basic_auth"]; ok {
		steps := tftypes.NewAttributePath().
			WithAttributeName("password").
			WithElementKeyValue(tftypes.NewValue(tftypes.String, password)).LastStep()
		value, _ := basicAuth.ApplyTerraform5AttributePathStep(steps)
		b.values["basic_auth"] = tftypes.NewValue(basicAuth.Type(), value)
	} else {
		b.values["basic_auth"] = tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"password": tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"password": tftypes.NewValue(tftypes.String, password),
			},
		)
	}
	return b
}

func (b *ProviderValueBuilder) WithIgnoreTLS(ignore bool) *ProviderValueBuilder {
	b.values["ignore_tls"] = tftypes.NewValue(tftypes.Bool, ignore)
	return b
}

func (b *ProviderValueBuilder) Build() map[string]tftypes.Value {
	return b.values
}

// Example of raw tftypes.Value construction for reference
// This direct approach could be replaced with the fluent builder pattern below
// Fluent API builder pattern for improved test readability (planned enhancement)
// Example usage that could replace the current builder implementation
// 	Build(),
