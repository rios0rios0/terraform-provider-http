package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	client *http.Client           // TODO: the client should be created in the provider and not here in this file
	config map[string]interface{} // TODO: it should be a struct well defined in the provider
}

// HTTPRequestResourceModel describes the resource data model.
type HTTPRequestResourceModel struct {
	Path         types.String `tfsdk:"path"`
	Method       types.String `tfsdk:"method"`
	Headers      types.Map    `tfsdk:"headers"`
	RequestBody  types.String `tfsdk:"request_body"`
	IsJSON       types.Bool   `tfsdk:"is_json"`
	ResponseBody types.String `tfsdk:"response_body"`
	//ResponseBodyJSON types.Map    `tfsdk:"response_body_json"` TODO: uncomment this line
	ResponseCode types.Int32  `tfsdk:"response_code"`
	Id           types.String `tfsdk:"id"`
}

func (it *HTTPRequestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_request"
}

func (it *HTTPRequestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "HTTP request resource",

		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "Path for the HTTP request",
				Required:            true,
			},
			"method": schema.StringAttribute{
				MarkdownDescription: "HTTP method",
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
			"is_json": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the response is JSON",
				Optional:            true,
				//Default:             false,
			},
			"response_body": schema.StringAttribute{
				MarkdownDescription: "Response body",
				Computed:            true,
			},
			// TODO: uncomment this block
			//"response_body_json": schema.MapAttribute{
			//	MarkdownDescription: "Response body as JSON",
			//	Computed:            true,
			//	ElementType:         types.MapType{ElemType: types.StringType},
			//},
			"response_code": schema.Int32Attribute{
				MarkdownDescription: "Response code",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (it *HTTPRequestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected map[string]interface{}, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	it.config = config
}

func (it *HTTPRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model HTTPRequestResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := it.config["url"].(string)
	ignoreTLS := it.config["ignore_tls"].(bool)

	var err error
	var request *http.Request
	if model.RequestBody.IsNull() {
		request, err = http.NewRequest(model.Method.ValueString(), baseURL+model.Path.ValueString(), nil)
	} else {
		request, err = http.NewRequest(model.Method.ValueString(), baseURL+model.Path.ValueString(), bytes.NewBuffer([]byte(model.RequestBody.ValueString())))
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}

	for key, value := range model.Headers.Elements() {
		request.Header.Set(key, value.(types.String).ValueString())
	}

	if value, ok := it.config["basic_auth"]; ok {
		auth := value.(types.Object)
		username := auth.Attributes()["username"].(types.String).ValueString()
		password := auth.Attributes()["password"].(types.String).ValueString()
		request.SetBasicAuth(username, password)
	}

	client := &http.Client{}
	if ignoreTLS {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client.Transport = transport
	}

	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Error executing using default HTTP client", err.Error())
		return
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the buffer from the response responseBody", err.Error())
		return
	}

	// avoid to change the state if the response is not successful
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		resp.Diagnostics.AddError(
			"Error performing HTTP request. Not expected status code...",
			fmt.Sprintf("Response code: %s. Response responseBody: %s", response.Status, string(responseBody)))
		return
	}

	model.ResponseCode = types.Int32Value(int32(response.StatusCode))
	model.ResponseBody = types.StringValue(string(responseBody))

	if model.IsJSON.ValueBool() {
		var jsonBody map[string]interface{}
		if err := json.Unmarshal(responseBody, &jsonBody); err != nil {
			resp.Diagnostics.AddError("Error parsing JSON response", err.Error())
			return
		}
		//model.ResponseBodyJSON, _ = types.ObjectValueFrom(ctx, types.StringType, jsonBody) TODO: uncomment this line
	}

	model.Id = types.StringValue(model.Path.ValueString())
	tflog.Trace(ctx, "created a resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
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
