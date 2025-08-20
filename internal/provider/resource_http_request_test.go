//go:build integration

package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var providerConfigs = []string{
	builders.NewProviderTFBuilder().WithURL("https://jsonplaceholder.typicode.com").
		WithBasicAuth("***", "***").
		WithIgnoreTLS(true).
		Build(),
	builders.NewProviderTFBuilder().WithURL("https://jsonplaceholder.typicode.com").
		WithIgnoreTLS(true).
		Build(),
	builders.NewProviderTFBuilder().WithURL("https://jsonplaceholder.typicode.com").
		WithBasicAuth("***", "***").
		Build(),
	builders.NewProviderTFBuilder().WithURL("https://jsonplaceholder.typicode.com").
		Build(),
}

func TestHTTPRequestResource(t *testing.T) {
	t.Parallel()

	t.Run("should apply and check the state when using GET method", func(t *testing.T) {
		// given
		var state bytes.Buffer
		_ = json.Compact(&state, []byte(`{
		"method": "GET",
		"path": "/posts/1",
		"query_parameters": {},
		"response_code": 200,
		"response_body": "{\n  \"userId\": 1,\n  \"id\": 1,\n  \"title\": \"sunt aut facere repellat provident occaecati excepturi optio reprehenderit\",\n  \"body\": \"quia et suscipit\\nsuscipit recusandae consequuntur expedita et cum\\nreprehenderit molestiae ut ut quas totam\\nnostrum rerum est autem sunt rem eveniet architecto\"\n}"
	}`))
		stateID := "unique"
		modelEncoded := base64.StdEncoding.EncodeToString(state.Bytes())
		importPayload := fmt.Sprintf("%s/%s", stateID, modelEncoded)

		for _, providerConfig := range providerConfigs {
			// given
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test1").
					WithMethod("GET").
					WithPath("/posts/1").
					WithQueryParameters(map[string]string{}).
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

	t.Run("should apply and check the state when using POST and a non-JSON request body", func(t *testing.T) {
		// given
		var state bytes.Buffer
		_ = json.Compact(&state, []byte(`{
		"method": "POST",
		"path": "/posts",
		"request_body": "test body",
		"query_parameters": {},
		"response_code": 201,
		"response_body":"{\n  \"id\": 101\n}"
	}`))
		stateID := "unique"
		modelEncoded := base64.StdEncoding.EncodeToString(state.Bytes())
		importPayload := fmt.Sprintf("%s/%s", stateID, modelEncoded)

		for _, providerConfig := range providerConfigs {
			// given
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test2").
					WithMethod("POST").
					WithPath("/posts").
					WithRequestBody(strconv.Quote("test body")).
					WithQueryParameters(map[string]string{}).
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
							resource.TestCheckResourceAttr("http_request.test2", "query_parameters.%", "0"),
						),
					},

					{
						Destroy: true,
						Config:  config,
					},

					{
						ImportState:   true,
						ResourceName:  "http_request.test2",
						ImportStateId: importPayload,
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
		"request_body": "{\"test\":\"test body\"}",
 		"is_response_body_json": true,
		"response_body_id_filter": "$.id",
		"response_code": 201,
		"response_body": "{\"id\":101}",
		"response_body_id": "101",
		"response_body_json": {"id":"101"}
	}`))
		stateID := "unique"
		modelEncoded := base64.StdEncoding.EncodeToString(state.Bytes())
		importPayload := fmt.Sprintf("%s/%s", stateID, modelEncoded)

		body, _ := json.Marshal(map[string]any{"test": "test body"})

		for _, providerConfig := range providerConfigs {
			// given
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test3").
					WithMethod("POST").
					WithPath("/posts").
					WithRequestBody(strconv.Quote(string(body))).
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
							resource.TestCheckResourceAttr("http_request.test3", "request_body", string(body)),
							resource.TestCheckResourceAttr("http_request.test3", "is_response_body_json", "true"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_id_filter", "$.id"),
							resource.TestCheckResourceAttrSet("http_request.test3", "id"),
							resource.TestCheckResourceAttr("http_request.test3", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test3", "response_body"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_id", "101"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_json.id", "101"),
						),
					},

					// Destroy testing - Skip for POST requests as JSONPlaceholder doesn't support DELETE
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
							assert.Equal(t, "{\"test\":\"test body\"}", state[0].Attributes["request_body"], "request_body should be { \"test\": \"test body\" }")
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
		resourceWithBody := resourceBuilder.WithRequestBody(strconv.Quote(string(body))).Build()

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
					{
						PlanOnly: true,
						Config:   providerConfig + resourceWithBody,
					},
				},
			})
		}
	})

	t.Run("should apply and destroy with is_delete_enabled = true using default DELETE method", func(t *testing.T) {
		// given
		for _, providerConfig := range providerConfigs {
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test_delete").
					WithMethod("POST").
					WithPath("/posts").
					WithHeaders(map[string]string{
						"Content-Type": "application/json",
					}).
					WithRequestBody(strconv.Quote(`{"title": "test delete", "body": "test body", "userId": 1}`)).
					WithIsResponseBodyJSON(true).
					WithResponseBodyIDFilter("$.id").
					WithIsDeleteEnabled(true).
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Create and verify
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test_delete", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test_delete", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test_delete", "is_delete_enabled", "true"),
							resource.TestCheckResourceAttrSet("http_request.test_delete", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete", "response_body_id", "101"),
						),
					},
					// Destroy testing - this will attempt actual DELETE to /posts when is_delete_enabled = true
					{
						Destroy: true,
						Config:  config,
					},
				},
			})
		}
	})

	t.Run("should apply and destroy with custom delete_path using JSONPath token", func(t *testing.T) {
		// given
		for _, providerConfig := range providerConfigs {
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test_delete_custom_path").
					WithMethod("POST").
					WithPath("/posts").
					WithHeaders(map[string]string{
						"Content-Type": "application/json",
					}).
					WithRequestBody(strconv.Quote(`{"title": "test delete custom path", "body": "test body", "userId": 1}`)).
					WithIsResponseBodyJSON(true).
					WithResponseBodyIDFilter("$.id").
					WithIsDeleteEnabled(true).
					WithDeletePath("/posts/$.id").
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Create and verify
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "is_delete_enabled", "true"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "delete_path", "/posts/$.id"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_path", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_path", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "response_body_id", "101"),
							// Verify that delete_resolved_path is computed with the ID from the response
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "delete_resolved_path", "/posts/101"),
						),
					},
					// Destroy testing - this will attempt DELETE to /posts/101
					{
						Destroy: true,
						Config:  config,
					},
				},
			})
		}
	})

	t.Run("should apply and destroy with custom delete_method, headers, and body", func(t *testing.T) {
		// given
		for _, providerConfig := range providerConfigs {
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test_delete_custom_all").
					WithMethod("POST").
					WithPath("/posts").
					WithHeaders(map[string]string{
						"Content-Type": "application/json",
					}).
					WithRequestBody(strconv.Quote(`{"title": "test delete custom all", "body": "test body", "userId": 1}`)).
					WithIsResponseBodyJSON(true).
					WithResponseBodyIDFilter("$.id").
					WithIsDeleteEnabled(true).
					WithDeleteMethod("POST").
					WithDeletePath("/posts/$.id/archive").
					WithDeleteHeaders(map[string]string{
						"X-Delete-Reason": "terraform-destroy",
						"Content-Type":    "application/json",
					}).
					WithDeleteRequestBody(strconv.Quote(`{"reason": "terraform destroy", "actor": "automation"}`)).
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Create and verify
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "is_delete_enabled", "true"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_method", "POST"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_path", "/posts/$.id/archive"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_headers.X-Delete-Reason", "terraform-destroy"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_headers.Content-Type", "application/json"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_request_body", `{"reason": "terraform destroy", "actor": "automation"}`),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_all", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_all", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "response_body_id", "101"),
							// Verify that delete_resolved_path is computed with the ID from the response
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_resolved_path", "/posts/101/archive"),
						),
					},
					// Destroy testing - this will attempt POST to /posts/101/archive with custom headers and body
					{
						Destroy: true,
						Config:  config,
					},
				},
			})
		}
	})

	t.Run("should apply and destroy with is_delete_enabled = false (state-only destruction)", func(t *testing.T) {
		// given
		for _, providerConfig := range providerConfigs {
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test_delete_disabled").
					WithMethod("POST").
					WithPath("/posts").
					WithHeaders(map[string]string{
						"Content-Type": "application/json",
					}).
					WithRequestBody(strconv.Quote(`{"title": "test delete disabled", "body": "test body", "userId": 1}`)).
					WithIsResponseBodyJSON(true).
					WithResponseBodyIDFilter("$.id").
					WithIsDeleteEnabled(false).
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Create and verify
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test_delete_disabled", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test_delete_disabled", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test_delete_disabled", "is_delete_enabled", "false"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_disabled", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete_disabled", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_disabled", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete_disabled", "response_body_id", "101"),
						),
					},
					// Destroy testing - this should only remove from state, no HTTP request
					{
						Destroy: true,
						Config:  config,
					},
				},
			})
		}
	})

	t.Run("should apply and destroy with GET method and delete enabled", func(t *testing.T) {
		// given
		for _, providerConfig := range providerConfigs {
			config := providerConfig +
				builders.NewResourceTFBuilder().
					WithName("test_delete_get").
					WithMethod("GET").
					WithPath("/posts/1").
					WithIsDeleteEnabled(true).
					WithDeletePath("/posts/1").
					Build()

			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// Create and verify
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test_delete_get", "method", "GET"),
							resource.TestCheckResourceAttr("http_request.test_delete_get", "path", "/posts/1"),
							resource.TestCheckResourceAttr("http_request.test_delete_get", "is_delete_enabled", "true"),
							resource.TestCheckResourceAttr("http_request.test_delete_get", "delete_path", "/posts/1"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_get", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete_get", "response_code", "200"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_get", "response_body"),
						),
					},
					// Destroy testing - this will attempt DELETE to /posts/1
					{
						Destroy: true,
						Config:  config,
					},
				},
			})
		}
	})
}

func TestHTTPRequestResource_ValidateConfig(t *testing.T) {
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
						WithQueryParameters().
						WithIsDeleteEnabled().
						WithDeleteMethod().
						WithDeletePath().
						WithDeleteHeaders().
						WithDeleteRequestBody().
						WithDeleteResolvedPath().
						WithID().
						WithResponseCode().
						WithResponseBody().
						WithResponseBodyID().
						WithResponseBodyJSON().
						Build(),
					map[string]tftypes.Value{
						"method":                  tftypes.NewValue(tftypes.String, "GET"),
						"path":                    tftypes.NewValue(tftypes.String, "/posts/1"),
						"headers":                 tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"request_body":            tftypes.NewValue(tftypes.String, nil),
						"is_response_body_json":   tftypes.NewValue(tftypes.Bool, false),
						"response_body_id_filter": tftypes.NewValue(tftypes.String, nil),
						"query_parameters":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),

						// Destroy controls
						"is_delete_enabled":    tftypes.NewValue(tftypes.Bool, nil),
						"delete_method":        tftypes.NewValue(tftypes.String, nil),
						"delete_path":          tftypes.NewValue(tftypes.String, nil),
						"delete_headers":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"delete_request_body":  tftypes.NewValue(tftypes.String, nil),
						"delete_resolved_path": tftypes.NewValue(tftypes.String, nil),

						// Computed fields
						"id":                 tftypes.NewValue(tftypes.String, nil),
						"response_code":      tftypes.NewValue(tftypes.Number, nil),
						"response_body":      tftypes.NewValue(tftypes.String, nil),
						"response_body_id":   tftypes.NewValue(tftypes.String, nil),
						"response_body_json": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
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
		assert.Equal(t, 0, len(resp.Diagnostics), "there should be no errors when required parameters are set")
		assert.Equal(t, 0, len(resp.Diagnostics), "there should be no errors when required parameters are set")
	})
}

func TestHTTPRequestResource_DestroyValidation(t *testing.T) {
	t.Parallel()

	t.Run("should validate destroy configuration with custom delete_path and JSONPath token", func(t *testing.T) {
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
						WithQueryParameters().
						WithIsDeleteEnabled().
						WithDeleteMethod().
						WithDeletePath().
						WithDeleteHeaders().
						WithDeleteRequestBody().
						WithDeleteResolvedPath().
						WithID().
						WithResponseCode().
						WithResponseBody().
						WithResponseBodyID().
						WithResponseBodyJSON().
						Build(),
					map[string]tftypes.Value{
						"method":                  tftypes.NewValue(tftypes.String, "POST"),
						"path":                    tftypes.NewValue(tftypes.String, "/posts"),
						"headers":                 tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"request_body":            tftypes.NewValue(tftypes.String, `{"title":"test"}`),
						"is_response_body_json":   tftypes.NewValue(tftypes.Bool, true),
						"response_body_id_filter": tftypes.NewValue(tftypes.String, "$.id"),
						"query_parameters":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),

						// Destroy controls
						"is_delete_enabled":    tftypes.NewValue(tftypes.Bool, true),
						"delete_method":        tftypes.NewValue(tftypes.String, "DELETE"),
						"delete_path":          tftypes.NewValue(tftypes.String, "/posts/$.id"),
						"delete_headers":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"delete_request_body":  tftypes.NewValue(tftypes.String, nil),
						"delete_resolved_path": tftypes.NewValue(tftypes.String, nil),

						// Computed fields
						"id":                 tftypes.NewValue(tftypes.String, nil),
						"response_code":      tftypes.NewValue(tftypes.Number, nil),
						"response_body":      tftypes.NewValue(tftypes.String, nil),
						"response_body_id":   tftypes.NewValue(tftypes.String, nil),
						"response_body_json": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
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
		assert.Equal(t, 0, len(resp.Diagnostics), "there should be no errors when all parameters are correctly set for destroy")
	})

	t.Run("should validate destroy configuration with custom delete_method POST", func(t *testing.T) {
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
						WithQueryParameters().
						WithIsDeleteEnabled().
						WithDeleteMethod().
						WithDeletePath().
						WithDeleteHeaders().
						WithDeleteRequestBody().
						WithDeleteResolvedPath().
						WithID().
						WithResponseCode().
						WithResponseBody().
						WithResponseBodyID().
						WithResponseBodyJSON().
						Build(),
					map[string]tftypes.Value{
						"method":                  tftypes.NewValue(tftypes.String, "POST"),
						"path":                    tftypes.NewValue(tftypes.String, "/posts"),
						"headers":                 tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"request_body":            tftypes.NewValue(tftypes.String, `{"title":"test"}`),
						"is_response_body_json":   tftypes.NewValue(tftypes.Bool, true),
						"response_body_id_filter": tftypes.NewValue(tftypes.String, "$.id"),
						"query_parameters":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),

						// Destroy controls - soft delete example
						"is_delete_enabled":    tftypes.NewValue(tftypes.Bool, true),
						"delete_method":        tftypes.NewValue(tftypes.String, "POST"),
						"delete_path":          tftypes.NewValue(tftypes.String, "/posts/$.id/archive"),
						"delete_headers":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
						"delete_request_body":  tftypes.NewValue(tftypes.String, `{"reason":"terraform destroy"}`),
						"delete_resolved_path": tftypes.NewValue(tftypes.String, nil),

						// Computed fields
						"id":                 tftypes.NewValue(tftypes.String, nil),
						"response_code":      tftypes.NewValue(tftypes.Number, nil),
						"response_body":      tftypes.NewValue(tftypes.String, nil),
						"response_body_id":   tftypes.NewValue(tftypes.String, nil),
						"response_body_json": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
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
		assert.Equal(t, 0, len(resp.Diagnostics), "there should be no errors when soft delete configuration is valid")
	})
}
