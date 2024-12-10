package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type StringNotEmpty struct {
	fieldName string
}

func NewStringNotEmpty(fieldName string) *StringNotEmpty {
	return &StringNotEmpty{
		fieldName: fieldName,
	}
}

func (it *StringNotEmpty) Description(context.Context) string {
	return fmt.Sprintf("'%s' value must not be empty.", it.fieldName)
}

func (it *StringNotEmpty) MarkdownDescription(context.Context) string {
	return fmt.Sprintf("'%s' value must not be empty.", it.fieldName)
}

func (it *StringNotEmpty) ValidateString(
	ctx context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	value, diags := req.ConfigValue.ToStringValue(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: is this the correct way to check for TF validators?
	// this is just appending the error to the diagnostics and showing them at the end
	if len(value.ValueString()) == 0 {
		resp.Diagnostics.AddError(it.Description(ctx), "")
	}
}
