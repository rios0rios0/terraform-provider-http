package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ohler55/ojg/jp"
	"github.com/rios0rios0/terraform-provider-http/internal/domain/entities"
	"github.com/rios0rios0/terraform-provider-http/internal/infrastructure/helpers"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &HTTPRequestResource{}
	_ resource.ResourceWithConfigure   = &HTTPRequestResource{}
	_ resource.ResourceWithImportState = &HTTPRequestResource{}
)

func NewHTTPRequestResource() resource.Resource {
	return &HTTPRequestResource{}
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

	// state
	Id           types.String `tfsdk:"id"`
	ResponseCode types.Int32  `tfsdk:"response_code"`
	ResponseBody types.String `tfsdk:"response_body"`
	//ResponseBodyJSON types.Map    `tfsdk:"response_body_json"`
}

func (it *HTTPRequestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_request"
}

func (it *HTTPRequestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "HTTP request resource",

		Attributes: map[string]schema.Attribute{
			// parameters
			"method": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "HTTP method",
			},
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path for the HTTP request",
			},
			"headers": schema.MapAttribute{
				Optional:            true,
				MarkdownDescription: "HTTP headers",
				ElementType:         types.StringType,
			},
			"request_body": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Request body",
			},
			"is_response_body_json": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Indicates if the response is JSON",
			},
			"response_body_id_filter": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter to extract JSON data",
			},

			// state
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Resource identifier",
				MarkdownDescription: "Resource identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"response_code": schema.Int32Attribute{
				Computed:            true,
				MarkdownDescription: "Response code",
			},
			"response_body": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Response body",
			},
			//"response_body_json": schema.MapAttribute{
			//	Computed:            true,
			//	ElementType:         types.StringType,
			//	MarkdownDescription: "Response body as JSON",
			//},
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

	if !isResponseSuccessful(response) {
		resp.Diagnostics.AddError(
			"Error performing HTTP request. Not expected status code...",
			fmt.Sprintf("Response code: %s. Response responseBody: %s", response.Status, string(responseBody)))
		return
	}

	updateModelWithID(&model, responseBody)
	updateModelWithResponse(ctx, &model, response, responseBody)

	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Completed HTTP request...", map[string]any{"success": true})
}

func (it *HTTPRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HTTPRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (it *HTTPRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	it.Create(ctx, resource.CreateRequest{
		Config:       req.Config,
		Plan:         req.Plan,
		ProviderMeta: req.ProviderMeta,
	}, (*resource.CreateResponse)(resp))
}

func (it *HTTPRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HTTPRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
}

func (it *HTTPRequestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: method,path. Got: %q", req.ID),
		)
		return
	}

	requestMethod := parts[0]
	requestPath := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("method"), requestMethod)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), requestPath)...)
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

func isResponseSuccessful(response *http.Response) bool {
	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}

func updateModelWithID(model *HTTPRequestResourceModel, responseBody []byte) {
	if model.IsResponseBodyJSON.ValueBool() {
		var jsonResponse map[string]interface{}
		if err := json.Unmarshal(responseBody, &jsonResponse); err == nil {
			jsonPath, err := jp.ParseString(model.ResponseBodyIDFilter.ValueString())
			if err == nil {
				element := jsonPath.Get(jsonResponse)
				if element != nil {
					model.Id = types.StringValue(fmt.Sprintf("%s", element))
					return
				}
			}
		}
	}

	model.Id = types.StringValue(fmt.Sprintf("%s,%s", model.Method.ValueString(), model.Path.ValueString()))
}

func updateModelWithResponse(ctx context.Context, model *HTTPRequestResourceModel, response *http.Response, responseBody []byte) {
	model.ResponseCode = types.Int32Value(int32(response.StatusCode))
	model.ResponseBody = types.StringValue(string(responseBody))
	//if model.IsResponseBodyJSON.ValueBool() {
	//var diags diag.Diagnostics
	//model.ResponseBodyJSON, diags = types.MapValueFrom(ctx, types.StringType, string(responseBody))
	//if diags.HasError() {
	//	tflog.Error(ctx, "Error parsing response body as JSON...", map[string]any{"error": diags})
	//}
	//}
}
