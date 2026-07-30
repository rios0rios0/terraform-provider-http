package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/rios0rios0/terraform-provider-http/internal/infrastructure/helpers"
)

// importAdoptPrivateKey is the private-state key holding the pending adoption, written by
// ImportState and cleared by the first Update that acts on it.
const importAdoptPrivateKey = "import_adopt"

// importAdoptPrivate records which attributes an import identifier left unspecified.
//
// Terraform never shows a resource's configuration to ImportState, so an identifier that does not
// spell out every argument necessarily produces a state that differs from the configuration. Left
// alone, that difference lands on RequiresReplace attributes and the first plan after the import
// destroys and recreates the remote resource -- exactly what import exists to avoid. Recording the
// gap lets the plan modifiers below suppress replacement for those attributes only, once, so the
// first apply adopts the configuration in place instead.
type importAdoptPrivate struct {
	Attributes []string `json:"attributes"`
}

// privateStateReader reads provider private state. It is declared structurally so the framework's
// internal private-state type satisfies it without being imported.
type privateStateReader interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
}

// privateStateWriter writes provider private state.
type privateStateWriter interface {
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// adoptableAttributes returns the attributes whose replacement adoption is allowed to suppress.
//
// These are exactly the arguments carrying the conditional RequiresReplace modifier. `basic_auth`
// is deliberately absent: it never forced replacement, so there is nothing to suppress, and the
// computed and write-only attributes are absent because a configuration cannot set them.
func adoptableAttributes() map[string]struct{} {
	return map[string]struct{}{
		attrMethod:               {},
		attrPath:                 {},
		attrHeaders:              {},
		attrRequestBody:          {},
		attrQueryParameters:      {},
		attrBaseURL:              {},
		attrIgnoreTLS:            {},
		attrIsResponseBodyJSON:   {},
		attrResponseBodyIDFilter: {},
	}
}

// pendingAdoptionFor returns the adoptable attributes an import payload did not mention, sorted so
// the private state and the resulting warning are stable across runs.
func pendingAdoptionFor(specified map[string]struct{}) []string {
	pending := make([]string, 0, len(adoptableAttributes()))

	for attribute := range adoptableAttributes() {
		if _, ok := specified[attribute]; ok {
			continue
		}

		pending = append(pending, attribute)
	}

	slices.Sort(pending)

	return pending
}

// marshalImportAdoptToPrivate stores the pending adoption. An empty list is not stored at all: a
// payload that spelled everything out has nothing to adopt, and writing the key anyway would leave
// a flag behind that no apply would ever clear.
func marshalImportAdoptToPrivate(
	ctx context.Context,
	attributes []string,
	private privateStateWriter,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics

	if len(attributes) == 0 {
		return diagnostics
	}

	data, err := json.Marshal(importAdoptPrivate{Attributes: attributes})
	if err != nil {
		diagnostics.AddError(
			"Unable to record the pending import adoption",
			fmt.Sprintf("Failed to encode the private state: %v", err),
		)

		return diagnostics
	}

	return private.SetKey(ctx, importAdoptPrivateKey, data)
}

// unmarshalImportAdoptFromPrivate reads the pending adoption, returning nil when there is none.
func unmarshalImportAdoptFromPrivate(
	ctx context.Context,
	private privateStateReader,
) (*importAdoptPrivate, diag.Diagnostics) {
	if private == nil {
		return nil, nil
	}

	data, diagnostics := private.GetKey(ctx, importAdoptPrivateKey)
	if diagnostics.HasError() || len(data) == 0 {
		return nil, diagnostics
	}

	var adopt importAdoptPrivate
	if err := json.Unmarshal(data, &adopt); err != nil {
		diagnostics.AddError(
			"Unable to read the pending import adoption",
			fmt.Sprintf("Failed to decode the private state: %v", err),
		)

		return nil, diagnostics
	}

	return &adopt, diagnostics
}

// clearImportAdoptFromPrivate removes the pending adoption. Passing a zero-length value deletes the
// key outright, so a later genuine change is planned with the normal replacement rules.
func clearImportAdoptFromPrivate(ctx context.Context, private privateStateWriter) diag.Diagnostics {
	return private.SetKey(ctx, importAdoptPrivateKey, nil)
}

// isAdoptionPending reports whether the named attribute is waiting to be adopted.
func isAdoptionPending(adopt *importAdoptPrivate, attribute string) bool {
	if adopt == nil {
		return false
	}

	return slices.Contains(adopt.Attributes, attribute)
}

// requiresReplaceUnlessAdopted decides whether a changed attribute forces replacement.
//
// It returns false only while the attribute is listed in the pending import adoption, which is the
// single window between `terraform import` and the apply that follows it.
func requiresReplaceUnlessAdopted(
	ctx context.Context,
	attribute string,
	private privateStateReader,
	diagnostics *diag.Diagnostics,
) bool {
	adopt, diags := unmarshalImportAdoptFromPrivate(ctx, private)
	diagnostics.Append(diags...)

	if diags.HasError() {
		// Fall back to the stricter behaviour: an unreadable flag must not silently disable
		// replacement for a resource that genuinely needs it.
		return true
	}

	return !isAdoptionPending(adopt, attribute)
}

const (
	descAdoptReplace = "Changing this argument replaces the resource, except on the first plan " +
		"after an import that did not specify it, where the value is adopted from the configuration " +
		"in place."
	descAdoptReplaceMarkdown = "Changing this argument replaces the resource, except on the first " +
		"plan after a `terraform import` that did not specify it, where the value is adopted from " +
		"the configuration in place."
)

// replaceableStringAttribute builds a string argument that replaces the resource when it changes,
// unless the change is the first one after an import that left it unspecified.
func replaceableStringAttribute(required bool, description string) schema.StringAttribute {
	attribute := helpers.StringAttributeNoReplace(required, description)
	attribute.PlanModifiers = []planmodifier.String{
		stringplanmodifier.RequiresReplaceIf(
			func(
				ctx context.Context,
				req planmodifier.StringRequest,
				resp *stringplanmodifier.RequiresReplaceIfFuncResponse,
			) {
				resp.RequiresReplace = requiresReplaceUnlessAdopted(
					ctx, attributeNameOf(req.Path), req.Private, &resp.Diagnostics,
				)
			},
			descAdoptReplace,
			descAdoptReplaceMarkdown,
		),
	}

	return attribute
}

// replaceableMapAttribute builds a map argument with the same conditional replacement rule.
func replaceableMapAttribute(required bool, elementType attr.Type, description string) schema.MapAttribute {
	attribute := helpers.MapAttributeNoReplace(required, elementType, description)
	attribute.PlanModifiers = []planmodifier.Map{
		mapplanmodifier.RequiresReplaceIf(
			func(
				ctx context.Context,
				req planmodifier.MapRequest,
				resp *mapplanmodifier.RequiresReplaceIfFuncResponse,
			) {
				resp.RequiresReplace = requiresReplaceUnlessAdopted(
					ctx, attributeNameOf(req.Path), req.Private, &resp.Diagnostics,
				)
			},
			descAdoptReplace,
			descAdoptReplaceMarkdown,
		),
	}

	return attribute
}

// replaceableBoolAttribute builds a bool argument with the same conditional replacement rule.
func replaceableBoolAttribute(required bool, description string) schema.BoolAttribute {
	attribute := helpers.BoolAttributeNoReplace(required, description)
	attribute.PlanModifiers = []planmodifier.Bool{
		boolplanmodifier.RequiresReplaceIf(
			func(
				ctx context.Context,
				req planmodifier.BoolRequest,
				resp *boolplanmodifier.RequiresReplaceIfFuncResponse,
			) {
				resp.RequiresReplace = requiresReplaceUnlessAdopted(
					ctx, attributeNameOf(req.Path), req.Private, &resp.Diagnostics,
				)
			},
			descAdoptReplace,
			descAdoptReplaceMarkdown,
		),
	}

	return attribute
}

// attributeNameOf reduces an attribute path to its top-level name. Every adoptable attribute is a
// root attribute, so the string form of the path is the name itself.
func attributeNameOf(attributePath interface{ String() string }) string {
	return strings.TrimPrefix(attributePath.String(), ".")
}
