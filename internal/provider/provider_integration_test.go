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
		// given: no provider-level URL, so the resource's base_url is the only one there is
		config := liveProvider().Build() +
			builders.NewResourceTFBuilder().
				WithName("test1").
				WithMethod("GET").
				WithPath("/posts/1").
				WithBaseURL(liveEndpoint).
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
						resource.TestCheckResourceAttr("http_request.test1", "response_code", "200"),
					),
				},
			},
		})
	})

	t.Run("should return an error when the 'username' is missing with 'basic_auth'", func(t *testing.T) {
		// given: a basic_auth block that names only the username
		config := builders.NewProviderTFBuilder().
			WithURL(liveEndpoint).
			WithPassword("anything").
			Build() +
			builders.NewResourceTFBuilder().
				WithName("test1").
				WithMethod("GET").
				WithPath("/posts/1").
				Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					// then: the configuration is rejected before any request is made
					ExpectError: regexp.MustCompile("Inappropriate value for attribute \"basic_auth\": attribute \"username\" is"),
				},
			},
		})
	})

	t.Run("should return an error when the 'password' is missing with 'basic_auth'", func(t *testing.T) {
		// given: a basic_auth block that names only the password
		config := builders.NewProviderTFBuilder().
			WithURL(liveEndpoint).
			WithUsername("anything").
			Build() +
			builders.NewResourceTFBuilder().
				WithName("test1").
				WithMethod("GET").
				WithPath("/posts/1").
				Build()

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					// then: the configuration is rejected before any request is made
					ExpectError: regexp.MustCompile("Inappropriate value for attribute \"basic_auth\": attribute \"password\" is"),
				},
			},
		})
	})
}
