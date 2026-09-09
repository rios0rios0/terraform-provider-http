//go:build integration

package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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

		for _, providerConfig := range liveProviderConfigs() {
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

		for _, providerConfig := range liveProviderConfigs() {
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

		for _, providerConfig := range liveProviderConfigs() {
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

		for _, providerConfig := range liveProviderConfigs() {
			// when
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
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
		for _, providerConfig := range liveProviderConfigs() {
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
							// Write-only attrs must NOT appear in state
							resource.TestCheckNoResourceAttr("http_request.test_delete", "is_delete_enabled"),
							resource.TestCheckNoResourceAttr("http_request.test_delete", "delete_path"),
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
		for _, providerConfig := range liveProviderConfigs() {
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
							// Write-only attrs must NOT appear in state
							resource.TestCheckNoResourceAttr("http_request.test_delete_custom_path", "is_delete_enabled"),
							resource.TestCheckNoResourceAttr("http_request.test_delete_custom_path", "delete_path"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_path", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_path", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_path", "response_body_id", "101"),
							// delete_resolved_path IS computed (not write-only) and must remain in state
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
		for _, providerConfig := range liveProviderConfigs() {
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
							// Write-only attrs must NOT appear in state
							resource.TestCheckNoResourceAttr("http_request.test_delete_custom_all", "is_delete_enabled"),
							resource.TestCheckNoResourceAttr("http_request.test_delete_custom_all", "delete_method"),
							resource.TestCheckNoResourceAttr("http_request.test_delete_custom_all", "delete_path"),
							resource.TestCheckNoResourceAttr("http_request.test_delete_custom_all", "delete_headers.X-Delete-Reason"),
							resource.TestCheckNoResourceAttr("http_request.test_delete_custom_all", "delete_request_body"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_all", "id"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test_delete_custom_all", "response_body"),
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "response_body_id", "101"),
							// delete_resolved_path IS computed (not write-only) and must remain in state
							resource.TestCheckResourceAttr("http_request.test_delete_custom_all", "delete_resolved_path", "/posts/101"),
						),
					},
					// Destroy testing - this will attempt PATCH to /posts/101 with custom headers and body
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
		for _, providerConfig := range liveProviderConfigs() {
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
							// Write-only attrs must NOT appear in state
							resource.TestCheckNoResourceAttr("http_request.test_delete_disabled", "is_delete_enabled"),
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
		for _, providerConfig := range liveProviderConfigs() {
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
							// Write-only attrs must NOT appear in state
							resource.TestCheckNoResourceAttr("http_request.test_delete_get", "is_delete_enabled"),
							resource.TestCheckNoResourceAttr("http_request.test_delete_get", "delete_path"),
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
		config := liveProvider().Build() + // No provider-level URL
			builders.NewResourceTFBuilder().
				WithName("test_resource_url").
				WithMethod("GET").
				WithPath("/posts/1").
				WithBaseURL(liveEndpoint).
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                  func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories:  testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.test_resource_url", "response_code", "200"),
						resource.TestCheckResourceAttrSet("http_request.test_resource_url", "response_body_id"),
						resource.TestCheckResourceAttr("http_request.test_resource_url", "base_url", liveEndpoint),
					),
				},
			},
		})
	})

	t.Run("should work with resource-level basic auth", func(t *testing.T) {
		config := liveProvider().Build() + // No provider-level auth
			builders.NewResourceTFBuilder().
				WithName("test_resource_auth").
				WithMethod("GET").
				WithPath("/posts/1").
				WithBaseURL(liveEndpoint).
				WithBasicAuth("testuser", "testpass").
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                  func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories:  testAccProtoV6ProviderFactories,
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
		config := liveProvider().Build() + // No provider-level TLS config
			builders.NewResourceTFBuilder().
				WithName("test_resource_tls").
				WithMethod("GET").
				WithPath("/posts/1").
				WithBaseURL(liveEndpoint).
				WithIgnoreTLS(true).
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                  func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories:  testAccProtoV6ProviderFactories,
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
		config := liveProviderWithURL().
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
				WithBaseURL(liveEndpoint). // Override base URL
				WithIgnoreTLS(false).      // Override TLS setting
				WithIsResponseBodyJSON(true).
				WithResponseBodyIDFilter("$.id").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                  func() { testAccPreCheck(t) },
			PreventPostDestroyRefresh: true,
			ProtoV6ProviderFactories:  testAccProtoV6ProviderFactories,
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
						resource.TestCheckResourceAttr("http_request.test_resource_config", "base_url", liveEndpoint),
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

	t.Run("should NOT produce plan diff when delete_method changes", func(t *testing.T) {
		providerConfig := liveProviderWithURL().Build()

		configStep1 := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_method_no_replace").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(false).
				WithDeleteMethod("DELETE").
				WithDeletePath("/posts/1").
				Build()

		configStep2 := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_method_no_replace").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(false).
				WithDeleteMethod("POST").
				WithDeletePath("/posts/1").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: configStep1,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckNoResourceAttr("http_request.test_delete_method_no_replace", "delete_method"),
					),
				},
				{
					Config:   configStep2,
					PlanOnly: true,
				},
			},
		})
	})

	t.Run("should NOT produce plan diff when delete_headers changes", func(t *testing.T) {
		providerConfig := liveProviderWithURL().Build()

		configStep1 := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_headers_no_replace").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(true).
				WithDeletePath("/posts/1").
				WithDeleteHeaders(map[string]string{
					"X-Delete-Reason": "initial",
				}).
				Build()

		configStep2 := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_headers_no_replace").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(true).
				WithDeletePath("/posts/1").
				WithDeleteHeaders(map[string]string{
					"X-Delete-Reason": "changed",
					"X-Extra-Header":  "new",
				}).
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: configStep1,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckNoResourceAttr("http_request.test_delete_headers_no_replace", "delete_headers.X-Delete-Reason"),
					),
				},
				{
					Config:   configStep2,
					PlanOnly: true,
				},
			},
		})
	})

	t.Run("should NOT produce plan diff when delete_request_body changes", func(t *testing.T) {
		providerConfig := liveProviderWithURL().Build()

		configStep1 := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_body_no_replace").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(true).
				WithDeletePath("/posts/1").
				WithDeleteRequestBody(strconv.Quote(`{"reason": "initial"}`)).
				Build()

		configStep2 := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_body_no_replace").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(true).
				WithDeletePath("/posts/1").
				WithDeleteRequestBody(strconv.Quote(`{"reason": "changed", "extra": "field"}`)).
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: configStep1,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckNoResourceAttr("http_request.test_delete_body_no_replace", "delete_request_body"),
					),
				},
				{
					Config:   configStep2,
					PlanOnly: true,
				},
			},
		})
	})

	t.Run("changing is_delete_enabled should NOT produce any plan diff", func(t *testing.T) {
		providerConfig := liveProviderWithURL().Build()

		configDeleteDisabled := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_enabled_no_diff").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(false).
				Build()

		configDeleteEnabled := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_delete_enabled_no_diff").
				WithMethod("GET").
				WithPath("/posts/1").
				WithIsDeleteEnabled(true).
				WithDeletePath("/posts/1").
				Build()

		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: configDeleteDisabled,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.test_delete_enabled_no_diff", "response_code", "200"),
						resource.TestCheckNoResourceAttr("http_request.test_delete_enabled_no_diff", "is_delete_enabled"),
						resource.TestCheckNoResourceAttr("http_request.test_delete_enabled_no_diff", "delete_path"),
					),
				},
				{
					Config:   configDeleteEnabled,
					PlanOnly: true,
				},
			},
		})
	})

	t.Run("should apply without error when server returns 404 and 404 is tolerated", func(t *testing.T) {
		// given
		providerConfig := liveProviderWithURL().Build()

		config := providerConfig +
			builders.NewResourceTFBuilder().
				WithName("test_tolerated_404").
				WithMethod("GET").
				WithPath("/posts/0"). // this endpoint returns 404
				WithToleratedStatusCodes([]int{404}).
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
						resource.TestCheckResourceAttr("http_request.test_tolerated_404", "method", "GET"),
						resource.TestCheckResourceAttr("http_request.test_tolerated_404", "path", "/posts/0"),
						resource.TestCheckResourceAttr("http_request.test_tolerated_404", "response_code", "404"),
						resource.TestCheckTypeSetElemAttr("http_request.test_tolerated_404", "tolerated_status_codes.*", "404"),
						resource.TestCheckResourceAttrSet("http_request.test_tolerated_404", "id"),
						resource.TestCheckResourceAttrSet("http_request.test_tolerated_404", "response_body"),
					),
				},
			},
		})
	})
}
