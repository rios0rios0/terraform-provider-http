package helpers

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func StringAttribute(required bool, description string) schema.StringAttribute {
	attribute := schema.StringAttribute{
		Description:         description,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
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
		PlanModifiers: []planmodifier.Map{
			mapplanmodifier.RequiresReplace(),
		},
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
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.RequiresReplace(),
		},
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
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func ComputedInt32Attribute(description string) schema.Int32Attribute {
	return schema.Int32Attribute{
		Computed:            true,
		Description:         description,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.Int32{
			int32planmodifier.UseStateForUnknown(),
		},
	}
}

func ComputedMapAttribute(elementType attr.Type, description string) schema.MapAttribute {
	return schema.MapAttribute{
		Computed:            true,
		ElementType:         elementType,
		Description:         description,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.Map{
			mapplanmodifier.UseStateForUnknown(),
		},
	}
}
