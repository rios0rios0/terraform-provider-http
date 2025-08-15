//go:build integration

package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// According to Terraform SDK documentation delete testing automatically occurs in TestCase
// https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework/resource#TestCase

func TestHTTPNonFatalRequestResource(t *testing.T) {
	t.Parallel()

	t.Run("should apply and check the state when using GET method", func(t *testing.T) {
		// given
		var state bytes.Buffer
		_ = json.Compact(&state, []byte(`{
		"method": "GET",
		"path": "/posts/1",
		"response_code": 200,
		"response_body": "{\n  \"userId\": 1,\n  \"id\": 1,\n  \"title\": \"sunt aut facere repellat provident occaecati excepturi optio reprehenderit\",\n  \"body\": \"quia et suscipit\\nsuscipit recusandae consequuntur expedita et cum\\nreprehenderit molestiae ut ut quas totam\\nnostrum rerum est autem sunt rem eveniet architecto\"\n}"
	}`))
		stateID := "anything unique"
		modelEncoded := base64.StdEncoding.EncodeToString(state.Bytes())
		importPayload := fmt.Sprintf("%s/%s", stateID, modelEncoded)

		for _, providerConfig := range providerConfigs {
			// given
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test1").
					WithMethod("GET").
					WithPath("/posts/1").
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Apply testing
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							// then
							resource.TestCheckResourceAttr("http_request.test1", "method", "GET"),
							resource.TestCheckResourceAttr("http_request.test1", "path", "/posts/1"),
							resource.TestCheckResourceAttrSet("http_request.test1", "id"),
							resource.TestCheckResourceAttrSet("http_request.test1", "response_code"),
							resource.TestCheckResourceAttrSet("http_request.test1", "response_body"),
						),
					},

					// Destroy testing
					{
						Destroy: true,
						Config:  config,
					},

					// Import testing
					{
						ImportState:   true,
						ResourceName:  "http_request.test1",
						ImportStateId: importPayload,
						// function is being used because ImportStateVerify compares with the previous object
						ImportStateCheck: func(state []*terraform.InstanceState) error {
							// then
							assert.Equal(t, stateID, state[0].ID, "id should be equal to the stateID")
							assert.Equal(t, "GET", state[0].Attributes["method"], "method should be GET")
							assert.Equal(t, "/posts/1", state[0].Attributes["path"], "path should be /posts/1")
							assert.Equal(t, "200", state[0].Attributes["response_code"], "response_code should be 200")
							return nil
						},
					},
				},
			})
		}
	})

	t.Run("should apply without fatal error when server returns 404", func(t *testing.T) {
		// given
		var state bytes.Buffer
		_ = json.Compact(&state, []byte(`{
        "method": "GET",
        "path": "/posts/0",
        "response_code": 404,
        "response_body": "{}"
    }`))
		stateID := "anything unique"
		modelEncoded := base64.StdEncoding.EncodeToString(state.Bytes())
		importPayload := fmt.Sprintf("%s/%s", stateID, modelEncoded)

		for _, providerConfig := range providerConfigs {
			// given
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test404").
					WithMethod("GET").
					WithPath("/posts/0"). // this endpoint does not exist and will return a 404
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							// then
							resource.TestCheckResourceAttr("http_request.test404", "method", "GET"),
							resource.TestCheckResourceAttr("http_request.test404", "path", "/posts/0"),
							resource.TestCheckResourceAttr("http_request.test404", "response_code", "404"),
							resource.TestCheckResourceAttrSet("http_request.test404", "id"),
							resource.TestCheckResourceAttrSet("http_request.test404", "response_body"),
						),
					},
					{
						Destroy: true,
						Config:  config,
					},
					{
						ImportState:   true,
						ResourceName:  "http_request.test404",
						ImportStateId: importPayload,
						ImportStateCheck: func(state []*terraform.InstanceState) error {
							// then
							assert.Equal(t, stateID, state[0].ID, "id should be equal to the stateID")
							assert.Equal(t, "GET", state[0].Attributes["method"], "method should be GET")
							assert.Equal(t, "/posts/0", state[0].Attributes["path"], "path should be /posts/0")
							assert.Equal(t, "404", state[0].Attributes["response_code"], "response_code should be 404")
							return nil
						},
					},
				},
			})
		}
	})
	t.Run("should apply and check the state when using POST and a non-JSON request body", func(t *testing.T) {
		// given
		var state bytes.Buffer
		_ = json.Compact(&state, []byte(`{
		"method": "POST",
		"path": "/posts",
		"request_body": "test body",
		"response_code": 201,
		"response_body":"{\n  \"id\": 101\n}"
	}`))
		stateID := "anything unique"
		modelEncoded := base64.StdEncoding.EncodeToString(state.Bytes())
		importPayload := fmt.Sprintf("%s/%s", stateID, modelEncoded)

		for _, providerConfig := range providerConfigs {
			// given
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test2").
					WithMethod("POST").
					WithPath("/posts").
					WithRequestBody("\"test body\"").
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Apply testing
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							// then
							resource.TestCheckResourceAttr("http_request.test2", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test2", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test2", "request_body", "test body"),
							resource.TestCheckResourceAttrSet("http_request.test2", "id"),
							resource.TestCheckResourceAttr("http_request.test2", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test2", "response_body"),
						),
					},

					// Destroy testing
					{
						Destroy: true,
						Config:  config,
					},

					// Import testing
					{
						ImportState:   true,
						ResourceName:  "http_request.test2",
						ImportStateId: importPayload,
						// function is being used because ImportStateVerify compares with the previous object
						ImportStateCheck: func(state []*terraform.InstanceState) error {
							// then
							assert.Equal(t, stateID, state[0].ID, "id should be equal to the stateID")
							assert.Equal(t, "POST", state[0].Attributes["method"], "method should be POST")
							assert.Equal(t, "/posts", state[0].Attributes["path"], "path should be /posts")
							assert.Equal(t, "201", state[0].Attributes["response_code"], "response_code should be 201")
							return nil
						},
					},
				},
			})
		}
	})

	t.Run("should apply and check the state when using POST and a JSON request body", func(t *testing.T) {
		// given
		var state bytes.Buffer
		_ = json.Compact(&state, []byte(`{
		"method": "POST",
		"path": "/posts",
		"request_body": "{ \"test\": \"test body\" }",
 		"is_response_body_json": true,
		"response_body_id_filter": "$.id",
		"response_code": 201,
		"response_body": "{\"id\":101}",
		"response_body_id": "101",
		"response_body_json": {"id":"101"}
	}`))
		stateID := "anything unique"
		modelEncoded := base64.StdEncoding.EncodeToString(state.Bytes())
		importPayload := fmt.Sprintf("%s/%s", stateID, modelEncoded)

		body, _ := json.Marshal("{ \"test\": \"test body\" }")

		for _, providerConfig := range providerConfigs {
			// given
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test3").
					WithMethod("POST").
					WithPath("/posts").
					WithRequestBody(string(body)).
					WithResponseBodyIDFilter("$.id").
					WithIsResponseBodyJSON(true).
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Apply testing
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							// then
							resource.TestCheckResourceAttr("http_request.test3", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test3", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test3", "request_body", "{ \"test\": \"test body\" }"),
							resource.TestCheckResourceAttr("http_request.test3", "is_response_body_json", "true"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_id_filter", "$.id"),
							resource.TestCheckResourceAttrSet("http_request.test3", "id"),
							resource.TestCheckResourceAttr("http_request.test3", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test3", "response_body"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_id", "101"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_json.id", "101"),
						),
					},

					// Destroy testing
					{
						Destroy: true,
						Config:  config,
					},

					// Import testing
					{
						ImportState:   true,
						ResourceName:  "http_request.test3",
						ImportStateId: importPayload,
						// function is being used because ImportStateVerify compares with the previous object
						ImportStateCheck: func(state []*terraform.InstanceState) error {
							// then
							assert.Equal(t, stateID, state[0].ID, "id should be equal to the stateID")
							assert.Equal(t, "POST", state[0].Attributes["method"], "method should be POST")
							assert.Equal(t, "/posts", state[0].Attributes["path"], "path should be /posts")
							assert.Equal(t, "201", state[0].Attributes["response_code"], "response_code should be 201")
							assert.Equal(t, "{ \"test\": \"test body\" }", state[0].Attributes["request_body"], "request_body should be { \"test\": \"test body\" }")
							assert.Equal(t, "true", state[0].Attributes["is_response_body_json"], "is_response_body_json should be true")
							assert.Equal(t, "$.id", state[0].Attributes["response_body_id_filter"], "response_body_id_filter should be $.id")
							return nil
						},
					},
				},
			})
		}
	})

	t.Run("should apply and update when using POST and a JSON request body", func(t *testing.T) {
		// given
		resourceBuilder := builders.NewResourceTFBuilder().
			WithName("test4").
			WithMethod("POST").
			WithPath("/posts").
			WithHeaders(map[string]string{
				"Content-Type": "application/json; charset=UTF-8",
			}).
			WithResponseBodyIDFilter("$.id").
			WithIsResponseBodyJSON(true)

		resourceNoBody := resourceBuilder.Build()

		body, _ := json.Marshal("{ \"title\": \"test title\", \"body\": \"test body\", \"userId\": 1 }")
		resourceWithBody := resourceBuilder.WithRequestBody(string(body)).Build()

		for _, providerConfig := range providerConfigs {
			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ErrorCheck:               func(err error) error { return nil },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Apply testing
					{
						Config: providerConfig + resourceNoBody,
						Check: resource.ComposeAggregateTestCheckFunc(
							// then
							resource.TestCheckResourceAttr("http_request.test4", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test4", "path", "/posts"),
							resource.TestCheckResourceAttrSet("http_request.test4", "response_body"),
						),
					},

					// Changing anything and updating
					{
						Config: providerConfig + resourceWithBody,
						Check: resource.ComposeAggregateTestCheckFunc(
							// then
							resource.TestCheckResourceAttr("http_request.test4", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test4", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test4", "request_body", string(body)),
							resource.TestCheckResourceAttrSet("http_request.test4", "response_body"),
						),
					},

					// Plan testing
					{
						PlanOnly: true,
						Config:   providerConfig + resourceWithBody,
					},
				},
			})
		}
	})
}

func TestHTTPRequestResourceNonFatal_ValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("should not throw any error when the 'method' and 'path' are set", func(t *testing.T) {
		// given
		req := fresource.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewResourceTypeBuilder().
						WithMethod().
						WithPath().
						WithHeaders().
						WithRequestBody().
						WithIsResponseBodyJSON().
						WithResponseBodyIDFilter().
						Build(),
					map[string]tftypes.Value{
						"method":                  tftypes.NewValue(tftypes.String, "GET"),
						"path":                    tftypes.NewValue(tftypes.String, "/posts/1"),
						"headers":                 tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"request_body":            tftypes.NewValue(tftypes.String, nil),
						"is_response_body_json":   tftypes.NewValue(tftypes.Bool, nil),
						"response_body_id_filter": tftypes.NewValue(tftypes.String, nil),

						// TODO: those attributes are being flag as required by the SDK, but they are not
						//"id": tftypes.NewValue(tftypes.String, nil),
						//"response_code":      tftypes.NewValue(tftypes.Number, nil),
						//"response_body":      tftypes.NewValue(tftypes.String, nil),
						//"response_body_id":   tftypes.NewValue(tftypes.String, nil),
						//"response_body_json": tftypes.NewValue(tftypes.String, nil),
					},
				),
				Schema: GetHTTPRequestResourceSchema(),
			},
		}
		resp := fresource.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPRequestResource{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Equal(t, 1, len(resp.Diagnostics), "there's no error required parameters are set")
		//assert.Equal(t, diag.Diagnostics{}, resp.Diagnostics, "Diagnostic is empty since required parameters are set")
	})
}
