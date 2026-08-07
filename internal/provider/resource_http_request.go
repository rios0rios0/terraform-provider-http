package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	gopath "path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ohler55/ojg/jp"
	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
	"github.com/rios0rios0/terraform-provider-http/internal/infrastructure/helpers"
)

// Package-level regex for JSONPath token resolution, compiled once during package initialization.
var jsonPathTokenRe = regexp.MustCompile(`\$\.[^/]+`)

// Ensure HTTPRequestResource satisfies various resources interfaces.
var (
	_ resource.Resource                   = &HTTPRequestResource{}
	_ resource.ResourceWithConfigure      = &HTTPRequestResource{}
	_ resource.ResourceWithIdentity       = &HTTPRequestResource{}
	_ resource.ResourceWithImportState    = &HTTPRequestResource{}
	_ resource.ResourceWithModifyPlan     = &HTTPRequestResource{}
	_ resource.ResourceWithUpgradeState   = &HTTPRequestResource{}
	_ resource.ResourceWithValidateConfig = &HTTPRequestResource{}
)

// Attribute names of the http_request resource. They are declared once because the same strings
// address the schema, the import payload's JSON keys and the adoption bookkeeping, and those three
// must never drift apart.
const (
	attrMethod               = "method"
	attrPath                 = "path"
	attrHeaders              = "headers"
	attrRequestBody          = "request_body"
	attrIsResponseBodyJSON   = "is_response_body_json"
	attrResponseBodyIDFilter = "response_body_id_filter"
	attrQueryParameters      = "query_parameters"
	attrToleratedStatusCodes = "tolerated_status_codes"
	attrIgnoreChanges        = "ignore_changes"
	attrBaseURL              = "base_url"
	attrIsDeleteEnabled      = "is_delete_enabled"
	attrDeleteMethod         = "delete_method"
	attrDeletePath           = "delete_path"
	attrDeleteHeaders        = "delete_headers"
	attrDeleteRequestBody    = "delete_request_body"
	attrDeleteResolvedPath   = "delete_resolved_path"
	attrIsRefreshEnabled     = "is_refresh_enabled"
	attrRefreshPath          = "refresh_path"
	attrID                   = "id"
	attrImportID             = "import_id"
	attrResponseCode         = "response_code"
	attrResponseBody         = "response_body"
	attrResponseBodyID       = "response_body_id"
	attrResponseBodyJSON     = "response_body_json"
)

// Schema versions of the http_request resource. Version 2 is shape-identical to version 1; the
// bump exists so a state upgrader can rewrite captured numbers that version 1 recorded in
// scientific notation. Version 3 adds the refresh controls and the `import_id` helper.
const (
	schemaVersionV1 = 1
	schemaVersionV2 = 2
	schemaVersionV3 = 3
)

const deleteParamsPrivateKey = "delete_params"

type deleteParamsPrivate struct {
	IsDeleteEnabled   bool              `json:"is_delete_enabled"`
	DeleteMethod      string            `json:"delete_method,omitempty"`
	DeletePath        string            `json:"delete_path,omitempty"`
	DeleteHeaders     map[string]string `json:"delete_headers,omitempty"`
	DeleteRequestBody string            `json:"delete_request_body,omitempty"`
}

// HTTPRequestResource defines the resource implementation.
type HTTPRequestResource struct {
	internal *entities.InternalContext
}

// HTTPRequestResourceModel describes the resource data model.
type HTTPRequestResourceModel struct {
	// parameters
	Method               types.String `tfsdk:"method"`
	Path                 types.String `tfsdk:"path"`
	Headers              types.Map    `tfsdk:"headers"`
	RequestBody          types.String `tfsdk:"request_body"`
	IsResponseBodyJSON   types.Bool   `tfsdk:"is_response_body_json"`
	ResponseBodyIDFilter types.String `tfsdk:"response_body_id_filter"`
	QueryParameters      types.Map    `tfsdk:"query_parameters"`
	ToleratedStatusCodes types.Set    `tfsdk:"tolerated_status_codes"`
	IgnoreChanges        types.Set    `tfsdk:"ignore_changes"`

	// resource-level configuration (alternative to provider-level)
	BaseURL          types.String `tfsdk:"base_url"`
	BasicAuth        types.Object `tfsdk:"basic_auth"`
	IgnoreTLS        types.Bool   `tfsdk:"ignore_tls"`
	RequestTimeoutMs types.Int64  `tfsdk:"request_timeout_ms"`
	Retry            types.Object `tfsdk:"retry"`

	// destroy controls
	IsDeleteEnabled    types.Bool   `tfsdk:"is_delete_enabled"`
	DeleteMethod       types.String `tfsdk:"delete_method"`
	DeletePath         types.String `tfsdk:"delete_path"`
	DeleteHeaders      types.Map    `tfsdk:"delete_headers"`
	DeleteRequestBody  types.String `tfsdk:"delete_request_body"`
	DeleteResolvedPath types.String `tfsdk:"delete_resolved_path"`

	// refresh controls
	IsRefreshEnabled types.Bool   `tfsdk:"is_refresh_enabled"`
	RefreshPath      types.String `tfsdk:"refresh_path"`

	// state
	ID               types.String `tfsdk:"id"`
	ImportID         types.String `tfsdk:"import_id"`
	ResponseCode     types.Int32  `tfsdk:"response_code"`
	ResponseBody     types.String `tfsdk:"response_body"`
	ResponseBodyID   types.String `tfsdk:"response_body_id"`
	ResponseBodyJSON types.Map    `tfsdk:"response_body_json"`
}

// Default retry delays, matching the upstream hashicorp/http provider behavior.
const (
	defaultRetryMinDelayMs int64 = 1000
	defaultRetryMaxDelayMs int64 = 30000
)

// retryObjectAttrTypes returns the attribute types of the `retry` nested object.
// It MUST be used wherever a typed null `retry` value is produced, otherwise the
// framework raises a "missing type" conversion error.
func retryObjectAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		attrAttempts:   types.Int64Type,
		attrMinDelayMs: types.Int64Type,
		attrMaxDelayMs: types.Int64Type,
	}
}

// retryConfigFromObject converts a `retry` nested object into the domain
// RetryConfig, applying the default delays for any unset sub-attribute. It
// returns nil when the object is null/unknown (meaning "no retries").
func retryConfigFromObject(obj types.Object) *entities.RetryConfig {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	cfg := &entities.RetryConfig{
		MinDelayMs: defaultRetryMinDelayMs,
		MaxDelayMs: defaultRetryMaxDelayMs,
	}
	attrs := obj.Attributes()
	if v, ok := attrs[attrAttempts].(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
		cfg.Attempts = v.ValueInt64()
	}
	if v, ok := attrs[attrMinDelayMs].(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
		cfg.MinDelayMs = v.ValueInt64()
	}
	if v, ok := attrs[attrMaxDelayMs].(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
		cfg.MaxDelayMs = v.ValueInt64()
	}

	// Defensive clamping: keep delays sane regardless of user input.
	if cfg.MinDelayMs < 0 {
		cfg.MinDelayMs = 0
	}
	if cfg.MaxDelayMs < cfg.MinDelayMs {
		cfg.MaxDelayMs = cfg.MinDelayMs
	}
	return cfg
}

func NewHTTPRequestResource() resource.Resource {
	return &HTTPRequestResource{}
}

func GetHTTPRequestResourceSchema() schema.Schema {
	attrs := make(map[string]schema.Attribute)
	addRequestAttributes(attrs)
	addResourceConfigAttributes(attrs)
	addRetryTimeoutAttributes(attrs)
	addDeleteControlAttributes(attrs)
	addRefreshControlAttributes(attrs)
	addStateAttributes(attrs)
	addImportHelperAttributes(attrs)

	return schema.Schema{
		Version: schemaVersionV3,
		Description: "Represents an HTTP request resource, allowing configuration of various " +
			"HTTP request parameters and capturing the response details.",
		MarkdownDescription: "Represents an HTTP request resource, allowing configuration of various " +
			"HTTP request parameters and capturing the response details.",
		Attributes: attrs,
		Blocks: map[string]schema.Block{
			attrRetry: resourceRetryBlock(),
		},
	}
}

// getHTTPRequestResourceSchemaPreV3 returns the schema shape shared by versions 1 and 2. The two
// are shape-identical -- the version 2 bump existed only so its upgrader could rewrite captured
// numbers that version 1 recorded in scientific notation -- so one builder serves both, with the
// version supplied by the caller. Neither carries the refresh controls or `import_id`, which is
// exactly what the 2 -> 3 upgrader has to add.
func getHTTPRequestResourceSchemaPreV3(version int64) schema.Schema {
	attrs := make(map[string]schema.Attribute)
	addRequestAttributes(attrs)
	addResourceConfigAttributes(attrs)
	addRetryTimeoutAttributes(attrs)
	addDeleteControlAttributes(attrs)
	addStateAttributes(attrs)

	return schema.Schema{
		Version:    version,
		Attributes: attrs,
		Blocks: map[string]schema.Block{
			attrRetry: resourceRetryBlock(),
		},
	}
}

// addRetryTimeoutAttributes adds the operational `request_timeout_ms` knob. It is
// intentionally NOT part of addResourceConfigAttributes so the v0 upgrade schema
// (which reuses that function) keeps its pre-existing shape.
func addRetryTimeoutAttributes(attrs map[string]schema.Attribute) {
	attrs[attrRequestTimeoutMs] = helpers.Int64AttributeNoReplace(false,
		"The per-request timeout in milliseconds for this specific HTTP request. When specified, "+
			"this overrides the provider-level request_timeout_ms. When unset or 0, no timeout is applied "+
			"and a request can wait indefinitely.")
}

// resourceRetryBlock returns the resource-level `retry` block. When set it
// overrides the provider-level retry configuration. Retries are attempted on
// connection errors and on 5xx (except 501) responses, with an exponential
// backoff bounded by min_delay_ms and max_delay_ms.
func resourceRetryBlock() schema.SingleNestedBlock {
	description := "Retry configuration for this specific HTTP request. When specified, this overrides " +
		"the provider-level retry configuration. By default there are no retries."
	return schema.SingleNestedBlock{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			attrAttempts:   helpers.Int64AttributeNoReplace(false, descRetryAttempts),
			attrMinDelayMs: helpers.Int64AttributeNoReplace(false, descRetryMinDelayMs),
			attrMaxDelayMs: helpers.Int64AttributeNoReplace(false, descRetryMaxDelayMs),
		},
	}
}

func addRequestAttributes(attrs map[string]schema.Attribute) {
	attrs[attrMethod] = replaceableStringAttribute(true,
		"The HTTP method to be used for the request (e.g., GET, POST, PUT, DELETE).")
	attrs[attrPath] = replaceableStringAttribute(true,
		"The URL path for the HTTP request. This should be a relative path (e.g., /api/v1/resource).")
	attrs[attrHeaders] = replaceableMapAttribute(false, types.StringType,
		"A map of HTTP headers to include in the request. Each key-value pair represents a "+
			"header name and its corresponding value.")
	attrs[attrQueryParameters] = replaceableMapAttribute(false, types.StringType,
		"Optional query parameters to append to the request path")
	attrs[attrToleratedStatusCodes] = schema.SetAttribute{
		Description: "HTTP status codes that should be treated as successful in addition to the default 2xx range. " +
			"For example, setting this to [404] allows the resource to succeed when the server returns a 404 Not Found.",
		MarkdownDescription: "HTTP status codes that should be treated as successful in addition to the default 2xx range. " +
			"For example, setting this to `[404]` allows the resource to succeed when the server returns a `404 Not Found`.",
		Optional:    true,
		ElementType: types.Int32Type,
	}
	attrs[attrIgnoreChanges] = schema.SetAttribute{
		Description: "Optional list of attribute paths that should not force replacement when they change. " +
			"Supports top-level attributes (e.g. \"request_body\"), individual map entries (e.g. \"headers.X-Correlation-Id\"), " +
			"and JSON paths inside request bodies (e.g. \"request_body.metadata.trace_id\").",
		MarkdownDescription: "Optional list of attribute paths that should not force replacement when they change. " +
			"Supports top-level attributes (e.g. `request_body`), individual map entries (e.g. `headers.X-Correlation-Id`), " +
			"and JSON paths inside request bodies (e.g. `request_body.metadata.trace_id`).",
		Optional:    true,
		ElementType: types.StringType,
	}
	attrs[attrRequestBody] = replaceableStringAttribute(false,
		"The body content to be sent with the HTTP request. This is typically used for POST and PUT requests.")
	attrs[attrIsResponseBodyJSON] = replaceableBoolAttribute(false,
		"A boolean flag indicating whether the response body is expected to be in JSON format.")
	attrs[attrResponseBodyIDFilter] = replaceableStringAttribute(false,
		"A JSONPath filter used to extract a specific ID from the JSON response body. "+
			"This is useful for identifying unique elements within the response.")
}

func addResourceConfigAttributes(attrs map[string]schema.Attribute) {
	attrs[attrBaseURL] = replaceableStringAttribute(false,
		"The base URL for this specific HTTP request. When specified, this overrides the provider-level URL "+
			"configuration. This allows for different APIs to be used within the same configuration.")
	attrs[attrBasicAuth] = schema.SingleNestedAttribute{
		Description: "Credentials for basic authentication for this specific request. " +
			"When specified, this overrides the provider-level basic authentication configuration.",
		MarkdownDescription: "Credentials for basic authentication for this specific request. " +
			"When specified, this overrides the provider-level basic authentication configuration.",
		Optional: true,
		Attributes: map[string]schema.Attribute{
			attrUsername: schema.StringAttribute{
				Description:         "The username for basic authentication.",
				MarkdownDescription: "The username for basic authentication.",
				Required:            true,
			},
			attrPassword: schema.StringAttribute{
				Description:         "The password for basic authentication.",
				MarkdownDescription: "The password for basic authentication.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
	attrs[attrIgnoreTLS] = replaceableBoolAttribute(false,
		"A boolean flag to indicate whether TLS certificate verification should be ignored for this specific request. "+
			"When specified, this overrides the provider-level ignore_tls configuration.")
}

// addRefreshControlAttributes adds the opt-in drift detection. They are ordinary state-stored
// arguments rather than write-only ones because the Read RPC receives no configuration, so a
// write-only value would be unavailable exactly when refresh needs it.
func addRefreshControlAttributes(attrs map[string]schema.Attribute) {
	attrs[attrIsRefreshEnabled] = helpers.BoolAttributeNoReplace(false,
		"Enables drift detection. When true, every refresh re-issues a GET against `refresh_path` "+
			"(or `path`) and updates the captured response. A response that is neither successful nor "+
			"listed in `tolerated_status_codes` removes the resource from state so it is planned for "+
			"creation again. Defaults to false, which keeps the response captured at create time.")
	attrs[attrRefreshPath] = helpers.StringAttributeNoReplace(false,
		"Path to call when refreshing. Defaults to `path`. Supports the same inline JSONPath tokens "+
			"as `delete_path` (e.g. \"/posts/$.id\"), evaluated against the captured `response_body`, "+
			"which is what lets a resource created with POST refresh the object it created.")
}

func addDeleteControlAttributes(attrs map[string]schema.Attribute) {
	attrs[attrIsDeleteEnabled] = helpers.BoolAttributeWriteOnly(false,
		"Enables remote deletion during `terraform destroy`. If true and no delete_path is provided, "+
			"a DELETE will be sent to the original `path`.")
	attrs[attrDeleteMethod] = helpers.StringAttributeWriteOnly(false,
		"HTTP method to use during deletion (e.g., DELETE, POST). Defaults to DELETE.")
	attrs[attrDeletePath] = helpers.StringAttributeWriteOnly(false,
		"Path to call during deletion. Supports inline JSONPath tokens like \"/posts/$.data.id\" "+
			"evaluated against the `response_body` from create.")
	attrs[attrDeleteHeaders] = helpers.MapAttributeWriteOnly(false, types.StringType,
		"Headers to send only during deletion.")
	attrs[attrDeleteRequestBody] = helpers.StringAttributeWriteOnly(false,
		"Body to send only during deletion.")
	attrs[attrDeleteResolvedPath] = helpers.ComputedStringAttribute(
		"The `delete_path` with JSONPath tokens resolved from the create response, when possible.")
}

func addStateAttributes(attrs map[string]schema.Attribute) {
	attrs[attrID] = helpers.ComputedStringAttribute(
		"A unique identifier for the resource, generated when it is created. " +
			"Use `import_id` to obtain the identifier accepted by `terraform import`.")
	attrs[attrResponseCode] = helpers.ComputedInt32Attribute(
		"The HTTP status code returned by the server in response to the request " +
			"(e.g., 200 for success, 404 for not found).")
	attrs[attrResponseBody] = helpers.ComputedStringAttribute(
		"The raw body content returned by the server in response to the request.")
	attrs[attrResponseBodyID] = helpers.ComputedStringAttribute(
		"The extracted ID from the JSON response body, based on the provided " +
			"`response_body_id_filter`. This is only populated if `is_response_body_json` is true.")
	attrs[attrResponseBodyJSON] = helpers.ComputedMapAttribute(types.StringType,
		"The response body parsed as a Terraform map object. Nested items can be accessed "+
			"using dot notation (e.g., \"response_body_json[\"nested.item.value\"]\").")
}

// addImportHelperAttributes adds `import_id`, the ready-made identifier for re-importing this
// exact resource.
//
// A structured import identifier is only usable if the provider also emits it; expecting a
// practitioner to hand-assemble one is what makes structured identifiers hostile. Capturing it as
// an output means the identifier survives the loss of the state file that would make it necessary.
func addImportHelperAttributes(attrs map[string]schema.Attribute) {
	attrs[attrImportID] = helpers.ComputedStringAttribute(
		"A ready-made identifier for `terraform import`, describing the arguments this resource " +
			"was applied with. Neither `basic_auth` nor the captured response is encoded into it, " +
			"so it is safe to paste into a shell or a CI log and it does not duplicate the response " +
			"body into a second copy in state; a re-import captures a current response instead. " +
			"Capture it with an `output` block so it remains available if the state is ever lost.")
}

func (it *HTTPRequestResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_request"

	// The identity is derived from `method`, `path` and `base_url`, and import adoption can settle
	// those in place on the first apply after an import. Without this the framework would reject
	// that apply as an unexpected identity change.
	resp.ResourceBehavior = resource.ResourceBehavior{MutableIdentity: true}
}

// IdentitySchema declares the resource identity, which lets Terraform 1.12 and later import with
// `import { identity = { ... } }` instead of a stringly-typed identifier.
func (it *HTTPRequestResource) IdentitySchema(
	_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			attrBaseURL: identityschema.StringAttribute{
				OptionalForImport: true,
				Description: "The base URL of the request, when the resource overrides the " +
					"provider-level URL.",
			},
			attrMethod: identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "The HTTP method of the request.",
			},
			attrPath: identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "The URL path of the request.",
			},
		},
	}
}

func (it *HTTPRequestResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = GetHTTPRequestResourceSchema()
}

func (it *HTTPRequestResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var isJSON types.Bool
	var filter types.String

	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("is_response_body_json"), &isJSON)...,
	)
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("response_body_id_filter"), &filter)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	if !isJSON.IsUnknown() && isJSON.ValueBool() &&
		(filter.IsUnknown() || filter.IsNull() || strings.TrimSpace(filter.ValueString()) == "") {
		resp.Diagnostics.AddAttributeError(
			path.Root("response_body_id_filter"),
			"Since the response is JSON, the filter must be provided.",
			"When the expected answer is a JSON, the ID must be parsed in the state. "+
				"Please provide a filter to extract the ID from the JSON response. "+
				"Refer to the documentation for more information (https://github.com/ohler55/ojg).",
		)
	}

	validateToleratedStatusCodes(ctx, req, resp)
}

func validateToleratedStatusCodes(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var codes types.Set
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("tolerated_status_codes"), &codes)...,
	)
	if resp.Diagnostics.HasError() || codes.IsNull() || codes.IsUnknown() {
		return
	}

	var values []int32
	resp.Diagnostics.Append(codes.ElementsAs(ctx, &values, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	const minHTTPStatus, maxHTTPStatus = 100, 599
	for _, code := range values {
		if code < minHTTPStatus || code > maxHTTPStatus {
			resp.Diagnostics.AddAttributeError(
				path.Root("tolerated_status_codes"),
				"Invalid HTTP status code in tolerated_status_codes",
				fmt.Sprintf(
					"Status code %d is outside the valid HTTP range (%d-%d).",
					code, minHTTPStatus, maxHTTPStatus,
				),
			)
		}
	}
}

func (it *HTTPRequestResource) Configure(
	ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	tflog.Info(ctx, "Configuring resource to use HTTP client...")

	// added a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	internal, ok := req.ProviderData.(*entities.InternalContext)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *InternalContext, got: %T. "+
				"Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	it.internal = internal
}

func (it *HTTPRequestResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	tflog.Info(ctx, "Starting HTTP request...")

	var model HTTPRequestResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// WriteOnly attributes are nullified in the plan artifact. Re-read them from config.
	resp.Diagnostics.Append(copyWriteOnlyDeleteParams(ctx, req.Config, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exchange, ok := it.performRequest(ctx, model, &resp.Diagnostics)
	if !ok {
		return
	}

	if !it.acceptExchange(ctx, model, exchange, &resp.Diagnostics) {
		return
	}

	populateResponseState(ctx, &model, exchange, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	resp.Diagnostics.Append(setResourceIdentity(ctx, model, resp.Identity)...)

	// Persist delete params (write-only attrs) in private state so Delete() can access them.
	resp.Diagnostics.Append(marshalDeleteParamsToPrivate(ctx, model, resp.Private)...)

	// A resource that has just been created carries nothing over from an import.
	resp.Diagnostics.Append(clearImportAdoptFromPrivate(ctx, resp.Private)...)

	tflog.Info(ctx, "Completed HTTP request...", map[string]any{"success": true})
}

// httpExchange is the outcome of one HTTP round trip: the body is already drained and the
// connection released, so the value is safe to hold on to.
type httpExchange struct {
	statusCode int
	status     string
	body       []byte
}

// performRequest issues the request described by the model and returns the drained exchange.
//
// Create, Read and ImportState all need the same round trip, and keeping it in one place is what
// lets an import capture a response through exactly the code path that produced the state it is
// reconstructing.
func (it *HTTPRequestResource) performRequest(
	ctx context.Context,
	model HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) (*httpExchange, bool) {
	endpoint, diags := it.buildFullURL(ctx, model)
	diagnostics.Append(diags...)

	if diagnostics.HasError() {
		return nil, false
	}

	request, err := it.buildRequest(ctx, model, endpoint)
	if err != nil {
		diagnostics.AddError(
			"Error creating request. Check the method or request body informed...",
			err.Error(),
		)

		return nil, false
	}

	response, err := it.getHTTPClient(ctx, model).Do(request)
	if err != nil {
		diagnostics.AddError("Error executing request using HTTP client...", err.Error())

		return nil, false
	}

	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			diagnostics.AddError("Error closing the response body...", closeErr.Error())
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		diagnostics.AddError("Error reading the buffer from the response body...", err.Error())

		return nil, false
	}

	return &httpExchange{
		statusCode: response.StatusCode,
		status:     response.Status,
		body:       body,
	}, true
}

// acceptExchange reports whether the status is successful or explicitly tolerated, recording an
// error when it is neither.
func (it *HTTPRequestResource) acceptExchange(
	ctx context.Context,
	model HTTPRequestResourceModel,
	exchange *httpExchange,
	diagnostics *diag.Diagnostics,
) bool {
	if exchange.isSuccessful() {
		return true
	}

	if isStatusCodeTolerated(ctx, model.ToleratedStatusCodes, exchange.statusCode, diagnostics) {
		return true
	}

	diagnostics.AddError(
		"Error performing HTTP request. Not expected status code...",
		fmt.Sprintf(
			"Response code: %s. Response body: %s",
			exchange.status,
			string(exchange.body),
		),
	)

	return false
}

// isSuccessful reports whether the status is in the 2xx range.
func (e *httpExchange) isSuccessful() bool {
	return e.statusCode >= http.StatusOK && e.statusCode < http.StatusMultipleChoices
}

// copyWriteOnlyDeleteParams restores the write-only destroy controls from configuration.
//
// Terraform nullifies write-only attributes in both plan and state, so every code path that needs
// them has to read the configuration directly.
func copyWriteOnlyDeleteParams(
	ctx context.Context,
	config tfsdk.Config,
	model *HTTPRequestResourceModel,
) diag.Diagnostics {
	var configModel HTTPRequestResourceModel

	diagnostics := config.Get(ctx, &configModel)
	if diagnostics.HasError() {
		return diagnostics
	}

	model.IsDeleteEnabled = configModel.IsDeleteEnabled
	model.DeleteMethod = configModel.DeleteMethod
	model.DeletePath = configModel.DeletePath
	model.DeleteHeaders = configModel.DeleteHeaders
	model.DeleteRequestBody = configModel.DeleteRequestBody

	return diagnostics
}

// httpRequestResourceIdentityModel mirrors the identity schema.
type httpRequestResourceIdentityModel struct {
	BaseURL types.String `tfsdk:"base_url"`
	Method  types.String `tfsdk:"method"`
	Path    types.String `tfsdk:"path"`
}

// setResourceIdentity writes the resource identity, which the framework requires on every
// successful Create, Read, Update and ImportState once an identity schema is declared.
func setResourceIdentity(
	ctx context.Context,
	model HTTPRequestResourceModel,
	identity *tfsdk.ResourceIdentity,
) diag.Diagnostics {
	if identity == nil {
		return nil
	}

	return identity.Set(ctx, httpRequestResourceIdentityModel{
		BaseURL: model.BaseURL,
		Method:  model.Method,
		Path:    model.Path,
	})
}

// Read refreshes the resource.
//
// Drift detection is opt-in through `is_refresh_enabled`. A generic HTTP resource has no way to
// know which endpoint reflects the object it created -- for a POST the creation path is the
// collection, not the object -- so refreshing unconditionally would either re-run the original
// call or read the wrong URL. When the option is off the captured response is carried forward
// unchanged, which is the behaviour every existing configuration already relies on.
func (it *HTTPRequestResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var model HTTPRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if isBoolTrue(model.IsRefreshEnabled) {
		if !it.refreshFromRemote(ctx, &model, resp) {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	resp.Diagnostics.Append(setResourceIdentity(ctx, model, resp.Identity)...)
}

// refreshFromRemote re-reads the resource and updates the captured response in place.
//
// It reports false when the caller must stop, either because the read failed or because the
// resource is gone -- a status that is neither successful nor tolerated is taken as "no longer
// there", so the resource is dropped from state and planned for creation again rather than left
// pointing at something that no longer exists.
func (it *HTTPRequestResource) refreshFromRemote(
	ctx context.Context,
	model *HTTPRequestResourceModel,
	resp *resource.ReadResponse,
) bool {
	path, ok := resolveRefreshPath(*model, &resp.Diagnostics)
	if !ok {
		return false
	}

	refreshModel := *model
	refreshModel.Method = types.StringValue(http.MethodGet)
	refreshModel.Path = types.StringValue(path)
	refreshModel.RequestBody = types.StringNull()

	exchange, ok := it.performRequest(ctx, refreshModel, &resp.Diagnostics)
	if !ok {
		return false
	}

	if !exchange.isSuccessful() &&
		!isStatusCodeTolerated(ctx, model.ToleratedStatusCodes, exchange.statusCode, &resp.Diagnostics) {
		tflog.Info(ctx, "Refresh reported the resource is gone, removing it from state...",
			map[string]any{"status": exchange.status, attrPath: path})
		resp.State.RemoveResource(ctx)

		return false
	}

	populateResponseState(ctx, model, exchange, &resp.Diagnostics)

	return !resp.Diagnostics.HasError()
}

// resolveRefreshPath returns the path to read, defaulting to `path` and resolving any inline
// JSONPath tokens against the captured response body.
func resolveRefreshPath(
	model HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) (string, bool) {
	if !isNonEmptyString(model.RefreshPath) {
		return model.Path.ValueString(), true
	}

	return resolveDeletePathTokens(
		model.RefreshPath.ValueString(),
		model.ResponseBody.ValueString(),
		diagnostics,
	)
}

func (it *HTTPRequestResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var planModel HTTPRequestResourceModel
	var stateModel HTTPRequestResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An adoption left over from an import is settled first, and without touching the remote
	// resource. The state came from an import identifier that did not spell every argument out,
	// so the difference against configuration describes the gap in that identifier, not a change
	// the practitioner asked for. Re-issuing the request here would repeat the original side
	// effect -- for a POST, creating a second object -- which is exactly what importing instead
	// of recreating was meant to avoid.
	adopt, diags := unmarshalImportAdoptFromPrivate(ctx, req.Private)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if adopt != nil {
		it.adoptConfiguration(ctx, req, resp, planModel)

		return
	}

	// Only re-issue the HTTP request when an attribute that defines the request actually
	// changed. Response-interpretation attributes (tolerated_status_codes,
	// response_body_id_filter, is_response_body_json) and ignore_changes do not alter the
	// outgoing request, so changing one must not re-send it: re-sending would discard the
	// recorded response and, for a non-idempotent method, repeat the side effect. In that
	// case the plan already carries the captured response forward via UseStateForUnknown,
	// so persist it unchanged instead of producing an inconsistent result after apply.
	if !RequestAttributesChanged(planModel, stateModel) {
		// Write-only delete-control attributes are nullified in plan and state, so re-read
		// them from config (as Create does) and refresh them into private state. Skipping
		// this would leave a later Destroy running with stale delete configuration when only
		// a client-side attribute changed.
		resp.Diagnostics.Append(copyWriteOnlyDeleteParams(ctx, req.Config, &planModel)...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &planModel)...)
		resp.Diagnostics.Append(setResourceIdentity(ctx, planModel, resp.Identity)...)
		resp.Diagnostics.Append(marshalDeleteParamsToPrivate(ctx, planModel, resp.Private)...)

		return
	}

	it.Create(ctx, resource.CreateRequest{
		Config:       req.Config,
		Plan:         req.Plan,
		ProviderMeta: req.ProviderMeta,
		Identity:     req.Identity,
	}, (*resource.CreateResponse)(resp))
}

// adoptConfiguration settles a pending import adoption: the configuration becomes the state, the
// captured response is kept as imported, and the adoption is cleared so every later change is
// planned with the normal replacement rules again.
func (it *HTTPRequestResource) adoptConfiguration(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
	planModel HTTPRequestResourceModel,
) {
	resp.Diagnostics.Append(copyWriteOnlyDeleteParams(ctx, req.Config, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Adopting the configuration into the imported state without re-issuing the request...")

	resp.Diagnostics.Append(resp.State.Set(ctx, &planModel)...)
	resp.Diagnostics.Append(setResourceIdentity(ctx, planModel, resp.Identity)...)
	resp.Diagnostics.Append(marshalDeleteParamsToPrivate(ctx, planModel, resp.Private)...)
	resp.Diagnostics.Append(clearImportAdoptFromPrivate(ctx, resp.Private)...)
}

// RequestAttributesChanged reports whether any attribute that defines the outgoing HTTP
// request differs between the plan and the prior state. Response-interpretation attributes
// (tolerated_status_codes, response_body_id_filter, is_response_body_json), ignore_changes,
// the computed response attributes, and the write-only destroy controls are intentionally
// excluded -- changing any of them does not change the request that was sent.
func RequestAttributesChanged(plan, state HTTPRequestResourceModel) bool {
	return !plan.Method.Equal(state.Method) ||
		!plan.Path.Equal(state.Path) ||
		!plan.Headers.Equal(state.Headers) ||
		!plan.RequestBody.Equal(state.RequestBody) ||
		!plan.QueryParameters.Equal(state.QueryParameters) ||
		!plan.BaseURL.Equal(state.BaseURL) ||
		!plan.BasicAuth.Equal(state.BasicAuth) ||
		!plan.IgnoreTLS.Equal(state.IgnoreTLS)
}

func (it *HTTPRequestResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() || !req.Plan.Raw.IsKnown() {
		return
	}
	if req.State.Raw.IsNull() || !req.State.Raw.IsKnown() {
		return
	}

	var planModel HTTPRequestResourceModel
	var stateModel HTTPRequestResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Honor ignore_changes first so the request-change check below reflects the effective
	// plan (an ignored request attribute must not count as a change).
	entries := parseIgnoreEntries(ctx, planModel.IgnoreChanges, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(entries) > 0 {
		if !applyIgnoreEntries(ctx, entries, &planModel, &stateModel, &resp.Diagnostics) {
			return
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// A pending import adoption is settled in place, so the captured response is kept exactly as
	// imported and none of the computed attributes may be marked unknown -- Update writes the plan
	// through untouched, and an unknown left here would fail apply as an inconsistent result.
	adopt, diags := unmarshalImportAdoptFromPrivate(ctx, req.Private)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if adopt != nil {
		warnAboutPendingAdoption(adopt, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &planModel)...)

		return
	}

	// When a request-defining attribute changes, Update re-issues the request and the
	// captured response changes with it. The computed attributes carry UseStateForUnknown,
	// which would otherwise pin them to their stale values and fail apply with "Provider
	// produced inconsistent result after apply"; mark them unknown so the freshly captured
	// response is accepted. delete_resolved_path is included because it is recomputed from
	// the response in populateResponseState and is likewise UseStateForUnknown. A change
	// limited to client-side attributes leaves the recorded response untouched, so the
	// pinned values stay correct.
	if RequestAttributesChanged(planModel, stateModel) {
		planModel.ID = types.StringUnknown()
		planModel.ImportID = types.StringUnknown()
		planModel.ResponseCode = types.Int32Unknown()
		planModel.ResponseBody = types.StringUnknown()
		planModel.ResponseBodyID = types.StringUnknown()
		planModel.ResponseBodyJSON = types.MapUnknown(types.StringType)
		planModel.DeleteResolvedPath = types.StringUnknown()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &planModel)...)
}

// warnAboutPendingAdoption tells the practitioner which arguments the import identifier left out
// and are therefore about to be taken from configuration.
//
// Silence here would be worse than noise: the apply writes values into state that the remote
// system was never asked about, and that is only safe because the practitioner asserted the
// resource already matches the configuration by importing it.
func warnAboutPendingAdoption(adopt *importAdoptPrivate, diagnostics *diag.Diagnostics) {
	diagnostics.AddWarning(
		"Adopting configuration after import",
		fmt.Sprintf(
			"The import identifier did not specify %s, so the configured values are being adopted "+
				"into state on this apply. No HTTP request is sent and the resource is not replaced.\n\n"+
				"The remote resource is assumed to already match the configuration. If it does not, "+
				"apply a change to those arguments afterwards so the request is re-issued.",
			strings.Join(adopt.Attributes, ", "),
		),
	)
}

func isStatusCodeTolerated(
	ctx context.Context,
	toleratedCodes types.Set,
	statusCode int,
	diagnostics *diag.Diagnostics,
) bool {
	if toleratedCodes.IsNull() || toleratedCodes.IsUnknown() {
		return false
	}

	var codes []int32
	diagnostics.Append(toleratedCodes.ElementsAs(ctx, &codes, false)...)
	if diagnostics.HasError() {
		return false
	}

	//nolint:gosec // G115: safe - HTTP status codes are 3-digit values (100-599), well within int32 range
	return slices.Contains(codes, int32(statusCode))
}

func populateResponseState(
	ctx context.Context,
	model *HTTPRequestResourceModel,
	exchange *httpExchange,
	diagnostics *diag.Diagnostics,
) {
	// A real response always carries a status, so the guard only matters for the synthetic
	// exchange an import builds from a payload that supplied a body but no code: there the
	// attribute stays null rather than being recorded as a meaningless zero.
	if exchange.statusCode > 0 {
		//nolint:gosec // G115: safe - HTTP status codes are 3-digit values (100-599), well within int32 range
		model.ResponseCode = types.Int32Value(int32(exchange.statusCode))
	}

	model.ResponseBody = types.StringValue(string(exchange.body))
	updateResponseBody(model, diagnostics)
	updateResponseBodyID(model, []byte(model.ResponseBody.ValueString()), diagnostics)
	updateResponseBodyJSON(model, []byte(model.ResponseBody.ValueString()), diagnostics)

	if !model.DeletePath.IsNull() && model.DeletePath.ValueString() != "" {
		resolved, ok := resolveDeletePathTokens(
			model.DeletePath.ValueString(), model.ResponseBody.ValueString(), diagnostics,
		)
		if ok {
			model.DeleteResolvedPath = types.StringValue(resolved)
		} else {
			model.DeleteResolvedPath = types.StringNull()
		}
	} else {
		model.DeleteResolvedPath = types.StringNull()
	}

	if len(model.ID.ValueString()) == 0 {
		model.ID = types.StringValue(uuid.NewString())
	}

	model.ImportID = buildImportID(ctx, *model, diagnostics)
}

func isBoolTrue(v types.Bool) bool {
	return !v.IsNull() && v.ValueBool()
}

func isNonEmptyString(v types.String) bool {
	return !v.IsNull() && strings.TrimSpace(v.ValueString()) != ""
}

func pickDeleteMethod(m HTTPRequestResourceModel) string {
	if isNonEmptyString(m.DeleteMethod) {
		return strings.ToUpper(strings.TrimSpace(m.DeleteMethod.ValueString()))
	}
	return http.MethodDelete
}

func resolveDeleteTargetPath(
	m HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) (string, bool) {
	if !isNonEmptyString(m.DeletePath) {
		return m.Path.ValueString(), true
	}

	if isNonEmptyString(m.DeleteResolvedPath) {
		return m.DeleteResolvedPath.ValueString(), true
	}
	if m.ResponseBody.IsNull() || m.ResponseBody.ValueString() == "" {
		diagnostics.AddError(
			"Missing response_body to resolve delete_path",
			"`delete_path` contains JSONPath tokens but `response_body` is empty; cannot resolve.",
		)
		return "", false
	}

	resolved, ok := resolveDeletePathTokens(
		m.DeletePath.ValueString(),
		m.ResponseBody.ValueString(),
		diagnostics,
	)
	if !ok {
		return "", false
	}
	return resolved, true
}

func makeDeleteModel(
	base HTTPRequestResourceModel,
	method string,
	targetPath string,
) HTTPRequestResourceModel {
	dm := base
	dm.Method = types.StringValue(method)
	dm.Path = types.StringValue(targetPath)

	// Body only if provided for delete
	if isNonEmptyString(base.DeleteRequestBody) {
		dm.RequestBody = types.StringValue(base.DeleteRequestBody.ValueString())
	} else {
		dm.RequestBody = types.StringNull()
	}

	// Headers only if provided for delete
	if !base.DeleteHeaders.IsNull() && base.DeleteHeaders.Elements() != nil {
		dm.Headers = base.DeleteHeaders
	} else {
		dm.Headers = types.MapNull(types.StringType)
	}

	return dm
}

func (it *HTTPRequestResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var model HTTPRequestResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete params are write-only and not stored in state. Read them from private state.
	deleteParams, diags := unmarshalDeleteParamsFromPrivate(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyDeleteParamsToModel(&model, deleteParams)

	if !isBoolTrue(model.IsDeleteEnabled) {
		resp.State.RemoveResource(ctx)
		return
	}

	method := pickDeleteMethod(model)

	targetPath, ok := resolveDeleteTargetPath(model, &resp.Diagnostics)
	if !ok {
		return
	}

	delModel := makeDeleteModel(model, method, targetPath)

	endpoint, diags := it.buildFullURL(ctx, delModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, err := it.buildRequest(ctx, delModel, endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Error creating DELETE request", err.Error())
		return
	}

	client := it.getHTTPClient(ctx, model)
	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Error executing DELETE HTTP request", err.Error())
		return
	}
	defer func() {
		if cerr := response.Body.Close(); cerr != nil {
			resp.Diagnostics.AddError("Error closing the DELETE response body...", cerr.Error())
		}
	}()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading DELETE response", err.Error())
		return
	}
	tflog.Debug(ctx, "DELETE response details", map[string]any{
		"status": response.StatusCode,
		"body":   string(responseBody),
	})

	// Treat any non-2xx as error (unless the status code is tolerated)
	tolerated := isStatusCodeTolerated(
		ctx, model.ToleratedStatusCodes, response.StatusCode, &resp.Diagnostics,
	)
	if !helpers.IsResponseSuccessful(response) && !tolerated {
		resp.Diagnostics.AddError(
			"DELETE request failed with unexpected status code",
			fmt.Sprintf("Response code: %s. Body: %s", response.Status, string(responseBody)),
		)
		return
	}

	resp.State.RemoveResource(ctx)
}

// ImportState brings an existing remote resource under management.
//
// Terraform never shows the resource's configuration to this method, so an identifier that does
// not spell out every argument necessarily produces a state that differs from it. Rather than let
// that difference land on the replacement rules -- which would destroy and recreate the very
// resource the practitioner could not afford to recreate -- the gap is recorded as a pending
// adoption and settled in place by the first apply. See [importAdoptPrivate].
func (it *HTTPRequestResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	payload := resolveImportPayload(ctx, req, &resp.Diagnostics)
	if payload == nil {
		return
	}

	model := buildModelFromNativeData(payload.native, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	model.ID = types.StringValue(payload.id)

	if !validateImportedModel(*model, &resp.Diagnostics) {
		return
	}

	it.captureImportedResponse(ctx, model, payload.native, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	model.ImportID = buildImportID(ctx, *model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	resp.Diagnostics.Append(setResourceIdentity(ctx, *model, resp.Identity)...)

	// Persist any delete params supplied in the import payload to private state.
	resp.Diagnostics.Append(marshalDeleteParamsToPrivate(ctx, *model, resp.Private)...)

	// Record what the identifier left unsaid so the first plan can adopt it from configuration
	// instead of planning a replacement.
	resp.Diagnostics.Append(
		marshalImportAdoptToPrivate(ctx, pendingAdoptionFor(payload.specified), resp.Private)...,
	)
}

// resolveImportPayload accepts either spelling of an import: the identifier string used by
// `terraform import` and `import { id = ... }`, or the `import { identity = ... }` block that
// Terraform 1.12 and later support.
func resolveImportPayload(
	ctx context.Context,
	req resource.ImportStateRequest,
	diagnostics *diag.Diagnostics,
) *importPayload {
	if req.ID != "" {
		return decodeImportID(req.ID, diagnostics)
	}

	if req.Identity == nil {
		addUnrecognisedImportIDError(diagnostics, "no import identifier or identity was supplied")

		return nil
	}

	var identity httpRequestResourceIdentityModel

	diagnostics.Append(req.Identity.Get(ctx, &identity)...)

	if diagnostics.HasError() {
		return nil
	}

	payload := newShorthandImportPayload(
		strings.ToUpper(identity.Method.ValueString()),
		identity.Path.ValueString(),
		true,
	)

	if isNonEmptyString(identity.BaseURL) {
		payload.native.BaseURL = identity.BaseURL.ValueString()
		payload.specified[attrBaseURL] = struct{}{}
	}

	return payload
}

// validateImportedModel checks the minimum an identifier has to carry.
func validateImportedModel(
	model HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) bool {
	if !isNonEmptyString(model.Method) || !isNonEmptyString(model.Path) {
		addUnrecognisedImportIDError(
			diagnostics,
			"it does not describe both a method and a path",
		)

		return false
	}

	if isBoolTrue(model.IsResponseBodyJSON) && !isNonEmptyString(model.ResponseBodyIDFilter) {
		diagnostics.AddAttributeError(
			path.Root(attrResponseBodyIDFilter),
			"Since the response is JSON, the filter must be provided.",
			"When the expected answer is a JSON, the ID must be parsed in the state. "+
				"Please provide a filter to extract the ID from the JSON response. "+
				"Refer to the documentation for more information (https://github.com/ohler55/ojg).",
		)

		return false
	}

	return true
}

// captureImportedResponse fills the computed response attributes of an imported resource.
//
// The response matters beyond reporting: `delete_resolved_path` is derived from it, and without
// that a `delete_path` carrying JSONPath tokens cannot be resolved, leaving the imported resource
// impossible to destroy.
func (it *HTTPRequestResource) captureImportedResponse(
	ctx context.Context,
	model *HTTPRequestResourceModel,
	native *HTTPRequestResourceModelNative,
	diagnostics *diag.Diagnostics,
) {
	// A payload that carried a response is taken at its word; it is replayed through the same
	// code path as a live one so the derived attributes are computed identically.
	if isNonEmptyString(model.ResponseBody) {
		populateResponseState(ctx, model, &httpExchange{
			statusCode: int(model.ResponseCode.ValueInt32()),
			body:       []byte(model.ResponseBody.ValueString()),
		}, diagnostics)

		return
	}

	readPath, ok := importReadPath(*model, native)
	if !ok {
		warnAboutUncapturedImportResponse(model.Method.ValueString(), diagnostics)

		return
	}

	readModel := *model
	readModel.Method = types.StringValue(http.MethodGet)
	readModel.Path = types.StringValue(readPath)
	readModel.RequestBody = types.StringNull()

	exchange, ok := it.performRequest(ctx, readModel, diagnostics)
	if !ok {
		return
	}

	if !it.acceptExchange(ctx, *model, exchange, diagnostics) {
		return
	}

	populateResponseState(ctx, model, exchange, diagnostics)
}

// importReadPath returns the path an import may safely read, and whether reading is allowed.
//
// Replaying the recorded method is only safe when it has no side effects. Re-sending a POST would
// create a second remote object -- the opposite of importing one -- so an unsafe method must name
// `import_read_path` explicitly to opt into a GET against the object it created.
func importReadPath(
	model HTTPRequestResourceModel,
	native *HTTPRequestResourceModelNative,
) (string, bool) {
	if native.ImportReadPath != "" {
		return native.ImportReadPath, true
	}

	if isSafeHTTPMethod(model.Method.ValueString()) {
		return model.Path.ValueString(), true
	}

	return "", false
}

// warnAboutUncapturedImportResponse explains why the computed response attributes are empty and
// what to do about it.
func warnAboutUncapturedImportResponse(method string, diagnostics *diag.Diagnostics) {
	diagnostics.AddWarning(
		"Imported without capturing a response",
		fmt.Sprintf(
			"The import payload describes a %s request, which cannot be replayed safely because "+
				"re-sending it would repeat its side effect. The computed attributes "+
				"(`response_code`, `response_body`, `response_body_id`, `response_body_json`, "+
				"`delete_resolved_path`) are therefore empty.\n\n"+
				"To capture them, re-import with either `response_body` or `import_read_path` in the "+
				"payload; the latter is a path this provider will GET to read the created object. "+
				"This matters when `delete_path` contains JSONPath tokens, because they are resolved "+
				"against `response_body` and destroy fails without it.",
			method,
		),
	)
}

// UpgradeState migrates prior state shapes to the current schema.
//
// Every upgrader writes into the *current* schema, not the next one along, because the framework
// hands each one a response state built from the current resource schema. That is why the version
// 0 upgrader below has to fill in attributes introduced long after version 1.
func (it *HTTPRequestResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	v0Attrs := make(map[string]schema.Attribute)
	addRequestAttributes(v0Attrs)
	addResourceConfigAttributes(v0Attrs)
	addDeleteControlAttributesV0(v0Attrs)
	addStateAttributes(v0Attrs)

	v0Schema := schema.Schema{
		Version:    0,
		Attributes: v0Attrs,
	}

	v1Schema := getHTTPRequestResourceSchemaPreV3(schemaVersionV1)
	v2Schema := getHTTPRequestResourceSchemaPreV3(schemaVersionV2)

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &v0Schema,
			StateUpgrader: upgradeStateFromV0,
		},
		1: {
			PriorSchema:   &v1Schema,
			StateUpgrader: upgradeStateFromV1,
		},
		2: {
			PriorSchema:   &v2Schema,
			StateUpgrader: upgradeStateFromV2,
		},
	}
}

// upgradeStateFromV0 migrates the original schema.
func upgradeStateFromV0(
	ctx context.Context,
	req resource.UpgradeStateRequest,
	resp *resource.UpgradeStateResponse,
) {
	var oldModel httpRequestResourceModelV0

	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newModel := HTTPRequestResourceModel{
		Method:               oldModel.Method,
		Path:                 oldModel.Path,
		Headers:              oldModel.Headers,
		RequestBody:          oldModel.RequestBody,
		IsResponseBodyJSON:   oldModel.IsResponseBodyJSON,
		ResponseBodyIDFilter: oldModel.ResponseBodyIDFilter,
		QueryParameters:      oldModel.QueryParameters,
		ToleratedStatusCodes: oldModel.ToleratedStatusCodes,
		IgnoreChanges:        oldModel.IgnoreChanges,
		BaseURL:              oldModel.BaseURL,
		BasicAuth:            oldModel.BasicAuth,
		IgnoreTLS:            oldModel.IgnoreTLS,
		DeleteResolvedPath:   oldModel.DeleteResolvedPath,

		// request_timeout_ms and retry are new in this schema version; carry them as
		// typed nulls so the very next plan does not fail with a "missing type"
		// conversion error (mirrors the delete_* WriteOnly fix in 3.1.10).
		RequestTimeoutMs: types.Int64Null(),
		Retry:            types.ObjectNull(retryObjectAttrTypes()),
		ID:               oldModel.ID,
		ResponseCode:     oldModel.ResponseCode,
		ResponseBody:     oldModel.ResponseBody,
		ResponseBodyID:   oldModel.ResponseBodyID,
		ResponseBodyJSON: oldModel.ResponseBodyJSON,

		// The delete-control attributes became WriteOnly in schema v1. They must be
		// carried as TYPED nulls here -- leaving them as the struct's zero value gives
		// DeleteHeaders an element-typeless types.Map{}, which makes the very next plan
		// fail with "Value Conversion Error ... Path: delete_headers"
		// (Map[!!! MISSING TYPE !!!] / Map[DynamicPseudoType]). WriteOnly values are
		// never persisted, so null is the correct upgraded-state value.
		IsDeleteEnabled:   types.BoolNull(),
		DeleteMethod:      types.StringNull(),
		DeletePath:        types.StringNull(),
		DeleteHeaders:     types.MapNull(types.StringType),
		DeleteRequestBody: types.StringNull(),

		// Introduced in schema v3, and typed nulls for the same reason as above.
		IsRefreshEnabled: types.BoolNull(),
		RefreshPath:      types.StringNull(),
		ImportID:         types.StringNull(),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newModel)...)
}

// upgradeStateFromV1 repairs the numbers version 1 recorded in scientific notation and then adds
// the attributes introduced by version 3.
func upgradeStateFromV1(
	ctx context.Context,
	req resource.UpgradeStateRequest,
	resp *resource.UpgradeStateResponse,
) {
	var oldModel httpRequestResourceModelPreV3

	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newModel := oldModel.toCurrent()
	repairExponentNotation(ctx, &newModel, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, newModel)...)
}

// upgradeStateFromV2 adds the refresh controls and `import_id`.
func upgradeStateFromV2(
	ctx context.Context,
	req resource.UpgradeStateRequest,
	resp *resource.UpgradeStateResponse,
) {
	var oldModel httpRequestResourceModelPreV3

	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, oldModel.toCurrent())...)
}

// RepairExponentNotation applies the schema v1 -> v2 captured-number repair to a model
// (exported for testing).
func RepairExponentNotation(
	ctx context.Context,
	model *HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) {
	repairExponentNotation(ctx, model, diagnostics)
}

// repairExponentNotation rewrites the captured attributes that schema version 1 rendered with
// the `%v` verb. Applied to the float64 that `encoding/json` produces for every JSON number,
// `%v` formats with `%g` and switches to scientific notation once the exponent grows, so a
// nine-digit identifier such as 803554429 was recorded as "8.03554429e+08".
//
// Three attributes carried the damage: `response_body_id`, `delete_resolved_path` (the id is
// substituted into the path, and Destroy prefers the stored value over re-resolving it, so a
// corrupted path makes the resource undeletable) and the `response_body_json` map.
//
// The repair is driven by the resource's own recorded `response_body`: every number in it is
// rendered both the old way and the new way, and only those exact renderings are substituted.
// Nothing is guessed, and a response body that no longer parses leaves the state untouched.
func repairExponentNotation(
	ctx context.Context,
	model *HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) {
	if model.ResponseBody.IsNull() || model.ResponseBody.ValueString() == "" {
		return
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(model.ResponseBody.ValueString()), &decoded); err != nil {
		return
	}

	replacements := make(map[string]string)
	collectExponentRenderings(decoded, replacements)
	if len(replacements) == 0 {
		return
	}

	// The id is a single scalar rendering, so require an exact match rather than a substring
	// one -- that cannot corrupt an id that merely contains a number as a substring.
	if !model.ResponseBodyID.IsNull() {
		if repaired, found := replacements[model.ResponseBodyID.ValueString()]; found {
			model.ResponseBodyID = types.StringValue(repaired)
		}
	}

	// The resolved path embeds the rendering inside a larger string, so substitute in place.
	// One rendering can be a substring of another -- "1e+08" sits inside "1.1e+08" -- and Go
	// randomises map iteration, so substituting the shorter one first would turn
	// "/posts/1.1e+08" into "/posts/1.100000000" on some runs and not others. Applying the
	// longest rendering first removes both the corruption and the non-determinism. Every
	// replacement value is digits only while every rendering carries an exponent, so a
	// substitution can never produce text that a later rendering matches.
	if !model.DeleteResolvedPath.IsNull() {
		repaired := model.DeleteResolvedPath.ValueString()
		for _, rendering := range renderingsLongestFirst(replacements) {
			repaired = strings.ReplaceAll(repaired, rendering, replacements[rendering])
		}
		model.DeleteResolvedPath = types.StringValue(repaired)
	}

	// The map is derived wholesale from the response body, so simply rebuild it.
	if model.IsResponseBodyJSON.ValueBool() {
		rebuilt, diags := types.MapValueFrom(ctx, types.StringType, helpers.ConvertToStringMap(decoded))
		diagnostics.Append(diags...)
		if !diags.HasError() {
			model.ResponseBodyJSON = rebuilt
		}
	}
}

// renderingsLongestFirst returns the old renderings ordered longest first, ties broken
// lexicographically so the sequence is stable across runs.
func renderingsLongestFirst(replacements map[string]string) []string {
	renderings := make([]string, 0, len(replacements))
	for rendering := range replacements {
		renderings = append(renderings, rendering)
	}

	slices.SortFunc(renderings, func(left, right string) int {
		if byLength := len(right) - len(left); byLength != 0 {
			return byLength
		}
		return strings.Compare(left, right)
	})

	return renderings
}

// collectExponentRenderings walks a decoded JSON document and records, for every number whose
// rendering changed between schema versions, the old rendering mapped to the new one.
func collectExponentRenderings(value any, replacements map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		for _, nested := range typed {
			collectExponentRenderings(nested, replacements)
		}
	case []any:
		for _, nested := range typed {
			collectExponentRenderings(nested, replacements)
		}
	default:
		previous := fmt.Sprintf("%v", value)
		if current := helpers.FormatJSONScalar(value); current != previous {
			replacements[previous] = current
		}
	}
}

// httpRequestResourceModelPreV3 is the state shape shared by schema versions 1 and 2. It exists so
// the upgraders can decode prior state without the attributes version 3 added -- decoding into the
// current model would fail with "Struct defines fields not found in object".
type httpRequestResourceModelPreV3 struct {
	Method               types.String `tfsdk:"method"`
	Path                 types.String `tfsdk:"path"`
	Headers              types.Map    `tfsdk:"headers"`
	RequestBody          types.String `tfsdk:"request_body"`
	IsResponseBodyJSON   types.Bool   `tfsdk:"is_response_body_json"`
	ResponseBodyIDFilter types.String `tfsdk:"response_body_id_filter"`
	QueryParameters      types.Map    `tfsdk:"query_parameters"`
	ToleratedStatusCodes types.Set    `tfsdk:"tolerated_status_codes"`
	IgnoreChanges        types.Set    `tfsdk:"ignore_changes"`
	BaseURL              types.String `tfsdk:"base_url"`
	BasicAuth            types.Object `tfsdk:"basic_auth"`
	IgnoreTLS            types.Bool   `tfsdk:"ignore_tls"`
	RequestTimeoutMs     types.Int64  `tfsdk:"request_timeout_ms"`
	Retry                types.Object `tfsdk:"retry"`
	IsDeleteEnabled      types.Bool   `tfsdk:"is_delete_enabled"`
	DeleteMethod         types.String `tfsdk:"delete_method"`
	DeletePath           types.String `tfsdk:"delete_path"`
	DeleteHeaders        types.Map    `tfsdk:"delete_headers"`
	DeleteRequestBody    types.String `tfsdk:"delete_request_body"`
	DeleteResolvedPath   types.String `tfsdk:"delete_resolved_path"`
	ID                   types.String `tfsdk:"id"`
	ResponseCode         types.Int32  `tfsdk:"response_code"`
	ResponseBody         types.String `tfsdk:"response_body"`
	ResponseBodyID       types.String `tfsdk:"response_body_id"`
	ResponseBodyJSON     types.Map    `tfsdk:"response_body_json"`
}

// toCurrent lifts a pre-version-3 state into the current model, carrying the attributes version 3
// introduced as TYPED nulls. An untyped zero value would give the very next plan a
// "Value Conversion Error ... MISSING TYPE", which is the failure the version 0 upgrader documents.
func (m httpRequestResourceModelPreV3) toCurrent() HTTPRequestResourceModel {
	return HTTPRequestResourceModel{
		Method:               m.Method,
		Path:                 m.Path,
		Headers:              m.Headers,
		RequestBody:          m.RequestBody,
		IsResponseBodyJSON:   m.IsResponseBodyJSON,
		ResponseBodyIDFilter: m.ResponseBodyIDFilter,
		QueryParameters:      m.QueryParameters,
		ToleratedStatusCodes: m.ToleratedStatusCodes,
		IgnoreChanges:        m.IgnoreChanges,
		BaseURL:              m.BaseURL,
		BasicAuth:            m.BasicAuth,
		IgnoreTLS:            m.IgnoreTLS,
		RequestTimeoutMs:     m.RequestTimeoutMs,
		Retry:                m.Retry,
		IsDeleteEnabled:      m.IsDeleteEnabled,
		DeleteMethod:         m.DeleteMethod,
		DeletePath:           m.DeletePath,
		DeleteHeaders:        m.DeleteHeaders,
		DeleteRequestBody:    m.DeleteRequestBody,
		DeleteResolvedPath:   m.DeleteResolvedPath,
		ID:                   m.ID,
		ResponseCode:         m.ResponseCode,
		ResponseBody:         m.ResponseBody,
		ResponseBodyID:       m.ResponseBodyID,
		ResponseBodyJSON:     m.ResponseBodyJSON,

		IsRefreshEnabled: types.BoolNull(),
		RefreshPath:      types.StringNull(),
		ImportID:         types.StringNull(),
	}
}

type httpRequestResourceModelV0 struct {
	Method               types.String `tfsdk:"method"`
	Path                 types.String `tfsdk:"path"`
	Headers              types.Map    `tfsdk:"headers"`
	RequestBody          types.String `tfsdk:"request_body"`
	IsResponseBodyJSON   types.Bool   `tfsdk:"is_response_body_json"`
	ResponseBodyIDFilter types.String `tfsdk:"response_body_id_filter"`
	QueryParameters      types.Map    `tfsdk:"query_parameters"`
	ToleratedStatusCodes types.Set    `tfsdk:"tolerated_status_codes"`
	IgnoreChanges        types.Set    `tfsdk:"ignore_changes"`
	BaseURL              types.String `tfsdk:"base_url"`
	BasicAuth            types.Object `tfsdk:"basic_auth"`
	IgnoreTLS            types.Bool   `tfsdk:"ignore_tls"`
	IsDeleteEnabled      types.Bool   `tfsdk:"is_delete_enabled"`
	DeleteMethod         types.String `tfsdk:"delete_method"`
	DeletePath           types.String `tfsdk:"delete_path"`
	DeleteHeaders        types.Map    `tfsdk:"delete_headers"`
	DeleteRequestBody    types.String `tfsdk:"delete_request_body"`
	DeleteResolvedPath   types.String `tfsdk:"delete_resolved_path"`
	ID                   types.String `tfsdk:"id"`
	ResponseCode         types.Int32  `tfsdk:"response_code"`
	ResponseBody         types.String `tfsdk:"response_body"`
	ResponseBodyID       types.String `tfsdk:"response_body_id"`
	ResponseBodyJSON     types.Map    `tfsdk:"response_body_json"`
}

func addDeleteControlAttributesV0(attrs map[string]schema.Attribute) {
	attrs["is_delete_enabled"] = helpers.BoolAttributeNoReplace(false,
		"Enables remote deletion during `terraform destroy`.")
	attrs["delete_method"] = helpers.StringAttributeNoReplace(false,
		"HTTP method to use during deletion.")
	attrs["delete_path"] = helpers.StringAttributeNoReplace(false,
		"Path to call during deletion.")
	attrs["delete_headers"] = helpers.MapAttributeNoReplace(false, types.StringType,
		"Headers to send only during deletion.")
	attrs["delete_request_body"] = helpers.StringAttributeNoReplace(false,
		"Body to send only during deletion.")
	attrs["delete_resolved_path"] = helpers.ComputedStringAttribute(
		"The `delete_path` with JSONPath tokens resolved from the create response, when possible.")
}

func marshalDeleteParamsToPrivate(ctx context.Context, model HTTPRequestResourceModel, private interface {
	SetKey(context.Context, string, []byte) diag.Diagnostics
}) diag.Diagnostics {
	params := deleteParamsPrivate{
		IsDeleteEnabled: isBoolTrue(model.IsDeleteEnabled),
	}
	if isNonEmptyString(model.DeleteMethod) {
		params.DeleteMethod = model.DeleteMethod.ValueString()
	}
	if isNonEmptyString(model.DeletePath) {
		params.DeletePath = model.DeletePath.ValueString()
	}
	if !model.DeleteHeaders.IsNull() && !model.DeleteHeaders.IsUnknown() {
		var headers map[string]string
		if d := model.DeleteHeaders.ElementsAs(ctx, &headers, false); !d.HasError() {
			params.DeleteHeaders = headers
		}
	}
	if isNonEmptyString(model.DeleteRequestBody) {
		params.DeleteRequestBody = model.DeleteRequestBody.ValueString()
	}

	data, err := json.Marshal(params)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Failed to marshal delete params to private state", err.Error())
		return d
	}

	return private.SetKey(ctx, deleteParamsPrivateKey, data)
}

func unmarshalDeleteParamsFromPrivate(ctx context.Context, private interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
}) (*deleteParamsPrivate, diag.Diagnostics) {
	if private == nil {
		return nil, nil
	}

	data, diags := private.GetKey(ctx, deleteParamsPrivateKey)
	if diags.HasError() || len(data) == 0 {
		return nil, diags
	}

	var params deleteParamsPrivate
	if err := json.Unmarshal(data, &params); err != nil {
		var d diag.Diagnostics
		d.AddError("Failed to unmarshal delete params from private state", err.Error())
		return nil, d
	}

	return &params, nil
}

func applyDeleteParamsToModel(model *HTTPRequestResourceModel, params *deleteParamsPrivate) {
	if params == nil {
		return
	}
	model.IsDeleteEnabled = types.BoolValue(params.IsDeleteEnabled)
	if params.DeleteMethod != "" {
		model.DeleteMethod = types.StringValue(params.DeleteMethod)
	}
	if params.DeletePath != "" {
		model.DeletePath = types.StringValue(params.DeletePath)
	}
	if len(params.DeleteHeaders) > 0 {
		model.DeleteHeaders, _ = types.MapValueFrom(context.Background(), types.StringType, params.DeleteHeaders)
	}
	if params.DeleteRequestBody != "" {
		model.DeleteRequestBody = types.StringValue(params.DeleteRequestBody)
	}
}

func (it *HTTPRequestResource) buildRequest(
	ctx context.Context, model HTTPRequestResourceModel, endpoint string,
) (*http.Request, error) {
	var body io.Reader
	looksJSON := false

	if !model.RequestBody.IsNull() {
		send, isJSON := coerceBodyString(model.RequestBody.ValueString())
		body = bytes.NewBufferString(send)
		looksJSON = isJSON
	}

	req, err := http.NewRequestWithContext(
		ctx,
		model.Method.ValueString(),
		endpoint,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	// Provider-level headers go on FIRST so the resource's own can override them, and so a
	// provider-level `Content-Type` still suppresses the JSON default below.
	it.applyProviderHeaders(req.Header)

	if applyErr := applyHeadersFromMapAttr(ctx, req.Header, model.Headers); applyErr != nil {
		return nil, applyErr
	}

	applyDefaultJSONHeaders(req.Header, isBoolTrue(model.IsResponseBodyJSON), looksJSON)

	// Apply authentication - resource-level takes precedence over provider-level
	if !model.BasicAuth.IsNull() {
		// Use resource-level basic auth
		authAttrs := model.BasicAuth.Attributes()
		username, ok := authAttrs[attrUsername].(types.String)
		if !ok {
			return nil, errors.New("failed to get username from basic_auth")
		}
		password, ok := authAttrs[attrPassword].(types.String)
		if !ok {
			return nil, errors.New("failed to get password from basic_auth")
		}
		req.SetBasicAuth(username.ValueString(), password.ValueString())
	} else if config := it.providerConfig(); config.HasAuthentication() {
		// Fall back to provider-level basic auth
		req.SetBasicAuth(
			config.BasicAuth.Username,
			config.BasicAuth.Password,
		)
	}

	return req, nil
}

func coerceBodyString(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false
	}
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		if unq, err := strconv.Unquote(trimmed); err == nil {
			unqTrimmed := strings.TrimSpace(unq)
			if strings.HasPrefix(unqTrimmed, "{") || strings.HasPrefix(unqTrimmed, "[") {
				return unq, true
			}
		}
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return raw, true
	}
	return raw, false
}

func applyHeadersFromMapAttr(ctx context.Context, h http.Header, m types.Map) error {
	if m.IsNull() || m.Elements() == nil {
		return nil
	}
	var headers map[string]string
	d := m.ElementsAs(ctx, &headers, false)
	if d.HasError() {
		var diagDetails []string
		for _, err := range d.Errors() {
			diagDetails = append(diagDetails, fmt.Sprintf("%s: %s", err.Summary(), err.Detail()))
		}
		return fmt.Errorf("invalid headers provided: %s", strings.Join(diagDetails, "; "))
	}
	for k, v := range headers {
		h.Set(k, v)
	}
	return nil
}

// applyProviderHeaders writes the provider-level headers onto a request.
//
// Every request the resource issues is built here, so this one call covers create, read, refresh,
// destroy AND the read an import performs -- which is the case that cannot be served any other way.
// `ImportState` is handed the import identifier and nothing else, because Terraform never shows it
// the configuration, so a credential kept in the resource's own `headers` is unavailable exactly
// when the import read needs it, and spelling it into the identifier would print it wherever plan
// output goes.
//
// `http.Header.Set` canonicalises the name, so a resource header applied afterwards overrides the
// provider's regardless of the casing either side used -- which is the behaviour RFC 9110 requires
// of header names.
func (it *HTTPRequestResource) applyProviderHeaders(h http.Header) {
	config := it.providerConfig()
	if config == nil {
		return
	}

	for name, value := range config.Headers {
		h.Set(name, value)
	}
}

// providerConfig returns the provider-level configuration, or nil when there is none to read.
//
// Terraform sets provider data AFTER the ConfigureProvider RPC, and `Configure` returns early while
// it is absent, which leaves `internal` nil in that window. Reading through this keeps every
// consumer of the provider configuration safe in it; before it existed, the provider-level
// basic-auth fallback below dereferenced `internal` unguarded.
func (it *HTTPRequestResource) providerConfig() *entities.Configuration {
	if it.internal == nil {
		return nil
	}

	return it.internal.Config
}

func applyDefaultJSONHeaders(h http.Header, expectJSON bool, looksJSON bool) {
	if (expectJSON || looksJSON) && h.Get("Content-Type") == "" {
		h.Set("Content-Type", "application/json; charset=UTF-8")
	}
	if expectJSON && h.Get("Accept") == "" {
		h.Set("Accept", "application/json")
	}
}

func updateResponseBody(model *HTTPRequestResourceModel, diagnostics *diag.Diagnostics) {
	if model.IsResponseBodyJSON.ValueBool() {
		var compactedJSON bytes.Buffer
		err := json.Compact(&compactedJSON, []byte(model.ResponseBody.ValueString()))
		if err != nil {
			diagnostics.AddError("Error compacting JSON response body...", err.Error())
			return
		}
		model.ResponseBody = types.StringValue(compactedJSON.String())
	}
}

func updateResponseBodyID(
	model *HTTPRequestResourceModel,
	responseBody []byte,
	diagnostics *diag.Diagnostics,
) {
	model.ResponseBodyID = types.StringNull()
	if !model.IsResponseBodyJSON.ValueBool() {
		return
	}

	jsonResponse, err := unmarshalJSON(responseBody, diagnostics)
	if err != nil {
		return
	}

	jsonPath, err := parseJSONPath(model.ResponseBodyIDFilter.ValueString(), diagnostics)
	if err != nil {
		return
	}

	element := jsonPath.First(jsonResponse)
	if element != nil {
		model.ResponseBodyID = types.StringValue(helpers.FormatJSONScalar(element))
	} else {
		diagnostics.AddWarning("The JSON path provided didn't return any value...",
			"Please check the `response_body_id_filter` provided.")
	}
}

func unmarshalJSON(
	responseBody []byte,
	diagnostics *diag.Diagnostics,
) (map[string]any, error) {
	var jsonResponse map[string]any
	if err := json.Unmarshal(responseBody, &jsonResponse); err != nil {
		diagnostics.AddWarning(
			"It wasn't possible to unmarshall response body to a JSON map reference...",
			err.Error(),
		)
		return nil, fmt.Errorf("%w", err)
	}
	return jsonResponse, nil
}

func parseJSONPath(filter string, diagnostics *diag.Diagnostics) (jp.Expr, error) {
	jsonPath, err := jp.ParseString(filter)
	if err != nil {
		diagnostics.AddWarning(
			"It wasn't possible to parse the JSON path using the `response_body_id_filter` provided...",
			err.Error(),
		)
		return nil, fmt.Errorf("%w", err)
	}
	return jsonPath, nil
}

func updateResponseBodyJSON(
	model *HTTPRequestResourceModel,
	responseBody []byte,
	diagnostics *diag.Diagnostics,
) {
	var diags diag.Diagnostics
	model.ResponseBodyJSON, diags = types.MapValue(types.StringType, make(map[string]attr.Value))
	diagnostics.Append(diags...)

	if model.IsResponseBodyJSON.ValueBool() {
		var result map[string]any
		err := json.Unmarshal(responseBody, &result)
		if err != nil {
			diagnostics.AddError(
				"Error unmarshalling response body to a JSON map reference...",
				err.Error(),
			)
		}

		model.ResponseBodyJSON, diags = types.MapValueFrom(context.Background(),
			types.StringType, helpers.ConvertToStringMap(result))
		diagnostics.Append(diags...)
	}
}

func resolveDeletePathTokens(rawPath, responseBody string, diagnostics *diag.Diagnostics) (string, bool) {
	if !strings.Contains(rawPath, "$.") {
		return rawPath, true
	}

	jsonResponse, err := unmarshalJSON([]byte(responseBody), diagnostics)
	if err != nil {
		diagnostics.AddError("Failed to parse response_body for delete_path resolution",
			"response_body is not valid JSON or could not be parsed.")
		return "", false
	}

	resolved := rawPath
	tokens := jsonPathTokenRe.FindAllString(rawPath, -1)
	for _, token := range tokens {
		expr, exprErr := parseJSONPath(token, diagnostics)
		if exprErr != nil {
			diagnostics.AddError("Failed to parse JSONPath token in delete_path",
				fmt.Sprintf("token: %q, cause: %v", token, exprErr))
			return "", false
		}
		val := expr.First(jsonResponse)
		if val == nil {
			diagnostics.AddError("JSONPath token not found in response_body",
				fmt.Sprintf("token: %q did not resolve against create response", token))
			return "", false
		}
		repl := helpers.FormatJSONScalar(val)
		resolved = strings.ReplaceAll(resolved, token, repl)
	}

	return resolved, true
}

func (it *HTTPRequestResource) buildFullURL(
	ctx context.Context,
	model HTTPRequestResourceModel,
) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	var baseURLString string
	switch {
	case !model.BaseURL.IsNull() && model.BaseURL.ValueString() != "":
		baseURLString = model.BaseURL.ValueString()
	case it.internal != nil && it.internal.Config != nil && it.internal.Config.URL != "":
		baseURLString = it.internal.Config.URL
	default:
		diags.AddError(
			"No base URL configured",
			"A base URL must be configured either at the provider level (using the 'url' attribute) "+
				"or at the resource level (using the 'base_url' attribute). "+
				"This is required to construct the full URL for the HTTP request.",
		)
		return "", diags
	}

	baseURL, err := url.Parse(baseURLString)
	if err != nil {
		diags.AddError("Error parsing base URL", err.Error())
		return "", diags
	}

	relativePath := model.Path.ValueString()
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	userURL, err := url.Parse(relativePath)
	if err != nil {
		diags.AddError("Error parsing user URL", err.Error())
		return "", diags
	}

	baseURL.Path = gopath.Join(baseURL.Path, userURL.Path)

	query := userURL.Query()
	var queryParams map[string]string
	if !model.QueryParameters.IsNull() && model.QueryParameters.Elements() != nil {
		d := model.QueryParameters.ElementsAs(ctx, &queryParams, false)
		diags.Append(d...)
		if diags.HasError() {
			return "", diags
		}
		for k, v := range queryParams {
			query.Add(k, v)
		}
	}
	baseURL.RawQuery = query.Encode()

	baseURL.Fragment = userURL.Fragment

	finalURL := baseURL.String()
	return finalURL, diags
}

// getHTTPClient returns the HTTP client to use for this request. It resolves the
// effective TLS, timeout, and retry settings -- resource-level values take
// precedence over the provider-level configuration -- and builds a client
// accordingly. When retries are configured the returned client transparently
// retries on connection errors and 5xx (except 501) responses, applying an
// exponential backoff bounded by the configured min/max delays. The per-request
// timeout (when set) bounds each individual attempt; an unset/zero timeout
// preserves the historical behavior of waiting indefinitely.
func (it *HTTPRequestResource) getHTTPClient(
	_ context.Context,
	model HTTPRequestResourceModel,
) *http.Client {
	ignoreTLS := it.resolveIgnoreTLS(model)
	timeout := it.resolveTimeout(model)
	retryCfg := it.resolveRetry(model)

	base := &http.Client{Timeout: timeout}
	if ignoreTLS {
		base.Transport = it.resolveInsecureTransport()
	}

	if retryCfg == nil || retryCfg.Attempts <= 0 {
		return base
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = base
	// Silence the library's default stderr logger; diagnostics flow through tflog.
	retryClient.Logger = nil
	retryClient.RetryMax = int(retryCfg.Attempts)
	retryClient.RetryWaitMin = time.Duration(retryCfg.MinDelayMs) * time.Millisecond
	retryClient.RetryWaitMax = time.Duration(retryCfg.MaxDelayMs) * time.Millisecond
	return retryClient.StandardClient()
}

// resolveIgnoreTLS resolves the effective ignore_tls setting: a resource-level
// value wins; otherwise the provider-level value (detected from the configured
// client transport) is used.
func (it *HTTPRequestResource) resolveIgnoreTLS(model HTTPRequestResourceModel) bool {
	if !model.IgnoreTLS.IsNull() {
		return model.IgnoreTLS.ValueBool()
	}
	if it.internal != nil && it.internal.Client != nil && it.internal.Client.Transport != nil {
		if transport, ok := it.internal.Client.Transport.(*http.Transport); ok && transport.TLSClientConfig != nil {
			return transport.TLSClientConfig.InsecureSkipVerify
		}
	}
	return false
}

// resolveInsecureTransport returns a transport that skips TLS verification. It
// reuses the provider-level transport when one is already configured so the
// underlying connection pool is shared across requests; a fresh transport is
// allocated only when there is none to reuse (for example, a resource-level
// ignore_tls override turning verification off where the provider left it on).
func (it *HTTPRequestResource) resolveInsecureTransport() http.RoundTripper {
	if it.internal != nil && it.internal.Client != nil {
		if transport, ok := it.internal.Client.Transport.(*http.Transport); ok &&
			transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
			return transport
		}
	}
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			// InsecureSkipVerify is intentionally set to true when ignore_tls is enabled.
			// This is a user-controlled feature for testing and self-signed certificates.
			//nolint:gosec // purposefully ignore TLS verification according to user configuration
			InsecureSkipVerify: true,
		},
	}
}

// resolveTimeout resolves the effective per-request timeout: a resource-level
// value wins; otherwise the provider-level value is used. A non-positive value
// means no timeout.
func (it *HTTPRequestResource) resolveTimeout(model HTTPRequestResourceModel) time.Duration {
	var ms int64
	if it.internal != nil && it.internal.Config != nil {
		ms = it.internal.Config.RequestTimeoutMs
	}
	if !model.RequestTimeoutMs.IsNull() && !model.RequestTimeoutMs.IsUnknown() {
		ms = model.RequestTimeoutMs.ValueInt64()
	}
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// resolveRetry resolves the effective retry configuration: a resource-level
// `retry` block wins; otherwise the provider-level configuration is used.
func (it *HTTPRequestResource) resolveRetry(model HTTPRequestResourceModel) *entities.RetryConfig {
	if !model.Retry.IsNull() && !model.Retry.IsUnknown() {
		return retryConfigFromObject(model.Retry)
	}
	if it.internal != nil && it.internal.Config != nil {
		return it.internal.Config.Retry
	}
	return nil
}
