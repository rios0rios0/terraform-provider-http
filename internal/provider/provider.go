package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure HTTPProvider satisfies various provider interfaces.
var (
	_ provider.Provider              = &HTTPProvider{}
	_ provider.ProviderWithFunctions = &HTTPProvider{}
)

// HTTPProvider defines the provider implementation.
type HTTPProvider struct {
	version string
}

// HTTPProviderModel describes the provider data model.
type HTTPProviderModel struct {
	URL       types.String `tfsdk:"url"`
	BasicAuth types.Object `tfsdk:"basic_auth"`
	IgnoreTLS types.Bool   `tfsdk:"ignore_tls"`
}

func (it *HTTPProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "http"
	resp.Version = it.version
}

func (it *HTTPProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Base URL for HTTP requests",
				Required:            true,
			},
			"basic_auth": schema.SingleNestedAttribute{
				MarkdownDescription: "Basic authentication credentials",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						MarkdownDescription: "Username for basic authentication",
						Required:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "Password for basic authentication",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
			"ignore_tls": schema.BoolAttribute{
				MarkdownDescription: "Ignore TLS certificate verification",
				Optional:            true,
				//Default:             false, TODO: uncomment this line
			},
		},
	}
}

func (it *HTTPProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data HTTPProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	config := map[string]interface{}{
		"url":        data.URL.ValueString(),
		"basic_auth": data.BasicAuth,
		"ignore_tls": data.IgnoreTLS.ValueBool(),
	}

	resp.ResourceData = config
	resp.DataSourceData = config
}

func (it *HTTPProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHTTPRequestResource,
	}
}

func (it *HTTPProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (it *HTTPProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HTTPProvider{
			version: version,
		}
	}
}
