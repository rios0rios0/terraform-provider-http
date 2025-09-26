package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	gopath "path"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &HTTPRequestResource{}
	_ resource.ResourceWithConfigure   = &HTTPRequestResource{}
	_ resource.ResourceWithImportState = &HTTPRequestResource{}
)

const AmountOfPartsInID = 2

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

	// resource-level configuration (alternative to provider-level)
	BaseURL   types.String `tfsdk:"base_url"`
	BasicAuth types.Object `tfsdk:"basic_auth"`
	IgnoreTLS types.Bool   `tfsdk:"ignore_tls"`

	// destroy controls
	IsDeleteEnabled    types.Bool   `tfsdk:"is_delete_enabled"`
	DeleteMethod       types.String `tfsdk:"delete_method"`
	DeletePath         types.String `tfsdk:"delete_path"`
	DeleteHeaders      types.Map    `tfsdk:"delete_headers"`
	DeleteRequestBody  types.String `tfsdk:"delete_request_body"`
	DeleteResolvedPath types.String `tfsdk:"delete_resolved_path"`

	// state
	ID               types.String `tfsdk:"id"`
	ResponseCode     types.Int32  `tfsdk:"response_code"`
	ResponseBody     types.String `tfsdk:"response_body"`
	ResponseBodyID   types.String `tfsdk:"response_body_id"`
	ResponseBodyJSON types.Map    `tfsdk:"response_body_json"`
}

// HTTPRequestResourceModelNative describes the resource data model in a native Go format.
type HTTPRequestResourceModelNative struct {
	// parameters
	Method               string            `json:"method"`
	Path                 string            `json:"path"`
	Headers              map[string]string `json:"headers,omitempty"`
	RequestBody          string            `json:"request_body,omitempty"`
	IsResponseBodyJSON   bool              `json:"is_response_body_json,omitempty"`
	ResponseBodyIDFilter string            `json:"response_body_id_filter,omitempty"`
	QueryParameters      map[string]string `json:"query_parameters,omitempty"`

	// resource-level configuration (alternative to provider-level)
	BaseURL   string            `json:"base_url,omitempty"`
	BasicAuth map[string]string `json:"basic_auth,omitempty"`
	IgnoreTLS bool              `json:"ignore_tls,omitempty"`

	// destroy controls
	IsDeleteEnabled   bool              `json:"is_delete_enabled,omitempty"`
	DeleteMethod      string            `json:"delete_method,omitempty"`
	DeletePath        string            `json:"delete_path,omitempty"`
	DeleteHeaders     map[string]string `json:"delete_headers,omitempty"`
	DeleteRequestBody string            `json:"delete_request_body,omitempty"`

	// state
	ResponseCode     int32             `json:"response_code"`
	ResponseBody     string            `json:"response_body,omitempty"`
	ResponseBodyID   string            `json:"response_body_id,omitempty"`
	ResponseBodyJSON map[string]string `json:"response_body_json,omitempty"`
}

func NewHTTPRequestResource() resource.Resource {
	return &HTTPRequestResource{}
}

func GetHTTPRequestResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Represents an HTTP request resource, allowing configuration of various " +
			"HTTP request parameters and capturing the response details.",
		MarkdownDescription: "Represents an HTTP request resource, allowing configuration of various " +
			"HTTP request parameters and capturing the response details.",
		Attributes: map[string]schema.Attribute{
			// parameters
			"method": helpers.StringAttribute(true,
				"The HTTP method to be used for the request (e.g., GET, POST, PUT, DELETE)."),
			"path": helpers.StringAttribute(
				true,
				"The URL path for the HTTP request. This should be a relative path (e.g., /api/v1/resource).",
			),
			"headers": helpers.MapAttribute(
				false,
				types.StringType,
				"A map of HTTP headers to include in the request. Each key-value pair represents a "+
					"header name and its corresponding value.",
			),
			"query_parameters": helpers.MapAttribute(false, types.StringType,
				"Optional query parameters to append to the request path"),
			"request_body": helpers.StringAttribute(
				false,
				"The body content to be sent with the HTTP request. This is typically used for POST and PUT requests.",
			),

			// resource-level configuration (alternative to provider-level)
			"base_url": helpers.StringAttribute(
				false,
				"The base URL for this specific HTTP request. When specified, this overrides the provider-level URL "+
					"configuration. This allows for different APIs to be used within the same configuration.",
			),
			"basic_auth": schema.SingleNestedAttribute{
				Description: "Credentials for basic authentication for this specific request. " +
					"When specified, this overrides the provider-level basic authentication configuration.",
				MarkdownDescription: "Credentials for basic authentication for this specific request. " +
					"When specified, this overrides the provider-level basic authentication configuration.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description:         "The username for basic authentication.",
						MarkdownDescription: "The username for basic authentication.",
						Required:            true,
					},
					"password": schema.StringAttribute{
						Description:         "The password for basic authentication.",
						MarkdownDescription: "The password for basic authentication.",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
			"ignore_tls": helpers.BoolAttribute(
				false,
				"A boolean flag to indicate whether TLS certificate verification should be ignored for this specific request. "+
					"When specified, this overrides the provider-level ignore_tls configuration.",
			),

			"is_response_body_json": helpers.BoolAttribute(
				false,
				"A boolean flag indicating whether the response body is expected to be in JSON format.",
			),
			"response_body_id_filter": helpers.StringAttribute(false,
				"A JSONPath filter used to extract a specific ID from the JSON response body. "+
					"This is useful for identifying unique elements within the response."),

			// destroy controls
			"is_delete_enabled": helpers.BoolAttribute(false,
				"Enables remote deletion during `terraform destroy`. If true and no delete_path is provided, "+
					"a DELETE will be sent to the original `path`."),
			"delete_method": helpers.StringAttribute(false,
				"HTTP method to use during deletion (e.g., DELETE, POST). Defaults to DELETE."),
			"delete_path": helpers.StringAttribute(false,
				"Path to call during deletion. Supports inline JSONPath tokens like \"/posts/$.data.id\" "+
					"evaluated against the `response_body` from create."),
			"delete_headers": helpers.MapAttribute(false, types.StringType,
				"Headers to send only during deletion."),
			"delete_request_body": helpers.StringAttribute(false,
				"Body to send only during deletion."),
			"delete_resolved_path": helpers.ComputedStringAttribute(
				"The `delete_path` with JSONPath tokens resolved from the create response, when possible."),

			// state
			// TODO: how to document the import of ths ID with examples?
			"id": helpers.ComputedStringAttribute(
				"A unique identifier for the resource. Format: `<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>`. " +
					"This is generated by encoding the entire model (excluding the ID itself) in Base64 format.",
			),
			"response_code": helpers.ComputedInt32Attribute(
				"The HTTP status code returned by the server in response to the request " +
					"(e.g., 200 for success, 404 for not found)."),
			"response_body": helpers.ComputedStringAttribute(
				"The raw body content returned by the server in response to the request."),
			"response_body_id": helpers.ComputedStringAttribute(
				"The extracted ID from the JSON response body, based on the provided " +
					"`response_body_id_filter`. This is only populated if `is_response_body_json` is true."),
			"response_body_json": helpers.ComputedMapAttribute(types.StringType,
				"The response body parsed as a Terraform map object. Nested items can be accessed "+
					"using dot notation (e.g., \"response_body_json[\"nested.item.value\"]\")."),
		},
	}
}

func (it *HTTPRequestResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_request"
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

	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint, diags := it.buildFullURL(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, err := it.buildRequest(ctx, model, endpoint)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating request. Check the method or request body informed...",
			err.Error(),
		)
		return
	}

	//nolint:bodyclose // closed in defer with error handling
	client := it.getHTTPClient(ctx, model)
	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Error executing request using HTTP client...", err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			resp.Diagnostics.AddError("Error closing the response body...", err.Error())
		}
	}(response.Body)

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the buffer from the response body...", err.Error())
		return
	}

	if !helpers.IsResponseSuccessful(response) {
		resp.Diagnostics.AddError(
			"Error performing HTTP request. Not expected status code...",
			fmt.Sprintf(
				"Response code: %s. Response responseBody: %s",
				response.Status,
				string(responseBody),
			),
		)
		return
	}

	//nolint:gosec // it's not possible integer overflow conversion int -> int32 in the default GoLang package (net/http)
	model.ResponseCode = types.Int32Value(int32(response.StatusCode))
	model.ResponseBody = types.StringValue(string(responseBody))
	updateResponseBody(&model, &resp.Diagnostics)
	updateResponseBodyID(&model, []byte(model.ResponseBody.ValueString()), &resp.Diagnostics)
	updateResponseBodyJSON(&model, []byte(model.ResponseBody.ValueString()), &resp.Diagnostics)

	if !model.DeletePath.IsNull() && model.DeletePath.ValueString() != "" {
		if resolved, ok := resolveDeletePathTokens(model.DeletePath.ValueString(), model.ResponseBody.ValueString(), &resp.Diagnostics); ok {
			model.DeleteResolvedPath = types.StringValue(resolved)
		} else {
			model.DeleteResolvedPath = types.StringNull()
		}
	} else {
		model.DeleteResolvedPath = types.StringNull()
	}

	// the ID should be the last attribute to be set
	if len(model.ID.ValueString()) == 0 {
		model.ID = types.StringValue(uuid.NewString())
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Completed HTTP request...", map[string]any{"success": true})
}

func (it *HTTPRequestResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	// TODO: should be implemented to be able to read the original source and compare with the TF state
	var model HTTPRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (it *HTTPRequestResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	it.Create(ctx, resource.CreateRequest{
		Config:       req.Config,
		Plan:         req.Plan,
		ProviderMeta: req.ProviderMeta,
	}, (*resource.CreateResponse)(resp))
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
	tflog.Debug(ctx, "DELETE response details", map[string]interface{}{
		"status": response.StatusCode,
		"body":   string(responseBody),
	})

	// Treat any non-2xx as error
	if !helpers.IsResponseSuccessful(response) {
		resp.Diagnostics.AddError(
			"DELETE request failed with unexpected status code",
			fmt.Sprintf("Response code: %s. Body: %s", response.Status, string(responseBody)),
		)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (it *HTTPRequestResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// decode the base64 input
	model := decodeImportPayloadToModel(req.ID, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// validate the mode
	if model.Method.IsNull() || model.Path.IsNull() {
		resp.Diagnostics.AddError(
			"Incomplete Model provided, please check the provided Base64 identifier...",
			"The imported model is incomplete, it's expected to have at least the method and path informed.",
		)
		return
	}

	if !model.IsResponseBodyJSON.IsUnknown() && model.IsResponseBodyJSON.ValueBool() &&
		(model.ResponseBodyIDFilter.IsUnknown() || model.ResponseBodyIDFilter.IsNull()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("response_body_id_filter"),
			"Since the response is JSON, the filter must be provided.",
			"When the expected answer is a JSON, the ID must be parsed in the state. "+
				"Please provide a filter to extract the ID from the JSON response. "+
				"Refer to the documentation for more information (https://github.com/ohler55/ojg).",
		)
	}

	// save the model in the state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), model.ID)...)
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

	if applyErr := applyHeadersFromMapAttr(ctx, req.Header, model.Headers); applyErr != nil {
		return nil, applyErr
	}

	applyDefaultJSONHeaders(req.Header, isBoolTrue(model.IsResponseBodyJSON), looksJSON)

	// Apply authentication - resource-level takes precedence over provider-level
	if !model.BasicAuth.IsNull() {
		// Use resource-level basic auth
		authAttrs := model.BasicAuth.Attributes()
		username := authAttrs["username"].(types.String).ValueString()
		password := authAttrs["password"].(types.String).ValueString()
		req.SetBasicAuth(username, password)
	} else if it.internal.Config.HasAuthentication() {
		// Fall back to provider-level basic auth
		req.SetBasicAuth(
			it.internal.Config.BasicAuth.Username,
			it.internal.Config.BasicAuth.Password,
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
		model.ResponseBodyID = types.StringValue(fmt.Sprintf("%v", element))
	} else {
		diagnostics.AddWarning("The JSON path provided didn't return any value...",
			"Please check the `response_body_id_filter` provided.")
	}
}

func unmarshalJSON(
	responseBody []byte,
	diagnostics *diag.Diagnostics,
) (map[string]interface{}, error) {
	var jsonResponse map[string]interface{}
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
		var result map[string]interface{}
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
		repl := fmt.Sprintf("%v", val)
		resolved = strings.ReplaceAll(resolved, token, repl)
	}

	return resolved, true
}

func decodeImportPayloadToModel(
	importPayload string,
	diagnostics *diag.Diagnostics,
) *HTTPRequestResourceModel {
	// Format: <RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>
	parts := strings.Split(importPayload, "/")
	if len(parts) != AmountOfPartsInID {
		diagnostics.AddError(
			"Invalid Import Identifier please check the Base64 encoding...",
			"Expected a string with the format <RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>.",
		)
		return nil
	}

	// Decode the base64 string
	modelMap, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		diagnostics.AddError(
			"Invalid Import Identifier please check the Base64 encoding...",
			fmt.Sprintf("Failed to decode Base64 identifier here is the specific cause: %v", err),
		)
		return nil
	}

	// Unmarshal the JSON to the intermediate struct
	var nativeModel HTTPRequestResourceModelNative
	if err = json.Unmarshal(modelMap, &nativeModel); err != nil {
		diagnostics.AddError(
			"Error unmarshalling the JSON to the intermediate struct...",
			err.Error(),
		)
		return nil
	}

	model := &HTTPRequestResourceModel{
		ID:     types.StringValue(parts[0]),
		Method: types.StringValue(nativeModel.Method),
		Path:   types.StringValue(nativeModel.Path),

		IsResponseBodyJSON: types.BoolValue(nativeModel.IsResponseBodyJSON),
		ResponseCode:       types.Int32Value(nativeModel.ResponseCode),

		// delete controls
		IsDeleteEnabled:   types.BoolValue(nativeModel.IsDeleteEnabled),
		DeleteMethod:      types.StringValue(nativeModel.DeleteMethod),
		DeletePath:        types.StringValue(nativeModel.DeletePath),
		DeleteRequestBody: types.StringValue(nativeModel.DeleteRequestBody),
	}
	// avoid optional values being in the state as empty (string)
	if len(nativeModel.RequestBody) > 0 {
		model.RequestBody = types.StringValue(nativeModel.RequestBody)
	}
	if len(nativeModel.ResponseBodyIDFilter) > 0 {
		model.ResponseBodyIDFilter = types.StringValue(nativeModel.ResponseBodyIDFilter)
	}
	if len(nativeModel.ResponseBody) > 0 {
		model.ResponseBody = types.StringValue(nativeModel.ResponseBody)
	}
	if len(nativeModel.ResponseBodyID) > 0 {
		model.ResponseBodyID = types.StringValue(nativeModel.ResponseBodyID)
	}
	// avoid optional values being in the state as empty (map)
	headers, diags := types.MapValueFrom(
		context.Background(),
		types.StringType,
		nativeModel.Headers,
	)
	if diags.HasError() {
		diagnostics.Append(diags...)
		return nil
	}
	model.Headers = headers

	queryParameters, diags := types.MapValueFrom(
		context.Background(),
		types.StringType,
		nativeModel.QueryParameters,
	)
	if diags.HasError() {
		diagnostics.Append(diags...)
		return nil
	}
	model.QueryParameters = queryParameters

	deleteHeaders, diags := types.MapValueFrom(
		context.Background(),
		types.StringType,
		nativeModel.DeleteHeaders,
	)
	if diags.HasError() {
		diagnostics.Append(diags...)
		return nil
	}
	model.DeleteHeaders = deleteHeaders

	responseBodyJSON, diags := types.MapValueFrom(
		context.Background(),
		types.StringType,
		nativeModel.ResponseBodyJSON,
	)
	if diags.HasError() {
		diagnostics.Append(diags...)
		return nil
	}
	model.ResponseBodyJSON = responseBodyJSON

	return model
}

func (it *HTTPRequestResource) buildFullURL(
	ctx context.Context,
	model HTTPRequestResourceModel,
) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Use resource-level base URL if provided, otherwise fall back to provider-level
	var baseURLString string
	if !model.BaseURL.IsNull() && model.BaseURL.ValueString() != "" {
		baseURLString = model.BaseURL.ValueString()
	} else if it.internal != nil && it.internal.Config != nil && it.internal.Config.URL != "" {
		baseURLString = it.internal.Config.URL
	} else {
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

// getHTTPClient returns the HTTP client to use for this request,
// taking into account resource-level TLS configuration
func (it *HTTPRequestResource) getHTTPClient(
	ctx context.Context,
	model HTTPRequestResourceModel,
) *http.Client {
	// Check if resource-level ignore_tls setting should override provider-level
	var ignoreTLS bool
	if !model.IgnoreTLS.IsNull() {
		ignoreTLS = model.IgnoreTLS.ValueBool()
	} else {
		// Fall back to provider-level setting (default is false if not set)
		if it.internal.Client.Transport != nil {
			if transport, ok := it.internal.Client.Transport.(*http.Transport); ok {
				if transport.TLSClientConfig != nil {
					ignoreTLS = transport.TLSClientConfig.InsecureSkipVerify
				}
			}
		}
	}

	// If resource-level setting matches what's already configured, reuse the client
	currentIgnoreTLS := false
	if it.internal.Client.Transport != nil {
		if transport, ok := it.internal.Client.Transport.(*http.Transport); ok {
			if transport.TLSClientConfig != nil {
				currentIgnoreTLS = transport.TLSClientConfig.InsecureSkipVerify
			}
		}
	}

	if ignoreTLS == currentIgnoreTLS {
		return it.internal.Client
	}

	// Create a new client with the desired TLS configuration
	client := &http.Client{}
	if ignoreTLS {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				// InsecureSkipVerify is intentionally set to true when ignore_tls is enabled
				// This is a user-controlled feature for testing and self-signed certificates
				//nolint:gosec // purposefully ignore TLS verification according to user configuration
				InsecureSkipVerify: true,
			},
		}
		client.Transport = transport
	}

	return client
}
