//go:build integration

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
)

func TestHTTPProvider(t *testing.T) {
	t.Parallel()

	t.Run("should work when the URL is missing at provider level but provided at resource level", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: liveProvider().Build() +
						builders.NewResourceTFBuilder().
							WithName("test1").
							WithMethod("GET").
							WithPath("/posts/1").
							WithBaseURL(liveEndpoint).
							Build(),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("http_request.test1", "response_code", "200"),
					),
				},
			},
		})
	})

	t.Run("should return an error when the 'username' is missing with 'basic_auth'", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: builders.NewProviderTFBuilder().
						WithURL(liveEndpoint).
						WithPassword("anything").
						Build() +
						builders.NewResourceTFBuilder().
							WithName("test1").
							WithMethod("GET").
							WithPath("/posts/1").
							Build(),
					ExpectError: regexp.MustCompile("Inappropriate value for attribute \"basic_auth\": attribute \"username\" is"),
				},
			},
		})
	})

	t.Run("should return an error when the 'password' is missing with 'basic_auth'", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: builders.NewProviderTFBuilder().
						WithURL(liveEndpoint).
						WithUsername("anything").
						Build() +
						builders.NewResourceTFBuilder().
							WithName("test1").
							WithMethod("GET").
							WithPath("/posts/1").
							Build(),
					ExpectError: regexp.MustCompile("Inappropriate value for attribute \"basic_auth\": attribute \"password\" is"),
				},
			},
		})
	})
}
