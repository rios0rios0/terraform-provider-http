package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHTTPRequestResource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read testing
			{
				Config: providerConfig + `
resource "http_request" "test" {
  method = "GET"
  path   = "/posts/1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("http_request.test", "method", "GET"),
					resource.TestCheckResourceAttr("http_request.test", "path", "/posts/1"),
					resource.TestCheckResourceAttrSet("http_request.test", "id"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_code"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_body"),
				),
			},

			// ImportState testing
			{
				ResourceName:      "http_request.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The response_body attribute does not exist in the HTTP request
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore: []string{"response_body", "response_body_json"},
			},

			// update and read testing
			{
				Config: providerConfig + `
resource "http_request" "test" {
  method = "POST"
  path   = "/posts/1"
  request_body = "test body"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("http_request.test", "method", "POST"),
					resource.TestCheckResourceAttr("http_request.test", "path", "/posts/1"),
					resource.TestCheckResourceAttr("http_request.test", "request_body", "test body"),
					resource.TestCheckResourceAttrSet("http_request.test", "id"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_code"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_body"),
				),
			},

			// id filtering from the JSON response body testing
			{
				Config: providerConfig + `
resource "http_request" "test" {
  method = "POST"
  path   = "/posts/1"
  request_body = "{ "test": "test body" }"
  is_response_json = true
  response_id_filter = "$.test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("http_request.test", "method", "POST"),
					resource.TestCheckResourceAttr("http_request.test", "path", "/posts/1"),
					resource.TestCheckResourceAttr("http_request.test", "request_body", "{ \"test\": \"test body\" }"),
					resource.TestCheckResourceAttrSet("http_request.test", "id"),
					resource.TestCheckResourceAttr("http_request.test", "id", "test body"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_code"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_body"),
					resource.TestCheckResourceAttr("http_request.test", "response_body_json", "{ \"test\": \"test body\" }"),
				),
			},

			// Delete testing automatically occurs in TestCase
		},
	})
}
