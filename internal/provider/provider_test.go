package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"http": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func TestAccHTTPRequestResource(t *testing.T) {
	resourceName := "http_request.example"
	baseURL := "https://jsonplaceholder.typicode.com"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRequestResourceConfig(baseURL),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHTTPRequestResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "path", "/posts/1"),
					resource.TestCheckResourceAttr(resourceName, "method", "GET"),
					resource.TestCheckResourceAttrSet(resourceName, "response_body"),
					resource.TestCheckResourceAttrSet(resourceName, "response_code"),
				),
			},
		},
	})
}

func testAccHTTPRequestResourceConfig(baseURL string) string {
	return `
provider "http" {
  url = "` + baseURL + `"
}

resource "http_request" "example" {
  path    = "/posts/1"
  method  = "GET"
  headers = {
    "Content-Type" = "application/json"
  }
}
`
}

func testAccCheckHTTPRequestResourceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		return nil
	}
}
