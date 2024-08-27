package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHTTPRequestResource(t *testing.T) {
	t.Parallel()

	var stateTest1 bytes.Buffer
	_ = json.Compact(&stateTest1, []byte(`{
		"method": "GET",
		"path": "/posts/1",
		"response_code": 200,
		"response_body": "{\n  \"userId\": 1,\n  \"id\": 1,\n  \"title\": \"sunt aut facere repellat provident occaecati excepturi optio reprehenderit\",\n  \"body\": \"quia et suscipit\\nsuscipit recusandae consequuntur expedita et cum\\nreprehenderit molestiae ut ut quas totam\\nnostrum rerum est autem sunt rem eveniet architecto\"\n}"
	}`))

	resource.Test(t, resource.TestCase{
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
				ResourceName:      "http_request.test1",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     base64.StdEncoding.EncodeToString(stateTest1.Bytes()),
			},

			// TODO: update and read testing for GET method (it's not completed)
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
		},
	})

	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttr("http_request.test2", "id", "POST,/posts"),
					resource.TestCheckResourceAttr("http_request.test2", "response_code", "201"),
					resource.TestCheckResourceAttrSet("http_request.test2", "response_body"),
				),
			},

			// ImportState testing for POST method
			{
				ResourceName:      "http_request.test2",
				ImportState:       true,
				ImportStateVerify: true,
				// attributes that don't exist in the HTTP request. There's no value for them during import.
				ImportStateVerifyIgnore: []string{
					"request_body",
					"response_code",
					"response_body",
				},
			},

			// update and read testing for POST method
			{
				Config: providerConfig + `
resource "http_request" "test2" {
  method = "POST"
  path   = "/posts"
  request_body = "test body 2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("http_request.test2", "method", "POST"),
					resource.TestCheckResourceAttr("http_request.test2", "path", "/posts"),
					resource.TestCheckResourceAttr("http_request.test2", "request_body", "test body 2"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttr("http_request.test2", "id", "POST,/posts"),
					resource.TestCheckResourceAttr("http_request.test2", "response_code", "201"),
					resource.TestCheckResourceAttrSet("http_request.test2", "response_body"),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttr("http_request.test3", "id", "101"),
					resource.TestCheckResourceAttr("http_request.test3", "response_code", "201"),
					resource.TestCheckResourceAttrSet("http_request.test3", "response_body"),
					resource.TestCheckResourceAttrSet("http_request.test3", "response_body_json.id"),
				),
			},

			// ImportState testing for POST method with response filtering
			{
				ResourceName:      "http_request.test3",
				ImportState:       true,
				ImportStateVerify: true,
				// ImportStateId is set to 101 in the state file because of external data source.
				ImportStateId: "101",
				// attributes that don't exist in the HTTP request. There's no value for them during import.
				ImportStateVerifyIgnore: []string{
					"is_response_body_json",
					"request_body",
					"response_body",
					"response_body_id_filter",
					"response_code",
				},
			},

			// update and read testing for POST method with response filtering
			{
				Config: providerConfig + `
resource "http_request" "test3" {
  method = "POST"
  path   = "/posts"
  request_body = "{ \"test\": \"test body 2\" }"
  is_response_body_json = true
  response_body_id_filter = "$.id"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("http_request.test3", "method", "POST"),
					resource.TestCheckResourceAttr("http_request.test3", "path", "/posts"),
					resource.TestCheckResourceAttr("http_request.test3", "request_body", "{ \"test\": \"test body 2\" }"),
					resource.TestCheckResourceAttr("http_request.test3", "is_response_body_json", "true"),
					resource.TestCheckResourceAttr("http_request.test3", "response_body_id_filter", "$.id"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttr("http_request.test3", "id", "101"),
					resource.TestCheckResourceAttr("http_request.test3", "response_code", "201"),
					resource.TestCheckResourceAttrSet("http_request.test3", "response_body"),
					resource.TestCheckResourceAttrSet("http_request.test3", "response_body_json.id"),
				),
			},
		},
	})

	// delete testing automatically occurs in TestCase
}
