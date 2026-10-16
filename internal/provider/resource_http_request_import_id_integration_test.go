//go:build integration

package provider

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/rios0rios0/terraform-provider-http/v3/test/infrastructure/builders"
)

// checkImportID decodes the stored `import_id` and hands the arguments it carries to a check.
func checkImportID(address string, check func(*HTTPRequestResourceModelNative) error) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(address, attrImportID, func(value string) error {
		var diagnostics diag.Diagnostics

		payload := decodeImportID(value, &diagnostics)
		if payload == nil {
			return fmt.Errorf("import_id does not decode: %v", diagnostics)
		}

		return check(payload.native)
	})
}

func TestHTTPRequestResourceImportIDFollowsArguments(t *testing.T) {
	t.Run("should re-render import_id when an argument changes without re-sending the request", func(t *testing.T) {
		// given
		srv, remote := newKeyServer(t)
		address := "http_request.key"
		plain := keyResourceConfig(srv, func(*builders.ResourceTFBuilder) {})
		sensitive := keyResourceConfig(srv, func(b *builders.ResourceTFBuilder) {
			b.WithIsResponseBodySensitive(true)
		})
		tolerant := keyResourceConfig(srv, func(b *builders.ResourceTFBuilder) {
			b.WithIsResponseBodySensitive(true).WithToleratedStatusCodes([]int{409})
		})
		importIDReRendered := resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
				plancheck.ExpectUnknownValue(address, tfjsonpath.New(attrImportID)),
			},
		}

		// when
		resource.UnitTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{Config: plain},
				{
					Config:           sensitive,
					ConfigPlanChecks: importIDReRendered,
					// then
					Check: checkImportID(address, func(native *HTTPRequestResourceModelNative) error {
						if native.IsResponseBodySensitive == nil || !*native.IsResponseBodySensitive {
							return errors.New("import_id should carry is_response_body_sensitive = true")
						}

						return nil
					}),
				},
				{
					Config:           tolerant,
					ConfigPlanChecks: importIDReRendered,
					Check: checkImportID(address, func(native *HTTPRequestResourceModelNative) error {
						if !slices.Equal(native.ToleratedStatusCodes, []int32{409}) {
							return fmt.Errorf("import_id should carry tolerated_status_codes [409], got %v",
								native.ToleratedStatusCodes)
						}

						return nil
					}),
				},
			},
		})

		// then
		if calls := remote.calls("POST /keys"); calls != 1 {
			t.Fatalf("the key should have been minted exactly once, got %d", calls)
		}
	})
}
