package helpers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func RetrieveProviderConfigureRequest(
	ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse, model any,
) bool {
	diags := req.Config.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

func RetrieveProviderValidateConfigRequest(
	ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse, model any,
) bool {
	diags := req.Config.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}
