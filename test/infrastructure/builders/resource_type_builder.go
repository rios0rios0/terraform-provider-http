package builders

import "github.com/hashicorp/terraform-plugin-go/tftypes"

type ResourceTypeBuilder struct {
	attributeTypes map[string]tftypes.Type
}

func NewResourceTypeBuilder() *ResourceTypeBuilder {
	return &ResourceTypeBuilder{
		attributeTypes: make(map[string]tftypes.Type),
	}
}

func (b *ResourceTypeBuilder) WithMethod() *ResourceTypeBuilder {
	b.attributeTypes["method"] = tftypes.String
	return b
}

func (b *ResourceTypeBuilder) WithPath() *ResourceTypeBuilder {
	b.attributeTypes["path"] = tftypes.String
	return b
}

func (b *ResourceTypeBuilder) WithHeaders() *ResourceTypeBuilder {
	b.attributeTypes["headers"] = tftypes.Map{ElementType: tftypes.String}
	return b
}

func (b *ResourceTypeBuilder) WithRequestBody() *ResourceTypeBuilder {
	b.attributeTypes["request_body"] = tftypes.String
	return b
}

func (b *ResourceTypeBuilder) WithIsResponseBodyJSON() *ResourceTypeBuilder {
	b.attributeTypes["is_response_body_json"] = tftypes.Bool
	return b
}

func (b *ResourceTypeBuilder) WithResponseBodyIDFilter() *ResourceTypeBuilder {
	b.attributeTypes["response_body_id_filter"] = tftypes.String
	return b
}

func (b *ResourceTypeBuilder) WithQueryParameters() *ResourceTypeBuilder {
	b.attributeTypes["query_parameters"] = tftypes.Map{ElementType: tftypes.String}
	return b
}

func (b *ResourceTypeBuilder) WithID() *ResourceTypeBuilder {
	b.attributeTypes["id"] = tftypes.String
	return b
}

func (b *ResourceTypeBuilder) WithResponseCode() *ResourceTypeBuilder {
	b.attributeTypes["response_code"] = tftypes.Number
	return b
}

func (b *ResourceTypeBuilder) WithResponseBody() *ResourceTypeBuilder {
	b.attributeTypes["response_body"] = tftypes.String
	return b
}

func (b *ResourceTypeBuilder) WithResponseBodyID() *ResourceTypeBuilder {
	b.attributeTypes["response_body_id"] = tftypes.String
	return b
}

func (b *ResourceTypeBuilder) WithResponseBodyJSON() *ResourceTypeBuilder {
	b.attributeTypes["response_body_json"] = tftypes.Map{ElementType: tftypes.String}
	return b
}

func (b *ResourceTypeBuilder) Build() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: b.attributeTypes,
	}
}
