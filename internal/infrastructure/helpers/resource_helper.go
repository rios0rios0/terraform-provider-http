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

// StringAttributeNoReplace creates an optional/required string attribute that does NOT trigger replacement on change.
// Use this for fields that only affect behavior during destroy or other non-create operations.
func StringAttributeNoReplace(required bool, description string) schema.StringAttribute {
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

// MapAttributeNoReplace creates an optional/required map attribute that does NOT trigger replacement on change.
// Use this for fields that only affect behavior during destroy or other non-create operations.
func MapAttributeNoReplace(required bool, elementType attr.Type, description string) schema.MapAttribute {
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

// BoolAttributeNoReplace creates an optional/required bool attribute that does NOT trigger replacement on change.
// Use this for fields that only affect behavior during destroy or other non-create operations.
func BoolAttributeNoReplace(required bool, description string) schema.BoolAttribute {
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

// StringAttributeWriteOnly creates a write-only string attribute that is never stored in state or shown in plan diffs.
// Use this for fields that are only needed during destroy-time and should not trigger plan diffs when changed.
// Requires Terraform 1.11+.
func StringAttributeWriteOnly(required bool, description string) schema.StringAttribute {
	attribute := schema.StringAttribute{
		Description:         description,
		MarkdownDescription: description,
		WriteOnly:           true,
	}
	if required {
		attribute.Required = true
	} else {
		attribute.Optional = true
	}
	return attribute
}

// MapAttributeWriteOnly creates a write-only map attribute that is never stored in state or shown in plan diffs.
// Use this for fields that are only needed during destroy-time and should not trigger plan diffs when changed.
// Requires Terraform 1.11+.
func MapAttributeWriteOnly(required bool, elementType attr.Type, description string) schema.MapAttribute {
	attribute := schema.MapAttribute{
		ElementType:         elementType,
		Description:         description,
		MarkdownDescription: description,
		WriteOnly:           true,
	}
	if required {
		attribute.Required = true
	} else {
		attribute.Optional = true
	}
	return attribute
}

// BoolAttributeWriteOnly creates a write-only bool attribute that is never stored in state or shown in plan diffs.
// Use this for fields that are only needed during destroy-time and should not trigger plan diffs when changed.
// Requires Terraform 1.11+.
func BoolAttributeWriteOnly(required bool, description string) schema.BoolAttribute {
	attribute := schema.BoolAttribute{
		Description:         description,
		MarkdownDescription: description,
		WriteOnly:           true,
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
