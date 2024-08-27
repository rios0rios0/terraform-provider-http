package helpers

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/provider"
)

type ProviderHelper struct {
}

func NewProviderHelper() *ProviderHelper {
	return &ProviderHelper{}
}

func (it *ProviderHelper) RetrieveConfigureRequest(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse, model any) bool {
	diags := req.Config.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

func (it *ProviderHelper) RetrieveValidateConfigRequest(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse, model any) bool {
	diags := req.Config.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}
