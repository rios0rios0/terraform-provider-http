package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	gopath "path"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
	"github.com/rios0rios0/terraform-provider-http/internal/infrastructure/helpers"
)

// Ensure HTTPRequestNonFatalResource satisfies various resources interfaces.
var (
	_ resource.Resource                = &HTTPRequestNonFatalResource{}
	_ resource.ResourceWithConfigure   = &HTTPRequestNonFatalResource{}
	_ resource.ResourceWithImportState = &HTTPRequestNonFatalResource{}
)

// HTTPRequestNonFatalResource defines the resource implementation.
type HTTPRequestNonFatalResource struct {
	internal *entities.InternalContext
}

// HTTPRequestNonFatalResourceModel describes the resource data model.
type HTTPRequestNonFatalResourceModel struct {
	// parameters
	Method               types.String `tfsdk:"method"`
	Path                 types.String `tfsdk:"path"`
	Headers              types.Map    `tfsdk:"headers"`
	RequestBody          types.String `tfsdk:"request_body"`
	IsResponseBodyJSON   types.Bool   `tfsdk:"is_response_body_json"`
	ResponseBodyIDFilter types.String `tfsdk:"response_body_id_filter"`
	QueryParameters      types.Map    `tfsdk:"query_parameters"`

	// state
	ID               types.String `tfsdk:"id"`
	ResponseCode     types.Int32  `tfsdk:"response_code"`
	ResponseBody     types.String `tfsdk:"response_body"`
	ResponseBodyID   types.String `tfsdk:"response_body_id"`
	ResponseBodyJSON types.Map    `tfsdk:"response_body_json"`
}

// HTTPRequestNonFatalResourceModelNative describes the resource data model in a native Go format.
type HTTPRequestNonFatalResourceModelNative struct {
	// parameters
	Method               string            `json:"method"`
	Path                 string            `json:"path"`
	Headers              map[string]string `json:"headers,omitempty"`
	RequestBody          string            `json:"request_body,omitempty"`
	IsResponseBodyJSON   bool              `json:"is_response_body_json,omitempty"`
	ResponseBodyIDFilter string            `json:"response_body_id_filter,omitempty"`
	QueryParameters      map[string]string `json:"query_parameters,omitempty"`

	// state
	ResponseCode     int32             `json:"response_code"`
	ResponseBody     string            `json:"response_body,omitempty"`
	ResponseBodyID   string            `json:"response_body_id,omitempty"`
	ResponseBodyJSON map[string]string `json:"response_body_json,omitempty"`
}

func NewHTTPRequestNonFatalResource() resource.Resource {
	return &HTTPRequestNonFatalResource{}
}

func GetHTTPRequestNonFatalResourceSchema() schema.Schema {
	return schema.Schema{
		Description:         "Represents an HTTP request resource (non-fatal), matching the template but treating 404s as non-fatal.",
		MarkdownDescription: "Represents an HTTP request resource (non-fatal), matching the template but treating 404s as non-fatal.",
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
				"A map of HTTP headers to include in the request.",
			),
			"query_parameters": helpers.MapAttribute(false, types.StringType,
				"Optional query parameters to append to the request path"),
			"request_body": helpers.StringAttribute(
				false,
				"The body content to be sent with the HTTP request. This is typically used for POST and PUT requests.",
			),
			"is_response_body_json": helpers.BoolAttribute(
				false,
				"A boolean flag indicating whether the response body is expected to be in JSON format.",
			),
			"response_body_id_filter": helpers.StringAttribute(false,
				"A JSONPath filter used to extract a specific ID from the JSON response body."),

			// state
			// TODO: how to document the import of ths ID with examples?
			"id": helpers.ComputedStringAttribute(
				"A unique identifier for the resource. Format: `<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>`.",
			),
			"response_code": helpers.ComputedInt32Attribute(
				"The HTTP status code returned by the server.",
			),
			"response_body": helpers.ComputedStringAttribute(
				"The raw body content returned by the server in response to the request."),
			"response_body_id": helpers.ComputedStringAttribute(
				"The extracted ID from the JSON response body, based on the provided `response_body_id_filter`."),
			"response_body_json": helpers.ComputedMapAttribute(types.StringType,
				"The response body parsed as a Terraform map object."),
		},
	}
}

func (it *HTTPRequestNonFatalResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_request_nonfatal"
}

func (it *HTTPRequestNonFatalResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = GetHTTPRequestNonFatalResourceSchema()
}

func (it *HTTPRequestNonFatalResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var model HTTPRequestNonFatalResourceModel

	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !model.IsResponseBodyJSON.IsUnknown() && model.IsResponseBodyJSON.ValueBool() &&
		(model.ResponseBodyIDFilter.IsUnknown() || model.ResponseBodyIDFilter.IsNull()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("response_id_filter"),
			"Since the response is JSON, the filter must be provided.",
			"When the expected answer is a JSON, the ID must be parsed in the state. "+
				"Please provide a filter to extract the ID from the JSON response. "+
				"Refer to the documentation for more information (https://github.com/ohler55/ojg).",
		)
	}
}

func (it *HTTPRequestNonFatalResource) Configure(
	ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	tflog.Info(ctx, "Configuring resource to use HTTP client...")

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

func (it *HTTPRequestNonFatalResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	tflog.Info(ctx, "Starting HTTP request (non-fatal)...")

	var model HTTPRequestNonFatalResourceModel

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
	response, err := it.internal.Client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Error executing request using HTTP client...", err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		if cerr := Body.Close(); cerr != nil {
			resp.Diagnostics.AddError("Error closing the response body...", cerr.Error())
		}
	}(response.Body)

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the buffer from the response body...", err.Error())
		return
	}

	// Non-fatal logic: allow 404, fail other non-2xx
	if !helpers.IsResponseSuccessful(response) && response.StatusCode != http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Error performing HTTP request. Not expected status code...",
			fmt.Sprintf("Response code: %s. Response responseBody: %s", response.Status, string(responseBody)),
		)
		return
	}

	//nolint:gosec
	model.ResponseCode = types.Int32Value(int32(response.StatusCode))
	model.ResponseBody = types.StringValue(string(responseBody))
	updateNonFatalResponseBody(&model, &resp.Diagnostics)
	updateNonFatalResponseBodyID(&model, []byte(model.ResponseBody.ValueString()), &resp.Diagnostics)
	updateNonFatalResponseBodyJSON(&model, []byte(model.ResponseBody.ValueString()), &resp.Diagnostics)

	if len(model.ID.ValueString()) == 0 {
		model.ID = types.StringValue(uuid.NewString())
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Completed HTTP request (non-fatal)...", map[string]any{"success": true})
}

func (it *HTTPRequestNonFatalResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var model HTTPRequestNonFatalResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (it *HTTPRequestNonFatalResource) Update(
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

	// TODO: should be implemented to perform a DELETE in original source (not just the TF state)
func (it *HTTPRequestNonFatalResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var model HTTPRequestNonFatalResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
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
		resp.Diagnostics.AddError("Error creating DELETE request", err.Error())
		return
	}
	request.Method = http.MethodDelete

	response, err := it.internal.Client.Do(request)
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
	tflog.Debug(ctx, "Response details (non-fatal delete)", map[string]interface{}{
		"status": response.StatusCode,
		"body":   string(responseBody),
	})

	if !helpers.IsResponseSuccessful(response) && response.StatusCode != http.StatusNotFound {
		resp.Diagnostics.AddError(
			"DELETE request failed with unexpected status code",
			fmt.Sprintf("Response code: %s. Body: %s", response.Status, string(responseBody)),
		)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (it *HTTPRequestNonFatalResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// decode the base64 input
	model := decodeNonFatalImportPayloadToModel(req.ID, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// validate the model
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
			path.Root("response_id_filter"),
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

func (it *HTTPRequestNonFatalResource) buildRequest(
	ctx context.Context, model HTTPRequestNonFatalResourceModel, endpoint string,
) (*http.Request, error) {
	var requestBody io.Reader
	if !model.RequestBody.IsNull() {
		requestBody = bytes.NewBufferString(model.RequestBody.ValueString())
	}
	request, err := http.NewRequestWithContext(
		ctx,
		model.Method.ValueString(),
		endpoint,
		requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	for key, value := range model.Headers.Elements() {
		if !value.IsUnknown() && !value.IsNull() {
			//nolint:errcheck // checked already via SDK state (line before)
			request.Header.Set(key, value.(types.String).ValueString())
		}
	}

	if it.internal.Config.HasAuthentication() {
		request.SetBasicAuth(
			it.internal.Config.BasicAuth.Username,
			it.internal.Config.BasicAuth.Password,
		)
	}

	return request, nil
}

func updateNonFatalResponseBody(model *HTTPRequestNonFatalResourceModel, diagnostics *diag.Diagnostics) {
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

func updateNonFatalResponseBodyID(
	model *HTTPRequestNonFatalResourceModel,
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

func updateNonFatalResponseBodyJSON(
	model *HTTPRequestNonFatalResourceModel,
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

func decodeNonFatalImportPayloadToModel(
	importPayload string,
	diagnostics *diag.Diagnostics,
) *HTTPRequestNonFatalResourceModel {
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
	var nativeModel HTTPRequestNonFatalResourceModelNative
	if err = json.Unmarshal(modelMap, &nativeModel); err != nil {
		diagnostics.AddError(
			"Error unmarshalling the JSON to the intermediate struct...",
			err.Error(),
		)
		return nil
	}

	model := &HTTPRequestNonFatalResourceModel{
		ID:     types.StringValue(parts[0]),
		Method: types.StringValue(nativeModel.Method),
		Path:   types.StringValue(nativeModel.Path),

		IsResponseBodyJSON: types.BoolValue(nativeModel.IsResponseBodyJSON),
		ResponseCode:       types.Int32Value(nativeModel.ResponseCode),
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

func (it *HTTPRequestNonFatalResource) buildFullURL(
	ctx context.Context,
	model HTTPRequestNonFatalResourceModel,
) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	baseURL, err := url.Parse(it.internal.Config.URL)
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
