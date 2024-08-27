package internal

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
	"github.com/rios0rios0/terraform-provider-http/internal/infrastructure/helpers"
	"os"
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
	URL       types.String `tfsdk:"url" json:"url"`
	BasicAuth types.Object `tfsdk:"basic_auth" json:"basic_auth"`
	IgnoreTLS types.Bool   `tfsdk:"ignore_tls" json:"-"`
}

func (it *HTTPProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "http"
	resp.Version = it.version
}

func (it *HTTPProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description:         "The base URL for all HTTP requests made by this provider. This URL serves as the root endpoint for the Web endpoint that the provider will interact with. It is a required attribute and must be specified to ensure proper communication with the target.",
				MarkdownDescription: "The base URL for all HTTP requests made by this provider. This URL serves as the root endpoint for the Web endpoint that the provider will interact with. It is a required attribute and must be specified to ensure proper communication with the target.",
				Required:            true,
			},
			"basic_auth": schema.SingleNestedAttribute{
				Description:         "Credentials for basic authentication. This attribute allows you to specify the username and password required for basic HTTP authentication. It is optional and should be used when the target Web endpoint requires basic authentication for access.",
				MarkdownDescription: "Credentials for basic authentication. This attribute allows you to specify the username and password required for basic HTTP authentication. It is optional and should be used when the target Web endpoint requires basic authentication for access.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description:         "The username for basic authentication. This is a required field within the `basic_auth` block and must be provided if basic authentication is used.",
						MarkdownDescription: "The username for basic authentication. This is a required field within the `basic_auth` block and must be provided if basic authentication is used.",
						Required:            true,
					},
					"password": schema.StringAttribute{
						Description:         "The password for basic authentication. This is a required field within the `basic_auth` block and must be provided if basic authentication is used. The password is marked as sensitive to ensure it is not exposed in logs or state files.",
						MarkdownDescription: "The password for basic authentication. This is a required field within the `basic_auth` block and must be provided if basic authentication is used. The password is marked as sensitive to ensure it is not exposed in logs or state files.",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
			"ignore_tls": schema.BoolAttribute{
				Description:         "A boolean flag to indicate whether TLS certificate verification should be ignored. This is useful for testing purposes or when interacting with APIs that use self-signed certificates. It is optional and defaults to `false`.",
				MarkdownDescription: "A boolean flag to indicate whether TLS certificate verification should be ignored. This is useful for testing purposes or when interacting with APIs that use self-signed certificates. It is optional and defaults to `false`.",
				Optional:            true,
			},
		},
	}
}

func (it *HTTPProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var model HTTPProviderModel
	helper := helpers.NewProviderHelper()
	if !helper.RetrieveValidateConfigRequest(ctx, req, resp, &model) {
		return
	}

	if model.URL.IsUnknown() || model.URL.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Unknown URL for HTTP client",
			"The provider cannot create the HTTP client as there is an unknown/null configuration value for the URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PROVIDER_HTTP_URL environment variable.",
		)
	}
	if !model.BasicAuth.IsUnknown() || !model.BasicAuth.IsNull() {
		if model.BasicAuth.Attributes()["username"].IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("basic_auth").AtName("username"),
				"Unknown username for HTTP client",
				"The provider cannot create the HTTP client as there is an unknown/null configuration value for the username. "+
					"Either target apply the source of the value first, set the value statically in the configuration, or use the PROVIDER_HTTP_USERNAME environment variable.",
			)
		}
		if model.BasicAuth.Attributes()["password"].IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("basic_auth").AtName("password"),
				"Unknown password for HTTP client",
				"The provider cannot create the HTTP client as there is an unknown/null configuration value for the password. "+
					"Either target apply the source of the value first, set the value statically in the configuration, or use the PROVIDER_HTTP_PASSWORD environment variable.",
			)
		}
	}
}

func (it *HTTPProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring HTTP client...")

	var model HTTPProviderModel
	helper := helpers.NewProviderHelper()
	if !helper.RetrieveConfigureRequest(ctx, req, resp, &model) {
		return
	}

	// here all configuration values are known and valid
	// check if it should override configuration values from environment variables
	url := os.Getenv("PROVIDER_HTTP_URL")
	username := os.Getenv("PROVIDER_HTTP_USERNAME")
	password := os.Getenv("PROVIDER_HTTP_PASSWORD")
	if url == "" {
		url = model.URL.ValueString()
	}
	if !model.BasicAuth.IsNull() { // double-checking because it is optional
		if username == "" {
			username = model.BasicAuth.Attributes()["username"].(types.String).ValueString()
		}
		if password == "" {
			password = model.BasicAuth.Attributes()["password"].(types.String).ValueString()
		}
	}

	ctx = tflog.SetField(ctx, "http_url", url)
	ctx = tflog.SetField(ctx, "http_username", username)
	ctx = tflog.SetField(ctx, "http_password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "http_password")
	ctx = tflog.SetField(ctx, "http_ignore_tls", model.IgnoreTLS.ValueBool())

	//TODO: is it worth to use JSON instead of getting value per value?
	//var source Configuration
	//jsonData, _ := json.Marshal(model)
	//_ = json.Unmarshal(jsonData, &source)

	tflog.Debug(ctx, "Creating HTTP client...")

	internal := entities.NewInternalContext(
		model.IgnoreTLS.ValueBool(),
		entities.NewConfiguration(url),
	)
	if !model.BasicAuth.IsNull() {
		internal.Config.BasicAuth = &entities.BasicAuth{
			Username: username,
			Password: password,
		}
	}

	resp.ResourceData = internal
	resp.DataSourceData = internal

	tflog.Info(ctx, "Configured HTTP client...", map[string]any{"success": true})
}

func (it *HTTPProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHTTPRequestResource,
	}
}

func (it *HTTPProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (it *HTTPProvider) Functions(context.Context) []func() function.Function {
	return []func() function.Function{}
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HTTPProvider{
			version: version,
		}
	}
}
