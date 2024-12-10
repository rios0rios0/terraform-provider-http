//go:build unit || integration

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/rios0rios0/terraform-provider-http/test/infrastructure/builders"
	"github.com/stretchr/testify/assert"
	"os"
	"regexp"
	"testing"
)

var (
	/*
		This factory is barely used to create the block "terraform.required_providers" in the Terraform configuration
	*/
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"http": providerserver.NewProtocol6WithError(New("test")()),
	}
)

func testAccPreCheck(_ *testing.T) {
	err := os.Setenv("TF_ACC_PROVIDER_NAMESPACE", "rios0rios0")
	if err != nil {
		return
	}

	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func TestHTTPProvider(t *testing.T) {
	t.Parallel()

	t.Run("should return an error when the URL is empty", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: builders.NewProviderTFBuilder().WithURL("").Build() +
						builders.NewResourceTFBuilder().
							WithName("test1").
							WithMethod("GET").
							WithPath("/posts/1").
							Build(),
					// TODO: ExpectError: regexp.MustCompile("'url' value must not be empty."),
					ExpectError: regexp.MustCompile("Unknown URL for HTTP client"),
				},
			},
		})
	})

	t.Run("should return an error when the URL is missing", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: builders.NewProviderTFBuilder().Build() +
						builders.NewResourceTFBuilder().
							WithName("test1").
							WithMethod("GET").
							WithPath("/posts/1").
							Build(),
					ExpectError: regexp.MustCompile("The argument \"url\" is required, but no definition was found."),
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
						WithURL("https://jsonplaceholder.typicode.com").
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
						WithURL("https://jsonplaceholder.typicode.com").
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

func TestHTTPProvider_ValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("should not throw any error when the URL is set", func(t *testing.T) {
		// given
		req := provider.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewProviderTypeBuilder().
						WithURL().
						WithIgnoreTLS().
						WithUsername().
						WithPassword().
						Build(),
					map[string]tftypes.Value{
						"url": tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
						"basic_auth": tftypes.NewValue(
							tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"username": tftypes.String,
									"password": tftypes.String,
								},
							},
							nil,
						),
						"ignore_tls": tftypes.NewValue(tftypes.Bool, nil),
					},
				),
				Schema: GetHTTPProviderSchema(),
			},
		}
		resp := provider.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPProvider{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Equal(t, 0, len(resp.Diagnostics), "there's no error since the URL is set")
		assert.Equal(t, diag.Diagnostics{}, resp.Diagnostics, "Diagnostic is empty since the URL is set")
	})

	t.Run("should throw an error when the URL was not set", func(t *testing.T) {
		// given
		req := provider.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewProviderTypeBuilder().
						WithURL().
						WithIgnoreTLS().
						WithUsername().
						WithPassword().
						Build(),
					map[string]tftypes.Value{
						"url": tftypes.NewValue(tftypes.String, nil),
						"basic_auth": tftypes.NewValue(
							tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"username": tftypes.String,
									"password": tftypes.String,
								},
							},
							nil,
						),
						"ignore_tls": tftypes.NewValue(tftypes.Bool, nil),
					},
				),
				Schema: GetHTTPProviderSchema(),
			},
		}
		resp := provider.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPProvider{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Equal(t, 1, len(resp.Diagnostics), "there's an error since the URL is not set")
		assert.Equal(t, "Unknown URL for HTTP client", resp.Diagnostics[0].Summary(), "the error message is correct")
	})

	t.Run("should throw an error when the schema was not properly set", func(t *testing.T) {
		// given
		req := provider.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewProviderTypeBuilder().
						WithURL().
						Build(),
					map[string]tftypes.Value{
						"url": tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
					},
				),
				Schema: GetHTTPProviderSchema(),
			},
		}
		resp := provider.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPProvider{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Equal(t, 1, len(resp.Diagnostics), "there's an error since provider schema wasn't properly set")
		assert.Equal(t, "Value Conversion Error", resp.Diagnostics[0].Summary(), "the summary error message is correct")
		assert.Contains(t, resp.Diagnostics[0].Detail(), "defines fields not found in object", "the detail error message is correct")
		assert.Contains(t, resp.Diagnostics[0].Detail(), "basic_auth", "the detail error message contains the missing field")
		assert.Contains(t, resp.Diagnostics[0].Detail(), "ignore_tls", "the detail error message contains the missing field")
	})

	t.Run("should throw an error when the 'basic_auth' was set but 'username' was not set", func(t *testing.T) {
		// given
		req := provider.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewProviderTypeBuilder().
						WithURL().
						WithIgnoreTLS().
						WithUsername().
						WithPassword().
						Build(),
					map[string]tftypes.Value{
						"url": tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
						"basic_auth": tftypes.NewValue(
							tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"username": tftypes.String,
									"password": tftypes.String,
								},
							},
							map[string]tftypes.Value{
								"username": tftypes.NewValue(tftypes.String, nil),
								"password": tftypes.NewValue(tftypes.String, "pass"),
							},
						),
						"ignore_tls": tftypes.NewValue(tftypes.Bool, nil),
					},
				),
				Schema: GetHTTPProviderSchema(),
			},
		}
		resp := provider.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPProvider{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Equal(t, 1, len(resp.Diagnostics), "there's an error since the username is not set")
		assert.Equal(t, "Unknown username for HTTP client", resp.Diagnostics[0].Summary(), "the error message is correct")
	})

	t.Run("should throw an error when the 'basic_auth' was set but 'password' was not set", func(t *testing.T) {
		// given
		req := provider.ValidateConfigRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(
					builders.NewProviderTypeBuilder().
						WithURL().
						WithIgnoreTLS().
						WithUsername().
						WithPassword().
						Build(),
					map[string]tftypes.Value{
						"url": tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
						"basic_auth": tftypes.NewValue(
							tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"username": tftypes.String,
									"password": tftypes.String,
								},
							},
							map[string]tftypes.Value{
								"username": tftypes.NewValue(tftypes.String, "user"),
								"password": tftypes.NewValue(tftypes.String, nil),
							},
						),
						"ignore_tls": tftypes.NewValue(tftypes.Bool, nil),
					},
				),
				Schema: GetHTTPProviderSchema(),
			},
		}
		resp := provider.ValidateConfigResponse{
			Diagnostics: make(diag.Diagnostics, 0),
		}

		// when
		it := &HTTPProvider{}
		it.ValidateConfig(context.Background(), req, &resp)

		// then
		assert.Equal(t, 1, len(resp.Diagnostics), "there's an error since the password is not set")
		assert.Equal(t, "Unknown password for HTTP client", resp.Diagnostics[0].Summary(), "the error message is correct")
	})
}
