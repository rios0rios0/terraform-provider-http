package helpers

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type ResourceHelper struct {
}

func NewResourceHelper() *ResourceHelper {
	return &ResourceHelper{}
}

func (it *ResourceHelper) RetrieveCreateRequest(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, model any) bool {
	diags := req.Plan.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

func (it *ResourceHelper) RetrieveValidateConfigRequest(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse, model any) bool {
	diags := req.Config.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}
