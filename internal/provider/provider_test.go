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

// fullProviderType returns the complete provider object type used by the
// ValidateConfig tests, keeping the (otherwise duplicated) builder chain in one place.
func fullProviderType() tftypes.Object {
	return builders.NewProviderTypeBuilder().
		WithURL().
		WithHeaders().
		WithIgnoreTLS().
		WithUsername().
		WithPassword().
		WithRequestTimeoutMs().
		WithRetry().
		Build()
}

// nullHeadersValue returns a null `headers` map value matching the schema. The provider object type
// names every attribute, so every one of them needs a typed value even when it is null.
func nullHeadersValue() tftypes.Value {
	return tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil)
}

// nullRetryValue returns a null `retry` nested-block value matching the schema,
// used to satisfy the full provider object type in ValidateConfig tests.
func nullRetryValue() tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"attempts":     tftypes.Number,
			"min_delay_ms": tftypes.Number,
			"max_delay_ms": tftypes.Number,
		}},
		nil,
	)
}

// basicAuthObjectType is the tftypes shape of the `basic_auth` attribute.
func basicAuthObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"username": tftypes.String,
			"password": tftypes.String,
		},
	}
}

// basicAuthValue builds a `basic_auth` value; a nil username or password is spelled as a null
// string, which is the case ValidateConfig exists to reject.
func basicAuthValue(username, password *string) tftypes.Value {
	stringOrNull := func(value *string) tftypes.Value {
		if value == nil {
			return tftypes.NewValue(tftypes.String, nil)
		}

		return tftypes.NewValue(tftypes.String, *value)
	}

	return tftypes.NewValue(basicAuthObjectType(), map[string]tftypes.Value{
		"username": stringOrNull(username),
		"password": stringOrNull(password),
	})
}

// nullBasicAuthValue returns an unset `basic_auth`.
func nullBasicAuthValue() tftypes.Value {
	return tftypes.NewValue(basicAuthObjectType(), nil)
}

// fullProviderValues builds the complete provider object value, with only the two parts that differ
// between the ValidateConfig cases supplied by the caller. `fullProviderType()` names every
// attribute, so every one needs a typed value even when null -- and spelling all six out per case is
// what made these blocks near-identical.
func fullProviderValues(url, basicAuth tftypes.Value) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"url":                url,
		"basic_auth":         basicAuth,
		"headers":            nullHeadersValue(),
		"ignore_tls":         tftypes.NewValue(tftypes.Bool, nil),
		"request_timeout_ms": tftypes.NewValue(tftypes.Number, nil),
		"retry":              nullRetryValue(),
	}
}

// validateConfigOf runs ValidateConfig over a raw provider value and returns the diagnostics.
func validateConfigOf(raw tftypes.Value) diag.Diagnostics {
	req := provider.ValidateConfigRequest{
		Config: tfsdk.Config{Raw: raw, Schema: GetHTTPProviderSchema()},
	}
	resp := provider.ValidateConfigResponse{Diagnostics: make(diag.Diagnostics, 0)}

	it := &HTTPProvider{}
	it.ValidateConfig(context.Background(), req, &resp)

	return resp.Diagnostics
}

func TestHTTPProvider(t *testing.T) {
	t.Parallel()

	t.Run("should work when the URL is missing at provider level but provided at resource level", func(t *testing.T) {
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
							WithBaseURL("https://jsonplaceholder.typicode.com").
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
		raw := tftypes.NewValue(fullProviderType(), fullProviderValues(
			tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
			nullBasicAuthValue(),
		))

		// when
		diagnostics := validateConfigOf(raw)

		// then
		assert.Equal(t, 0, len(diagnostics), "there's no error since the URL is set")
		assert.Equal(t, diag.Diagnostics{}, diagnostics, "Diagnostic is empty since the URL is set")
	})

	t.Run("should not throw an error when the URL was not set since it can be provided at resource level", func(t *testing.T) {
		// given
		raw := tftypes.NewValue(fullProviderType(), fullProviderValues(
			tftypes.NewValue(tftypes.String, nil),
			nullBasicAuthValue(),
		))

		// when
		diagnostics := validateConfigOf(raw)

		// then
		assert.Equal(t, 0, len(diagnostics), "there's no error since URL can be provided at resource level")
	})

	t.Run("should throw an error when the schema was not properly set", func(t *testing.T) {
		// given: a type naming only `url`, against the full provider schema
		raw := tftypes.NewValue(
			builders.NewProviderTypeBuilder().WithURL().Build(),
			map[string]tftypes.Value{
				"url": tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
			},
		)

		// when
		diagnostics := validateConfigOf(raw)

		// then
		assert.Equal(t, 1, len(diagnostics), "there's an error since provider schema wasn't properly set")
		assert.Equal(t, "Value Conversion Error", diagnostics[0].Summary(), "the summary error message is correct")
		assert.Contains(t, diagnostics[0].Detail(), "defines fields not found in object", "the detail error message is correct")
		assert.Contains(t, diagnostics[0].Detail(), "basic_auth", "the detail error message contains the missing field")
		assert.Contains(t, diagnostics[0].Detail(), "ignore_tls", "the detail error message contains the missing field")
	})

	t.Run("should throw an error when the 'basic_auth' was set but 'username' was not set", func(t *testing.T) {
		// given
		password := "pass"
		raw := tftypes.NewValue(fullProviderType(), fullProviderValues(
			tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
			basicAuthValue(nil, &password),
		))

		// when
		diagnostics := validateConfigOf(raw)

		// then
		assert.Equal(t, 1, len(diagnostics), "there's an error since the username is not set")
		assert.Equal(t, "Unknown username for HTTP client", diagnostics[0].Summary(), "the error message is correct")
	})

	t.Run("should throw an error when the 'basic_auth' was set but 'password' was not set", func(t *testing.T) {
		// given
		username := "user"
		raw := tftypes.NewValue(fullProviderType(), fullProviderValues(
			tftypes.NewValue(tftypes.String, "https://jsonplaceholder.typicode.com"),
			basicAuthValue(&username, nil),
		))

		// when
		diagnostics := validateConfigOf(raw)

		// then
		assert.Equal(t, 1, len(diagnostics), "there's an error since the password is not set")
		assert.Equal(t, "Unknown password for HTTP client", diagnostics[0].Summary(), "the error message is correct")
	})
}
