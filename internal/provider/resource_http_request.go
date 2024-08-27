package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ohler55/ojg/jp"
	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
	"github.com/rios0rios0/terraform-provider-http/internal/infrastructure/helpers"
	"io"
	"net/http"
	"net/url"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &HTTPRequestResource{}
	_ resource.ResourceWithConfigure   = &HTTPRequestResource{}
	_ resource.ResourceWithImportState = &HTTPRequestResource{}
)

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

	// state
	ResponseCode     int32             `json:"response_code"`
	ResponseBody     string            `json:"response_body,omitempty"`
	ResponseBodyID   string            `json:"response_body_id,omitempty"`
	ResponseBodyJSON map[string]string `json:"response_body_json,omitempty"`
}

func NewHTTPRequestResource() resource.Resource {
	return &HTTPRequestResource{}
}

func (it *HTTPRequestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_request"
}

func (it *HTTPRequestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Represents an HTTP request resource, allowing configuration of various HTTP request parameters and capturing the response details.",
		MarkdownDescription: "Represents an HTTP request resource, allowing configuration of various HTTP request parameters and capturing the response details.\n\n" +
			"```hcl\n" +
			`resource "http_request" "example1" {
  method  = "GET"
  path    = "/posts/1"
  headers = {
    "Content-Type" = "application/json"
  }
  is_response_body_json = true
  response_body_id_filter = "$.id"
}

resource "http_request" "example2" {
  method  = "POST"
  path    = "/posts"
  headers = {
    "Content-Type" = "application/json"
  }
  request_body = jsonencode({
    anything = http_request.example1.response_body_json["id"]
  })
}` +
			"\n```\n\n" +
			"See complete example at the [GitHub repository](https://github.com/rios0rios0/terraform-provider-http/blob/main/examples/main.tf).",
		Attributes: map[string]schema.Attribute{
			// parameters
			"method": schema.StringAttribute{
				Required:            true,
				Description:         "The HTTP method to be used for the request (e.g., GET, POST, PUT, DELETE).",
				MarkdownDescription: "The HTTP method to be used for the request (e.g., GET, POST, PUT, DELETE).",
			},
			"path": schema.StringAttribute{
				Required:            true,
				Description:         "The URL path for the HTTP request. This should be a relative path (e.g., /api/v1/resource).",
				MarkdownDescription: "The URL path for the HTTP request. This should be a relative path (e.g., /api/v1/resource).",
			},
			"headers": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				Description:         "A map of HTTP headers to include in the request. Each key-value pair represents a header name and its corresponding value.",
				MarkdownDescription: "A map of HTTP headers to include in the request. Each key-value pair represents a header name and its corresponding value.",
			},
			"request_body": schema.StringAttribute{
				Optional:            true,
				Description:         "The body content to be sent with the HTTP request. This is typically used for POST and PUT requests.",
				MarkdownDescription: "The body content to be sent with the HTTP request. This is typically used for POST and PUT requests.",
			},
			"is_response_body_json": schema.BoolAttribute{
				Optional:            true,
				Description:         "A boolean flag indicating whether the response body is expected to be in JSON format.",
				MarkdownDescription: "A boolean flag indicating whether the response body is expected to be in JSON format.",
			},
			"response_body_id_filter": schema.StringAttribute{
				Optional:            true,
				Description:         "A JSONPath filter used to extract a specific ID from the JSON response body. This is useful for identifying unique elements within the response.",
				MarkdownDescription: "A JSONPath filter used to extract a specific ID from the JSON response body. This is useful for identifying unique elements within the response.",
			},

			// state
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "A unique identifier for the resource, generated by encoding the entire model (excluding the ID itself) in base64 format.",
				MarkdownDescription: "A unique identifier for the resource, generated by encoding the entire model (excluding the ID itself) in base64 format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"response_code": schema.Int32Attribute{
				Computed:            true,
				Description:         "The HTTP status code returned by the server in response to the request (e.g., 200 for success, 404 for not found).",
				MarkdownDescription: "The HTTP status code returned by the server in response to the request (e.g., 200 for success, 404 for not found).",
			},
			"response_body": schema.StringAttribute{
				Computed:            true,
				Description:         "The raw body content returned by the server in response to the request.",
				MarkdownDescription: "The raw body content returned by the server in response to the request.",
			},
			"response_body_id": schema.StringAttribute{
				Computed:            true,
				Description:         "The extracted ID from the JSON response body, based on the provided `response_body_id_filter`. This is only populated if `is_response_body_json` is true.",
				MarkdownDescription: "The extracted ID from the JSON response body, based on the provided `response_body_id_filter`. This is only populated if `is_response_body_json` is true.",
			},
			"response_body_json": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				Description:         `The response body parsed as a Terraform map object. Nested items can be accessed using dot notation (e.g., "response_body_json["nested.item.value"]").`,
				MarkdownDescription: `The response body parsed as a Terraform map object. Nested items can be accessed using dot notation (e.g., "response_body_json["nested.item.value"]").`,
			},
		},
	}
}

func (it *HTTPRequestResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model HTTPRequestResourceModel
	helper := helpers.NewResourceHelper()
	if !helper.RetrieveValidateConfigRequest(ctx, req, resp, &model) {
		return
	}

	if !model.IsResponseBodyJSON.IsUnknown() && model.IsResponseBodyJSON.ValueBool() &&
		(model.ResponseBodyIDFilter.IsUnknown() || model.ResponseBodyIDFilter.IsNull()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("response_id_filter"),
			"Since the response is JSON, the filter must be provided.",
			"When the expected answer is a JSON, the ID must be parsed in the state. "+
				"Please provide a filter to extract the ID from the JSON response. Refer to the documentation for more information (https://github.com/ohler55/ojg).",
		)
	}
}

func (it *HTTPRequestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			fmt.Sprintf("Expected *InternalContext, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	it.internal = internal
}

func (it *HTTPRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Starting HTTP request...")

	var model HTTPRequestResourceModel
	helper := helpers.NewResourceHelper()
	if !helper.RetrieveCreateRequest(ctx, req, resp, &model) {
		return
	}

	endpoint, err := url.JoinPath(it.internal.Config.URL, model.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error joining the URL path, the URL + Path informed are malformed...", err.Error())
		return
	}

	request, err := it.buildRequest(ctx, model, endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request. Check the method or request body informed...", err.Error())
		return
	}

	response, err := it.internal.Client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Error executing request using HTTP client...", err.Error())
		return
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the buffer from the response body...", err.Error())
		return
	}

	if !helpers.IsResponseSuccessful(response) {
		resp.Diagnostics.AddError(
			"Error performing HTTP request. Not expected status code...",
			fmt.Sprintf("Response code: %s. Response responseBody: %s", response.Status, string(responseBody)))
		return
	}

	model.ResponseCode = types.Int32Value(int32(response.StatusCode))
	model.ResponseBody = types.StringValue(string(responseBody))
	updateResponseBody(&model, &resp.Diagnostics)
	updateResponseBodyID(&model, []byte(model.ResponseBody.ValueString()), &resp.Diagnostics)
	updateResponseBodyJSON(&model, []byte(model.ResponseBody.ValueString()), &resp.Diagnostics)

	// the ID should be the last attribute to be set so it can be encoded
	model.ID = types.StringValue(encodeModelToBase64(&model, &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Completed HTTP request...", map[string]any{"success": true})
}

func (it *HTTPRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: should be implemented to be able to read the original source and compare with the TF state
	var model HTTPRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (it *HTTPRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	it.Create(ctx, resource.CreateRequest{
		Config:       req.Config,
		Plan:         req.Plan,
		ProviderMeta: req.ProviderMeta,
	}, (*resource.CreateResponse)(resp))
}

func (it *HTTPRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: should be implemented to perform a DELETE in original source (not just the TF state)
	var model HTTPRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
}

func (it *HTTPRequestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// decode the base64 input
	model := decodeBase64ToModel(req.ID, &resp.Diagnostics)
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
			path.Root("response_id_filter"),
			"Since the response is JSON, the filter must be provided.",
			"When the expected answer is a JSON, the ID must be parsed in the state. "+
				"Please provide a filter to extract the ID from the JSON response. Refer to the documentation for more information (https://github.com/ohler55/ojg).",
		)
	}

	// save the model in the state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (it *HTTPRequestResource) buildRequest(ctx context.Context, model HTTPRequestResourceModel, endpoint string) (*http.Request, error) {
	var requestBody io.Reader
	if !model.RequestBody.IsNull() {
		requestBody = bytes.NewBuffer([]byte(model.RequestBody.ValueString()))
	}
	request, err := http.NewRequestWithContext(ctx, model.Method.ValueString(), endpoint, requestBody)
	if err != nil {
		return nil, err
	}

	for key, value := range model.Headers.Elements() {
		request.Header.Set(key, value.(types.String).ValueString())
	}

	if it.internal.Config.HasAuthentication() {
		request.SetBasicAuth(it.internal.Config.BasicAuth.Username, it.internal.Config.BasicAuth.Password)
	}

	return request, nil
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

func updateResponseBodyID(model *HTTPRequestResourceModel, responseBody []byte, diagnostics *diag.Diagnostics) {
	model.ResponseBodyID = types.StringNull()
	if model.IsResponseBodyJSON.ValueBool() {
		var jsonResponse map[string]interface{}
		if err := json.Unmarshal(responseBody, &jsonResponse); err == nil {
			jsonPath, err := jp.ParseString(model.ResponseBodyIDFilter.ValueString())
			if err == nil {
				element := jsonPath.First(jsonResponse)
				if element != nil {
					model.ResponseBodyID = types.StringValue(fmt.Sprintf("%v", element))
					return
				} else {
					diagnostics.AddWarning("The JSON path provided didn't return any value...", "Please check the `response_body_id_filter` provided.")
				}
			} else {
				diagnostics.AddWarning("It wasn't possible to parse the JSON path using the `response_body_id_filter` provided...", err.Error())
			}
		} else {
			diagnostics.AddWarning("It wasn't possible to unmarshall response body to a JSON map reference...", err.Error())
		}
	}
}

func updateResponseBodyJSON(model *HTTPRequestResourceModel, responseBody []byte, diagnostics *diag.Diagnostics) {
	var diags diag.Diagnostics
	model.ResponseBodyJSON, diags = types.MapValue(types.StringType, make(map[string]attr.Value))
	diagnostics.Append(diags...)

	if model.IsResponseBodyJSON.ValueBool() {
		var result map[string]interface{}
		err := json.Unmarshal(responseBody, &result)
		if err != nil {
			diagnostics.AddError("Error unmarshalling response body to a JSON map reference...", err.Error())
		}

		model.ResponseBodyJSON, diags = types.MapValueFrom(context.Background(), types.StringType, helpers.ConvertToStringMap(result))
		diagnostics.Append(diags...)
	}
}

func encodeModelToBase64(model *HTTPRequestResourceModel, diagnostics *diag.Diagnostics) string {
	// Convert the Terraform model to a native Go struct
	modelNative := HTTPRequestResourceModelNative{
		Method: model.Method.ValueString(),
		Path:   model.Path.ValueString(),

		// all optional values are removed with "omitempty" tag
		RequestBody:          model.RequestBody.ValueString(),
		IsResponseBodyJSON:   model.IsResponseBodyJSON.ValueBool(),
		ResponseBodyIDFilter: model.ResponseBodyIDFilter.ValueString(),
		ResponseCode:         model.ResponseCode.ValueInt32(),
		ResponseBody:         model.ResponseBody.ValueString(),
		ResponseBodyID:       model.ResponseBodyID.ValueString(),
	}
	// avoid optional values being in the ID as empty (map)
	if !model.Headers.IsNull() {
		model.Headers.ElementsAs(context.Background(), &modelNative.Headers, false)
	}
	if !model.ResponseBodyJSON.IsNull() {
		model.ResponseBodyJSON.ElementsAs(context.Background(), &modelNative.ResponseBodyJSON, false)
	}

	// Marshal the map to JSON
	modelJSON, err := json.Marshal(modelNative)
	if err != nil {
		diagnostics.AddError("Error marshalling the model to JSON...", err.Error())
		return ""
	}

	var compactedJSON bytes.Buffer
	err = json.Compact(&compactedJSON, modelJSON)
	if err != nil {
		diagnostics.AddError("Error compacting JSON response body...", err.Error())
		return ""
	}

	// Encode the JSON to base64
	return base64.StdEncoding.EncodeToString(compactedJSON.Bytes())
}

func decodeBase64ToModel(modelEncoded string, diagnostics *diag.Diagnostics) *HTTPRequestResourceModel {
	// Decode the base64 string
	modelMap, err := base64.StdEncoding.DecodeString(modelEncoded)
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
		diagnostics.AddError("Error unmarshalling the JSON to the intermediate struct...", err.Error())
		return nil
	}

	model := &HTTPRequestResourceModel{
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
	headers, diags := types.MapValueFrom(context.Background(), types.StringType, nativeModel.Headers)
	if diags.HasError() {
		diagnostics.Append(diags...)
		return nil
	}
	model.Headers = headers
	responseBodyJSON, diags := types.MapValueFrom(context.Background(), types.StringType, nativeModel.ResponseBodyJSON)
	if diags.HasError() {
		diagnostics.Append(diags...)
		return nil
	}
	model.ResponseBodyJSON = responseBodyJSON

	return model
}