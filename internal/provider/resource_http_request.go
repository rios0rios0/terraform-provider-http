package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"net/http"
	"net/url"

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
	internal *InternalContext
}

// HTTPRequestResourceModel describes the resource data model.
type HTTPRequestResourceModel struct {
	// parameters
	Method           types.String `tfsdk:"method"`
	Path             types.String `tfsdk:"path"`
	Headers          types.Map    `tfsdk:"headers"`
	RequestBody      types.String `tfsdk:"request_body"`
	IsResponseJSON   types.Bool   `tfsdk:"is_response_json"`
	ResponseIDFilter types.String `tfsdk:"response_id_filter"`

	// state
	Id               types.String    `tfsdk:"id"`
	ResponseCode     types.Int32     `tfsdk:"response_code"`
	ResponseBody     types.String    `tfsdk:"response_body"`
	ResponseBodyJSON jsontypes.Exact `tfsdk:"response_body_json"`
}

func (it *HTTPRequestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_request"
}

func (it *HTTPRequestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "HTTP request resource",

		Attributes: map[string]schema.Attribute{
			// parameters
			"method": schema.StringAttribute{
				MarkdownDescription: "HTTP method",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "Path for the HTTP request",
				Required:            true,
			},
			"headers": schema.MapAttribute{
				MarkdownDescription: "HTTP headers",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"request_body": schema.StringAttribute{
				MarkdownDescription: "Request body",
				Optional:            true,
			},
			"is_response_body_json": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the response is JSON",
				Optional:            true,
			},
			"response_json_filter": schema.StringAttribute{
				MarkdownDescription: "Filter to extract JSON data",
				Optional:            true,
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
				MarkdownDescription: "Response code",
				Computed:            true,
			},
			"response_body": schema.StringAttribute{
				MarkdownDescription: "Response body",
				Computed:            true,
			},
			"response_body_json": schema.MapAttribute{
				MarkdownDescription: "Response body as JSON",
				Computed:            true,
				ElementType:         types.MapType{ElemType: types.StringType},
			},
		},
	}
}

func (it *HTTPRequestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "Configuring resource to use HTTP client...")

	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	internal, ok := req.ProviderData.(*InternalContext)
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

	// Retrieve values from plan
	var model HTTPRequestResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// base url and path
	var request *http.Request
	endpoint, err := url.JoinPath(it.internal.config.URL, model.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error joining the URL path, the URL + Path informed are malformed...", err.Error())
		return
	}

	// request body
	var requestBody io.Reader
	if !model.RequestBody.IsNull() {
		requestBody = bytes.NewBuffer([]byte(model.RequestBody.ValueString()))
	}
	request, err = http.NewRequestWithContext(ctx, model.Method.ValueString(), endpoint, requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request. Check the method or request body informed...", err.Error())
		return
	}

	// headers
	for key, value := range model.Headers.Elements() {
		request.Header.Set(key, value.(types.String).ValueString())
	}

	// basic auth
	if it.internal.config.HasAuthentication() {
		request.SetBasicAuth(it.internal.config.BasicAuth.Username, it.internal.config.BasicAuth.Password)
	}

	// execute the request
	client := it.internal.client
	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Error executing request using HTTP client...", err.Error())
		return
	}
	defer response.Body.Close()

	// read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the buffer from the response body...", err.Error())
		return
	}

	// avoid to change the state if the response is not successful
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		resp.Diagnostics.AddError(
			"Error performing HTTP request. Not expected status code...",
			fmt.Sprintf("Response code: %s. Response responseBody: %s", response.Status, string(responseBody)))
		return
	}

	model.ResponseCode = types.Int32Value(int32(response.StatusCode))
	model.ResponseBody = types.StringValue(string(responseBody))

	if model.IsResponseJSON.ValueBool() {
		model.ResponseBodyJSON = jsontypes.NewExactValue(string(responseBody))

		// extract the ID from the JSON response using the filter
		var jsonResponse map[string]interface{}
		if err := json.Unmarshal(responseBody, &jsonResponse); err != nil {
			resp.Diagnostics.AddError("Error unmarshalling JSON response...", err.Error())
			return
		}

		element, err := jsonpath.JsonPathLookup(jsonResponse, model.ResponseJSONFilter.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error querying JSON response with the provided filter...", err.Error())
			return
		}

		if element != nil {
			model.Id = element
		}
	}
	if model.Id.IsNull() {
		model.Id = types.StringValue(fmt.Sprintf("%s-%v", model.Method.ValueString(), model.Path.ValueString()))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
