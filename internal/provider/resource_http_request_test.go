//go:build integration

package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// According to Terraform SDK documentation delete testing automatically occurs in TestCase
// https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework/resource#TestCase

func TestHTTPRequestResource_Create_getting(t *testing.T) {
	var state1 bytes.Buffer
	_ = json.Compact(&state1, []byte(`{
		"method": "GET",
		"path": "/posts/1",
		"response_code": 200,
		"response_body": "{\n  \"userId\": 1,\n  \"id\": 1,\n  \"title\": \"sunt aut facere repellat provident occaecati excepturi optio reprehenderit\",\n  \"body\": \"quia et suscipit\\nsuscipit recusandae consequuntur expedita et cum\\nreprehenderit molestiae ut ut quas totam\\nnostrum rerum est autem sunt rem eveniet architecto\"\n}"
	}`))
	state1ID := base64.StdEncoding.EncodeToString(state1.Bytes())
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read testing for GET method
			{
				Config: providerConfig + `
				resource "http_request" "test1" {
				  method = "GET"
				  path   = "/posts/1"
				}
				`,
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

			// TODO: update and read testing should be implemented here
		},
	})
}

func TestHTTPRequestResource_Create_posting(t *testing.T) {
	var state2 bytes.Buffer
	_ = json.Compact(&state2, []byte(`{
		"method": "POST",
		"path": "/posts",
		"request_body": "test body",
		"response_code": 201,
		"response_body":"{\n  \"id\": 101\n}"
	}`))
	state2ID := base64.StdEncoding.EncodeToString(state2.Bytes())
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// create and read testing for POST method
			{
				Config: providerConfig + `
				resource "http_request" "test2" {
				  method = "POST"
				  path   = "/posts"
				  request_body = "test body"
				}
				`,
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

func TestHTTPRequestResource_Filter_writing(t *testing.T) {
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
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read testing for POST method with response filtering
			{
				Config: providerConfig + `
				resource "http_request" "test3" {
				  method = "POST"
				  path   = "/posts"
				  request_body = "{ \"test\": \"test body\" }"
				  is_response_body_json = true
				  response_body_id_filter = "$.id"
				}
				`,
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
