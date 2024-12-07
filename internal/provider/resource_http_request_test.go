//go:build integration

package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/helpers"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// According to Terraform SDK documentation delete testing automatically occurs in TestCase
// https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework/resource#TestCase

var providerConfigs = []string{
	helpers.GenerateProviderConfig("https://jsonplaceholder.typicode.com", true, true),
	helpers.GenerateProviderConfig("https://jsonplaceholder.typicode.com", false, true),
	helpers.GenerateProviderConfig("https://jsonplaceholder.typicode.com", true, false),
	helpers.GenerateProviderConfig("https://jsonplaceholder.typicode.com", false, false),
}

func TestHTTPRequestResource(t *testing.T) {
	t.Parallel()

	t.Run("should apply and check the state when using GET method", func(t *testing.T) {
		var state1 bytes.Buffer
		_ = json.Compact(&state1, []byte(`{
		"method": "GET",
		"path": "/posts/1",
		"response_code": 200,
		"response_body": "{\n  \"userId\": 1,\n  \"id\": 1,\n  \"title\": \"sunt aut facere repellat provident occaecati excepturi optio reprehenderit\",\n  \"body\": \"quia et suscipit\\nsuscipit recusandae consequuntur expedita et cum\\nreprehenderit molestiae ut ut quas totam\\nnostrum rerum est autem sunt rem eveniet architecto\"\n}"
	}`))
		state1ID := base64.StdEncoding.EncodeToString(state1.Bytes())

		for _, providerConfig := range providerConfigs {
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// create and read testing for GET method
					{
						Config: providerConfig +
							helpers.GenerateResourceConfig("test1", "GET", "/posts/1", "", "", false),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test1", "method", "GET"),
							resource.TestCheckResourceAttr("http_request.test1", "path", "/posts/1"),
							// Verify dynamic values have any value set in the state.
							resource.TestCheckResourceAttrSet("http_request.test1", "id"),
							resource.TestCheckResourceAttrSet("http_request.test1", "response_code"),
							resource.TestCheckResourceAttrSet("http_request.test1", "response_body"),
						),
					},

					// ImportState testing for GET method
					{
						ResourceName:            "http_request.test1",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateId:           state1ID,
						ImportStateVerifyIgnore: []string{"is_response_body_json"}, // bool empty are considered null via Terraform SDK
					},
				},
			})
		}
	})

	t.Run("should apply and check the state when using POST and a non-JSON request body", func(t *testing.T) {
		var state2 bytes.Buffer
		_ = json.Compact(&state2, []byte(`{
		"method": "POST",
		"path": "/posts",
		"request_body": "test body",
		"response_code": 201,
		"response_body":"{\n  \"id\": 101\n}"
	}`))
		state2ID := base64.StdEncoding.EncodeToString(state2.Bytes())

		for _, providerConfig := range providerConfigs {
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// create and read testing for POST method
					{
						Config: providerConfig +
							helpers.GenerateResourceConfig("test2", "POST", "/posts", "test body", "", false),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test2", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test2", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test2", "request_body", "test body"),
							// Verify dynamic values have any value set in the state.
							resource.TestCheckResourceAttr("http_request.test2", "id", state2ID),
							resource.TestCheckResourceAttr("http_request.test2", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test2", "response_body"),
						),
					},

					// ImportState testing for POST method
					{
						ResourceName:            "http_request.test2",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateId:           state2ID,
						ImportStateVerifyIgnore: []string{"is_response_body_json"}, // bool empty are considered null via Terraform SDK
					},

					// TODO: update and read testing should be implemented here
				},
			})
		}
	})

	t.Run("should apply and check the state when using POST and a JSON request body", func(t *testing.T) {
		var state3 bytes.Buffer
		_ = json.Compact(&state3, []byte(`{
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
		state3ID := base64.StdEncoding.EncodeToString(state3.Bytes())

		for _, providerConfig := range providerConfigs {
			resource.UnitTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					// create and read testing for POST method with response filtering
					{
						Config: providerConfig +
							helpers.GenerateResourceConfig("test3", "POST", "/posts", "{ \"test\": \"test body\" }", "$.id", true),

						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("http_request.test3", "method", "POST"),
							resource.TestCheckResourceAttr("http_request.test3", "path", "/posts"),
							resource.TestCheckResourceAttr("http_request.test3", "request_body", "{ \"test\": \"test body\" }"),
							resource.TestCheckResourceAttr("http_request.test3", "is_response_body_json", "true"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_id_filter", "$.id"),
							// Verify dynamic values have any value set in the state.
							resource.TestCheckResourceAttr("http_request.test3", "id", state3ID),
							resource.TestCheckResourceAttr("http_request.test3", "response_code", "201"),
							resource.TestCheckResourceAttrSet("http_request.test3", "response_body"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_id", "101"),
							resource.TestCheckResourceAttr("http_request.test3", "response_body_json.id", "101"),
						),
					},

					// ImportState testing for POST method with response filtering
					{
						ResourceName:            "http_request.test3",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"is_response_body_json"}, // bool empty are considered null via Terraform SDK
					},

					// TODO: update and read testing should be implemented here
				},
			})
		}
	})
}
