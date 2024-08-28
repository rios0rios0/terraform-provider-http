package helpers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func RetrieveResourceCreateRequest(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, model any,
) bool {
	diags := req.Plan.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

func RetrieveResourceValidateConfigRequest(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse, model any,
) bool {
	diags := req.Config.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

func StringAttribute(required bool, description string) schema.StringAttribute {
	attribute := schema.StringAttribute{
		Description:         description,
		MarkdownDescription: description,
	}
	if required {
		attribute.Required = true
	} else {
		attribute.Optional = true
	}
	return attribute
}

func MapAttribute(required bool, elementType attr.Type, description string) schema.MapAttribute {
	attribute := schema.MapAttribute{
		ElementType:         elementType,
		Description:         description,
		MarkdownDescription: description,
	}
	if required {
		attribute.Required = true
	} else {
		attribute.Optional = true
	}
	return attribute
}

func BoolAttribute(required bool, description string) schema.BoolAttribute {
	attribute := schema.BoolAttribute{
		Description:         description,
		MarkdownDescription: description,
	}
	if required {
		attribute.Required = true
	} else {
		attribute.Optional = true
	}
	return attribute
}

func ComputedStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:            true,
		Description:         description,
		MarkdownDescription: description,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func ComputedInt32Attribute(description string) schema.Int32Attribute {
	return schema.Int32Attribute{
		Computed:            true,
		Description:         description,
		MarkdownDescription: description,
	}
}

func ComputedMapAttribute(elementType attr.Type, description string) schema.MapAttribute {
	return schema.MapAttribute{
		Computed:            true,
		ElementType:         elementType,
		Description:         description,
		MarkdownDescription: description,
	}
}
