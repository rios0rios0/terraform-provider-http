package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
)

const (
	attrBasicAuth        = "basic_auth"
	attrIgnoreTLS        = "ignore_tls"
	attrUsername         = "username"
	attrPassword         = "password"
	attrRequestTimeoutMs = "request_timeout_ms"
	attrRetry            = "retry"
	attrAttempts         = "attempts"
	attrMinDelayMs       = "min_delay_ms"
	attrMaxDelayMs       = "max_delay_ms"
)

// Descriptions for the retry/timeout knobs. They are shared between the provider
// and the resource schema builders (same package) so the wording lives in one
// place and the two schemas cannot drift apart.
const (
	descRetryAttempts = "The maximum number of retries. For example, if `2` is specified, the request is " +
		"tried a maximum of 3 times (the initial attempt plus 2 retries)."
	descRetryMinDelayMs          = "The minimum delay between retries, in milliseconds. Defaults to `1000`."
	descRetryMaxDelayMs          = "The maximum delay between retries, in milliseconds. Defaults to `30000`."
	descRequestTimeoutMsProvider = "The per-request timeout in milliseconds applied to every HTTP request " +
		"made by this provider. When unset or `0`, no timeout is applied and a request can wait indefinitely. " +
		"It can be overridden per resource using the `request_timeout_ms` argument."
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
	URL              types.String `tfsdk:"url"                json:"url"`
	BasicAuth        types.Object `tfsdk:"basic_auth"         json:"basic_auth"`
	IgnoreTLS        types.Bool   `tfsdk:"ignore_tls"         json:"-"`
	RequestTimeoutMs types.Int64  `tfsdk:"request_timeout_ms" json:"-"`
	Retry            types.Object `tfsdk:"retry"              json:"-"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HTTPProvider{
			version: version,
		}
	}
}

func GetHTTPProviderSchema() schema.Schema {
	return schema.Schema{
		Description: "The HTTP provider allows you to interact with Web endpoints using HTTP requests. " +
			"It is useful for interacting with RESTful APIs, webhooks, and other HTTP-based services.",
		MarkdownDescription: "The HTTP provider allows you to interact with Web endpoints using HTTP requests. " +
			"It is useful for interacting with RESTful APIs, webhooks, and other HTTP-based services.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "The base URL for all HTTP requests made by this provider. " +
					"This URL serves as the root endpoint for the Web endpoint that the provider will interact with. " +
					"This is optional when base_url is specified at the resource level.",
				MarkdownDescription: "The base URL for all HTTP requests made by this provider. " +
					"This URL serves as the root endpoint for the Web endpoint that the provider will interact with. " +
					"This is optional when base_url is specified at the resource level.",
				Optional: true,
				// TODO: Validators: []validator.String{validators.NewStringNotEmpty("url")},
			},
			attrBasicAuth: schema.SingleNestedAttribute{
				Description: "Credentials for basic authentication. " +
					"This attribute allows you to specify the username and password required for basic HTTP authentication. " +
					"It is optional and should be used when the target Web endpoint requires basic authentication for access.",
				MarkdownDescription: "Credentials for basic authentication. " +
					"This attribute allows you to specify the username and password required for basic HTTP authentication. " +
					"It is optional and should be used when the target Web endpoint requires basic authentication for access.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					attrUsername: schema.StringAttribute{
						Description: "The username for basic authentication. " +
							"This is a required field within the `basic_auth` block and must be provided if basic authentication is used.",
						MarkdownDescription: "The username for basic authentication. " +
							"This is a required field within the `basic_auth` block and must be provided if basic authentication is used.",
						Required: true,
					},
					attrPassword: schema.StringAttribute{
						Description: "The password for basic authentication. " +
							"This is a required field within the `basic_auth` block and must be provided if basic authentication is used. " +
							"The password is marked as sensitive to ensure it is not exposed in logs or state files.",
						MarkdownDescription: "The password for basic authentication. " +
							"This is a required field within the `basic_auth` block and must be provided if basic authentication is used. " +
							"The password is marked as sensitive to ensure it is not exposed in logs or state files.",
						Required:  true,
						Sensitive: true,
					},
				},
			},
			attrIgnoreTLS: schema.BoolAttribute{
				Description: "A boolean flag to indicate whether TLS certificate verification should be ignored. " +
					"This is useful for testing purposes or when interacting with APIs that use self-signed certificates. " +
					"It is optional and defaults to `false`.",
				MarkdownDescription: "A boolean flag to indicate whether TLS certificate verification should be ignored. " +
					"This is useful for testing purposes or when interacting with APIs that use self-signed certificates. " +
					"It is optional and defaults to `false`.",
				Optional: true,
			},
			attrRequestTimeoutMs: providerOptionalInt64(descRequestTimeoutMsProvider),
		},
		Blocks: map[string]schema.Block{
			attrRetry: retryBlock(),
		},
	}
}

// providerOptionalInt64 builds an optional provider-level Int64 attribute,
// keeping `Description` and `MarkdownDescription` in sync.
func providerOptionalInt64(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Description:         description,
		MarkdownDescription: description,
		Optional:            true,
	}
}

// retryBlock returns the provider-level `retry` block. It mirrors the upstream
// hashicorp/http provider's retry semantics: retries are attempted on connection
// errors and on 5xx (except 501) responses, with an exponential backoff bounded
// by `min_delay_ms` and `max_delay_ms`. Configured here it applies to every
// request; it can be overridden per resource.
func retryBlock() schema.SingleNestedBlock {
	description := "Retry configuration applied to every HTTP request made by this provider. " +
		"By default there are no retries. Retries are attempted on connection errors and on 5xx " +
		"(except 501) responses. It can be overridden per resource using the `retry` block."
	return schema.SingleNestedBlock{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			attrAttempts:   providerOptionalInt64(descRetryAttempts),
			attrMinDelayMs: providerOptionalInt64(descRetryMinDelayMs),
			attrMaxDelayMs: providerOptionalInt64(descRetryMaxDelayMs),
		},
	}
}

func (it *HTTPProvider) Metadata(
	_ context.Context,
	_ provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "http"
	resp.Version = it.version
}

func (it *HTTPProvider) Schema(
	_ context.Context,
	_ provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = GetHTTPProviderSchema()
}

// ValidateConfig At this point "IsUnknown()" is not useful because it is always true in the real apply.
func (it *HTTPProvider) ValidateConfig(
	ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse,
) {
	var model HTTPProviderModel

	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	const detailMessage = "Either target apply the source of the value first, " +
		"set the value statically in the configuration, "

	// you can't access the content here because they're not known yet, they'll be known in the Configure method
	// URL is now optional since it can be provided at the resource level

	if !model.BasicAuth.IsNull() {
		username := model.BasicAuth.Attributes()[attrUsername]
		if username.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root(attrBasicAuth).AtName(attrUsername),
				"Unknown username for HTTP client",
				"The provider cannot create the HTTP client as there is a null configuration value for the username. "+
					detailMessage+"or use the PROVIDER_HTTP_USERNAME environment variable.",
			)
		}

		password := model.BasicAuth.Attributes()[attrPassword]
		if password.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root(attrBasicAuth).AtName(attrPassword),
				"Unknown password for HTTP client",
				"The provider cannot create the HTTP client as there is a null configuration value for the password. "+
					detailMessage+"or use the PROVIDER_HTTP_PASSWORD environment variable.",
			)
		}
	}
}

func (it *HTTPProvider) Configure(
	ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse,
) {
	tflog.Info(ctx, "Configuring HTTP client...")

	var model HTTPProviderModel

	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// here all configuration values are known and valid
	// check if it should override configuration values from environment variables
	url := os.Getenv("PROVIDER_HTTP_URL")
	username := os.Getenv("PROVIDER_HTTP_USERNAME")
	password := os.Getenv("PROVIDER_HTTP_PASSWORD")
	if url == "" && !model.URL.IsNull() {
		url = model.URL.ValueString()
	}
	if !model.BasicAuth.IsNull() { // double-checking because it is optional
		if username == "" {
			//nolint:errcheck // it was checked before in the validation
			username = model.BasicAuth.Attributes()[attrUsername].(types.String).ValueString()
		}
		if password == "" {
			//nolint:errcheck // it was checked before in the validation
			password = model.BasicAuth.Attributes()[attrPassword].(types.String).ValueString()
		}
	}

	ctx = tflog.SetField(ctx, "http_url", url)
	ctx = tflog.SetField(ctx, "http_username", username)
	ctx = tflog.SetField(ctx, "http_password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "http_password")
	ctx = tflog.SetField(ctx, "http_ignore_tls", model.IgnoreTLS.ValueBool())

	/* TODO: is it worth to use JSON instead of getting value per value?
	var source Configuration
	jsonData, _ := json.Marshal(model)
	_ = json.Unmarshal(jsonData, &source) */

	tflog.Debug(ctx, "Creating HTTP client...")

	internal := entities.NewInternalContext(
		model.IgnoreTLS.ValueBool(),
		entities.NewConfiguration(url), // URL can be empty string if only using resource-level URLs
	)
	if !model.BasicAuth.IsNull() {
		internal.Config.BasicAuth = &entities.BasicAuth{
			Username: username,
			Password: password,
		}
	}
	if !model.RequestTimeoutMs.IsNull() && !model.RequestTimeoutMs.IsUnknown() {
		internal.Config.RequestTimeoutMs = model.RequestTimeoutMs.ValueInt64()
	}
	internal.Config.Retry = retryConfigFromObject(model.Retry)

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
