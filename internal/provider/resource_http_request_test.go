//go:build integration

package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
					WithDeletePath("/posts/$.id").
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
					// Use the created ID so the DELETE hits a 2xx endpoint
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
							resource.TestCheckResourceAttr("http_request.test_delete", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test_delete", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test_delete", "is_delete_enabled", "true"),
							resource.TestCheckResourceAttrSet("http_request.test_delete", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete", "response_body_id", "101"),
						),
					},
					// Destroy testing - this will attempt DELETE to /posts/{id}
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
					WithDeleteMethod("PATCH").
					WithDeletePath("/posts/$.id").
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
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_method", "PATCH"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_path", "/posts/$.id"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_headers.X-Delete-Reason", "terraform-destroy"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_headers.Content-Type", "application/json"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_request_body", `{"reason": "terraform destroy", "actor": "automation"}`),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_all", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_all", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "response_body_id", "101"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_resolved_path", "/posts/101")),
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

	// New tests for resource-level configuration feature
	t.Run("should work with resource-level base URL", func(t *testing.T) {
		config := builders.NewProviderTFBuilder().Build() + // No provider-level URL
			builders.NewResourceTFBuilder().
				WithName("test_resource_url").
				WithMethod("GET").
				WithPath("/posts/1").
				WithBaseURL("https://jsonplaceholder.typicode.com").
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.test_resource_url", "response_code", "200"),
						resource.TestCheckResourceAttrSet("http_request.test_resource_url", "response_body_id"),
						resource.TestCheckResourceAttr("http_request.test_resource_url", "base_url", "https://jsonplaceholder.typicode.com"),
					),
				},
			},
		})
	})

	t.Run("should work with resource-level basic auth", func(t *testing.T) {
		config := builders.NewProviderTFBuilder().Build() + // No provider-level auth
			builders.NewResourceTFBuilder().
				WithName("test_resource_auth").
				WithMethod("GET").
				WithPath("/posts/1").
				WithBaseURL("https://jsonplaceholder.typicode.com").
				WithBasicAuth("testuser", "testpass").
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.test_resource_auth", "response_code", "200"),
						resource.TestCheckResourceAttrSet("http_request.test_resource_auth", "response_body_id"),
						resource.TestCheckResourceAttr("http_request.test_resource_auth", "basic_auth.username", "testuser"),
						// Password should be sensitive and not directly checkable
					),
				},
			},
		})
	})

	t.Run("should work with resource-level ignore_tls", func(t *testing.T) {
		config := builders.NewProviderTFBuilder().Build() + // No provider-level TLS config
			builders.NewResourceTFBuilder().
				WithName("test_resource_tls").
				WithMethod("GET").
				WithPath("/posts/1").
				WithBaseURL("https://jsonplaceholder.typicode.com").
				WithIgnoreTLS(true).
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.test_resource_tls", "response_code", "200"),
						resource.TestCheckResourceAttrSet("http_request.test_resource_tls", "response_body_id"),
						resource.TestCheckResourceAttr("http_request.test_resource_tls", "ignore_tls", "true"),
					),
				},
			},
		})
	})

	t.Run("should work with mixed provider and resource level configurations", func(t *testing.T) {
		config := builders.NewProviderTFBuilder().
			WithURL("https://jsonplaceholder.typicode.com").
			WithIgnoreTLS(true).
			Build() +
			// Resource using provider-level configuration
			builders.NewResourceTFBuilder().
				WithName("test_provider_config").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build() +
			// Resource overriding with resource-level configuration
			builders.NewResourceTFBuilder().
				WithName("test_resource_config").
				WithMethod("GET").
				WithPath("/posts/2").
				WithBaseURL("https://jsonplaceholder.typicode.com"). // Override base URL
				WithIgnoreTLS(false).                               // Override TLS setting
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						// First resource should use provider config
						resource.TestCheckResourceAttr("http_request.test_provider_config", "response_code", "200"),
						resource.TestCheckResourceAttrSet("http_request.test_provider_config", "response_body_id"),
						resource.TestCheckNoResourceAttr("http_request.test_provider_config", "base_url"),
						
						// Second resource should use resource-level config
						resource.TestCheckResourceAttr("http_request.test_resource_config", "response_code", "200"),
						resource.TestCheckResourceAttrSet("http_request.test_resource_config", "response_body_id"),
						resource.TestCheckResourceAttr("http_request.test_resource_config", "base_url", "https://jsonplaceholder.typicode.com"),
						resource.TestCheckResourceAttr("http_request.test_resource_config", "ignore_tls", "false"),
					),
				},
			},
		})
	})

	t.Run("should return error when no base URL is configured", func(t *testing.T) {
		config := builders.NewProviderTFBuilder().Build() + // No provider-level URL
			builders.NewResourceTFBuilder().
				WithName("test_no_url").
				WithMethod("GET").
				WithPath("/posts/1").
				Build() // No resource-level base_url either

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("No base URL configured"),
				},
			},
		})
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

func TestHTTPRequestResource_JSONPathTokenResolution(t *testing.T) {
	t.Parallel()

	t.Run("should resolve single JSONPath token in delete_path", func(t *testing.T) {
		// given
		rawPath := "/posts/$.id"
		responseBody := `{"id": 123, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully resolve token")
		assert.Equal(t, "/posts/123", resolved, "should replace $.id with 123")
		assert.Equal(t, 0, len(diagnostics), "should have no errors")
	})

	t.Run("should resolve multiple JSONPath tokens in delete_path", func(t *testing.T) {
		// given
		rawPath := "/users/$.userId/posts/$.id"
		responseBody := `{"id": 456, "userId": 789, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully resolve tokens")
		assert.Equal(t, "/users/789/posts/456", resolved, "should replace both tokens")
		assert.Equal(t, 0, len(diagnostics), "should have no errors")
	})

	t.Run("should return original path when no JSONPath tokens present", func(t *testing.T) {
		// given
		rawPath := "/posts/123"
		responseBody := `{"id": 123, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully process")
		assert.Equal(t, "/posts/123", resolved, "should return original path unchanged")
		assert.Equal(t, 0, len(diagnostics), "should have no errors")
	})

	t.Run("should handle error when JSONPath token not found in response", func(t *testing.T) {
		// given
		rawPath := "/posts/$.nonexistent"
		responseBody := `{"id": 123, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.False(t, ok, "should fail to resolve")
		assert.Equal(t, "", resolved, "should return empty string on error")
		assert.Greater(t, len(diagnostics), 0, "should have error diagnostics")
		assert.Contains(t, diagnostics[0].Summary(), "JSONPath token not found", "should have appropriate error message")
	})

	t.Run("should handle error when response body is invalid JSON", func(t *testing.T) {
		// given
		rawPath := "/posts/$.id"
		responseBody := `invalid json`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.False(t, ok, "should fail to resolve")
		assert.Equal(t, "", resolved, "should return empty string on error")
		assert.Greater(t, len(diagnostics), 0, "should have error diagnostics")
		assert.Contains(t, diagnostics[0].Summary(), "unmarshall response body", "should have appropriate error message")
	})

	t.Run("should resolve nested JSONPath token", func(t *testing.T) {
		// given
		rawPath := "/posts/$.data.id"
		responseBody := `{"data": {"id": 999}, "title": "test post"}`
		var diagnostics diag.Diagnostics

		// when
		resolved, ok := resolveDeletePathTokens(rawPath, responseBody, &diagnostics)

		// then
		assert.True(t, ok, "should successfully resolve nested token")
		assert.Equal(t, "/posts/999", resolved, "should replace $.data.id with 999")
		assert.Equal(t, 0, len(diagnostics), "should have no errors")
	})
}

func TestHTTPRequestResource_DestroyHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("isBoolTrue should correctly identify true boolean values", func(t *testing.T) {
		// given
		trueValue := types.BoolValue(true)
		falseValue := types.BoolValue(false)
		nullValue := types.BoolNull()

		// then
		assert.True(t, isBoolTrue(trueValue), "should return true for true boolean")
		assert.False(t, isBoolTrue(falseValue), "should return false for false boolean")
		assert.False(t, isBoolTrue(nullValue), "should return false for null boolean")
	})

	t.Run("isNonEmptyString should correctly identify non-empty strings", func(t *testing.T) {
		// given
		nonEmptyValue := types.StringValue("test")
		emptyValue := types.StringValue("")
		whitespaceValue := types.StringValue("   ")
		nullValue := types.StringNull()

		// then
		assert.True(t, isNonEmptyString(nonEmptyValue), "should return true for non-empty string")
		assert.False(t, isNonEmptyString(emptyValue), "should return false for empty string")
		assert.False(t, isNonEmptyString(whitespaceValue), "should return false for whitespace-only string")
		assert.False(t, isNonEmptyString(nullValue), "should return false for null string")
	})

	t.Run("pickDeleteMethod should return correct HTTP method", func(t *testing.T) {
		// given
		model1 := HTTPRequestResourceModel{
			DeleteMethod: types.StringValue("POST"),
		}
		model2 := HTTPRequestResourceModel{
			DeleteMethod: types.StringValue("  put  "),
		}
		model3 := HTTPRequestResourceModel{
			DeleteMethod: types.StringNull(),
		}
		model4 := HTTPRequestResourceModel{
			DeleteMethod: types.StringValue(""),
		}

		// then
		assert.Equal(t, "POST", pickDeleteMethod(model1), "should return POST for POST")
		assert.Equal(t, "PUT", pickDeleteMethod(model2), "should return PUT and trim whitespace")
		assert.Equal(t, "DELETE", pickDeleteMethod(model3), "should return DELETE for null")
		assert.Equal(t, "DELETE", pickDeleteMethod(model4), "should return DELETE for empty string")
	})

	t.Run("resolveDeleteTargetPath should handle different path scenarios", func(t *testing.T) {
		// Test case 1: No delete_path specified, should use original path
		model1 := HTTPRequestResourceModel{
			Path:       types.StringValue("/posts"),
			DeletePath: types.StringNull(),
		}
		var diag1 diag.Diagnostics
		path1, ok1 := resolveDeleteTargetPath(model1, &diag1)
		assert.True(t, ok1, "should succeed with no delete_path")
		assert.Equal(t, "/posts", path1, "should return original path")
		assert.Equal(t, 0, len(diag1), "should have no errors")

		// Test case 2: delete_path without JSONPath tokens
		model2 := HTTPRequestResourceModel{
			Path:         types.StringValue("/posts"),
			DeletePath:   types.StringValue("/posts/123"),
			ResponseBody: types.StringValue(`{"id": 123}`), // Need some response body for the function
		}
		var diag2 diag.Diagnostics
		path2, ok2 := resolveDeleteTargetPath(model2, &diag2)
		assert.True(t, ok2, "should succeed with simple delete_path")
		assert.Equal(t, "/posts/123", path2, "should return delete_path")
		assert.Equal(t, 0, len(diag2), "should have no errors")

		// Test case 3: delete_path already resolved
		model3 := HTTPRequestResourceModel{
			Path:               types.StringValue("/posts"),
			DeletePath:         types.StringValue("/posts/$.id"),
			DeleteResolvedPath: types.StringValue("/posts/456"),
		}
		var diag3 diag.Diagnostics
		path3, ok3 := resolveDeleteTargetPath(model3, &diag3)
		assert.True(t, ok3, "should succeed with resolved path")
		assert.Equal(t, "/posts/456", path3, "should return resolved path")
		assert.Equal(t, 0, len(diag3), "should have no errors")
	})

	t.Run("makeDeleteModel should create correct delete model", func(t *testing.T) {
		// given
		baseHeaders, _ := types.MapValue(types.StringType, map[string]attr.Value{
			"Content-Type": types.StringValue("application/json"),
		})
		deleteHeaders, _ := types.MapValue(types.StringType, map[string]attr.Value{
			"X-Delete-Reason": types.StringValue("terraform-destroy"),
		})

		baseModel := HTTPRequestResourceModel{
			Method:            types.StringValue("POST"),
			Path:              types.StringValue("/posts"),
			Headers:           baseHeaders,
			RequestBody:       types.StringValue(`{"title":"test"}`),
			DeleteHeaders:     deleteHeaders,
			DeleteRequestBody: types.StringValue(`{"reason":"destroy"}`),
		}

		// when
		deleteModel := makeDeleteModel(baseModel, "DELETE", "/posts/123")

		// then
		assert.Equal(t, "DELETE", deleteModel.Method.ValueString(), "should set delete method")
		assert.Equal(t, "/posts/123", deleteModel.Path.ValueString(), "should set target path")
		assert.Equal(t, `{"reason":"destroy"}`, deleteModel.RequestBody.ValueString(), "should use delete request body")
		assert.Equal(t, "terraform-destroy", deleteModel.Headers.Elements()["X-Delete-Reason"].(types.String).ValueString(), "should use delete headers")
	})
}
