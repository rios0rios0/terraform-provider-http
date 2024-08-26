package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHTTPRequestResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "http_request" "test" {
  method = "GET"
  path   = "/test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify method
					resource.TestCheckResourceAttr("http_request.test", "method", "GET"),
					// Verify path
					resource.TestCheckResourceAttr("http_request.test", "path", "/test"),
					// Verify dynamic values have any value set in the state.
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
				ImportStateVerifyIgnore: []string{"response_body"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "http_request" "test" {
  method = "POST"
  path   = "/test"
  request_body = "test body"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify method updated
					resource.TestCheckResourceAttr("http_request.test", "method", "POST"),
					// Verify path
					resource.TestCheckResourceAttr("http_request.test", "path", "/test"),
					// Verify request body
					resource.TestCheckResourceAttr("http_request.test", "request_body", "test body"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("http_request.test", "id"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_code"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_body"),
				),
			},
			{
				Config: providerConfig + `
resource "http_request" "test" {
  method = "POST"
  path   = "/test"
  request_body = "{ "test": "test body" }"
  is_json = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify method updated
					resource.TestCheckResourceAttr("http_request.test", "method", "POST"),
					// Verify path
					resource.TestCheckResourceAttr("http_request.test", "path", "/test"),
					// Verify request body
					resource.TestCheckResourceAttr("http_request.test", "request_body", "test body"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("http_request.test", "id"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_code"),
					resource.TestCheckResourceAttrSet("http_request.test", "response_body"),
					// Verify response body JSON accessing the JSON value
					resource.TestCheckResourceAttr("http_request.test", "response_body_json", "{ \"test\": \"test body\" }"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
