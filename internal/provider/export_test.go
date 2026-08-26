package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// This file is compiled only during tests, so the shims below let the external `provider_test`
// package reach unexported behaviour without widening the shipped API.

// DecodeImportIDForTest decodes an import identifier into the resource model it produces, together
// with the set of attributes the identifier actually mentioned.
func DecodeImportIDForTest(
	rawID string,
	diagnostics *diag.Diagnostics,
) (*HTTPRequestResourceModel, map[string]struct{}) {
	payload := decodeImportID(rawID, diagnostics)
	if payload == nil {
		return nil, nil
	}

	model := buildModelFromNativeData(payload.native, diagnostics)
	if model == nil {
		return nil, nil
	}

	model.ID = types.StringValue(payload.id)

	return model, payload.specified
}

// BuildImportIDForTest renders the identifier that re-imports a model.
func BuildImportIDForTest(
	ctx context.Context,
	model HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) types.String {
	return buildImportID(ctx, model, diagnostics)
}

// PendingAdoptionForTest returns the adoptable attributes an identifier left unspecified.
func PendingAdoptionForTest(specified map[string]struct{}) []string {
	return pendingAdoptionFor(specified)
}

// MarshalImportAdoptForTest records a pending adoption in private state.
func MarshalImportAdoptForTest(
	ctx context.Context,
	attributes []string,
	private privateStateWriter,
) diag.Diagnostics {
	return marshalImportAdoptToPrivate(ctx, attributes, private)
}

// UnmarshalImportAdoptForTest reads a pending adoption back, returning nil when there is none.
func UnmarshalImportAdoptForTest(
	ctx context.Context,
	private privateStateReader,
) ([]string, diag.Diagnostics) {
	adopt, diagnostics := unmarshalImportAdoptFromPrivate(ctx, private)
	if adopt == nil {
		return nil, diagnostics
	}

	return adopt.Attributes, diagnostics
}

// ClearImportAdoptForTest removes a pending adoption from private state.
func ClearImportAdoptForTest(ctx context.Context, private privateStateWriter) diag.Diagnostics {
	return clearImportAdoptFromPrivate(ctx, private)
}

// RequiresReplaceUnlessAdoptedForTest reports whether a changed attribute forces replacement.
func RequiresReplaceUnlessAdoptedForTest(
	ctx context.Context,
	attribute string,
	private privateStateReader,
	diagnostics *diag.Diagnostics,
) bool {
	return requiresReplaceUnlessAdopted(ctx, attribute, private, diagnostics)
}

// MarshalDeleteParamsForTest stores the write-only destroy controls in private state.
func MarshalDeleteParamsForTest(
	ctx context.Context,
	model HTTPRequestResourceModel,
	private privateStateWriter,
) diag.Diagnostics {
	return marshalDeleteParamsToPrivate(ctx, model, private)
}

// ApplyDeleteParamsForTest restores the write-only destroy controls from private state.
func ApplyDeleteParamsForTest(
	ctx context.Context,
	model *HTTPRequestResourceModel,
	private privateStateReader,
) diag.Diagnostics {
	params, diagnostics := unmarshalDeleteParamsFromPrivate(ctx, private)
	if diagnostics.HasError() {
		return diagnostics
	}

	applyDeleteParamsToModel(model, params)

	return diagnostics
}

// ShadowedHeaderNamesForTest reports the delete-header names the provider configuration also sets.
func ShadowedHeaderNamesForTest(
	providerHeaders map[string]string,
	deleteHeaderNames []string,
) []string {
	return shadowedHeaderNames(providerHeaders, deleteHeaderNames)
}

// FormatHeaderListForTest renders header names the way the shadowing diagnostic does.
func FormatHeaderListForTest(names []string) string {
	return formatHeaderList(names)
}
