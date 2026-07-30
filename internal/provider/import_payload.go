package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// HTTPRequestResourceModelNative describes the resource data model in a native Go format.
//
// It is the wire shape of every JSON import payload, and each field's JSON name is exactly the
// schema attribute name it populates. That correspondence is what lets the decoder report which
// attributes an import payload actually mentioned, which in turn drives import adoption -- see
// [importPayload] and the `import_adopt` private-state key.
//
// Optional scalars are pointers on purpose. A payload that omits `is_response_body_json` has to
// leave the attribute null in state: decoding into a plain bool yields a concrete false, and
// `null -> false` is a change on the very next plan. For a RequiresReplace attribute that change
// means a destroy and a create, which is precisely the failure import is supposed to avoid.
type HTTPRequestResourceModelNative struct {
	// parameters
	Method               string            `json:"method"`
	Path                 string            `json:"path"`
	Headers              map[string]string `json:"headers,omitempty"`
	RequestBody          string            `json:"request_body,omitempty"`
	IsResponseBodyJSON   *bool             `json:"is_response_body_json,omitempty"`
	ResponseBodyIDFilter string            `json:"response_body_id_filter,omitempty"`
	QueryParameters      map[string]string `json:"query_parameters,omitempty"`
	ToleratedStatusCodes []int32           `json:"tolerated_status_codes,omitempty"`
	IgnoreChanges        []string          `json:"ignore_changes,omitempty"`

	// resource-level configuration (alternative to provider-level)
	BaseURL          string            `json:"base_url,omitempty"`
	BasicAuth        map[string]string `json:"basic_auth,omitempty"`
	IgnoreTLS        *bool             `json:"ignore_tls,omitempty"`
	RequestTimeoutMs *int64            `json:"request_timeout_ms,omitempty"`
	Retry            *retryNative      `json:"retry,omitempty"`

	// destroy controls
	IsDeleteEnabled   *bool             `json:"is_delete_enabled,omitempty"`
	DeleteMethod      string            `json:"delete_method,omitempty"`
	DeletePath        string            `json:"delete_path,omitempty"`
	DeleteHeaders     map[string]string `json:"delete_headers,omitempty"`
	DeleteRequestBody string            `json:"delete_request_body,omitempty"`

	// refresh controls
	IsRefreshEnabled *bool  `json:"is_refresh_enabled,omitempty"`
	RefreshPath      string `json:"refresh_path,omitempty"`

	// state
	ID               string            `json:"id,omitempty"`
	ResponseCode     *int32            `json:"response_code,omitempty"`
	ResponseBody     string            `json:"response_body,omitempty"`
	ResponseBodyID   string            `json:"response_body_id,omitempty"`
	ResponseBodyJSON map[string]string `json:"response_body_json,omitempty"`

	// ImportReadPath is import-only: it is not a schema attribute. When the payload describes a
	// resource created with an unsafe method (POST, PUT, PATCH), import refuses to replay that
	// method. Naming a path here lets import issue a GET against the already-created object so
	// the computed response attributes are captured from the live API instead of left null.
	ImportReadPath string `json:"import_read_path,omitempty"`
}

// retryNative is the native (JSON) representation of the `retry` block, used by
// the import payload.
type retryNative struct {
	Attempts   *int64 `json:"attempts,omitempty"`
	MinDelayMs *int64 `json:"min_delay_ms,omitempty"`
	MaxDelayMs *int64 `json:"max_delay_ms,omitempty"`
}

// importPayload is a decoded `terraform import` identifier.
type importPayload struct {
	native *HTTPRequestResourceModelNative
	// id is the value to store in the `id` attribute. Forms that carry no identifier of their
	// own get a freshly generated UUID.
	id string
	// specified holds the schema attribute names the identifier actually mentioned. Everything
	// adoptable that is absent from this set is adopted from configuration on the first plan
	// after the import.
	specified map[string]struct{}
}

// importForm describes one accepted shape of the import identifier.
//
// The forms are kept in an ordered slice rather than a map because dispatch order is part of the
// contract: the first detector that matches wins, and `bare base64` deliberately runs last so it
// can only ever claim what nothing more specific recognised.
type importForm struct {
	name    string
	example string
	detect  func(rawID string) bool
	decode  func(rawID string, diagnostics *diag.Diagnostics) *importPayload
}

const (
	// importFileSigil marks a payload stored in a file, borrowed from the `curl -d @file` idiom.
	// No URL path, UUID, base64 alphabet or JSON document can start with it.
	importFileSigil = "@"
	// importJSONSigil is the first byte of a raw JSON object. It cannot start a path, a UUID or
	// a base64 string either.
	importJSONSigil = "{"
	// importPathSigil is the first byte of a bare path. A legacy identifier always starts with a
	// UUID, so the two can never collide.
	importPathSigil = "/"

	// importIDParts is the number of segments in the `<uuid>/<base64>` identifier and in the
	// `METHOD path` shorthand. The legacy split is bounded to this many so a base64 payload
	// containing `/` (standard alphabet) stays intact.
	importIDParts = 2
)

// importForms returns the accepted identifier shapes in dispatch order.
//
// Every form is distinguished by its first byte (or by the presence of a space), so no two forms
// can claim the same identifier and the decoder never has to guess:
//
//   - `@` cannot begin a path, a UUID, base64 or JSON;
//   - `{` is outside the base64 alphabet and cannot begin a path or a UUID;
//   - base64, UUIDs and paths never contain a space;
//   - a legacy identifier begins with a UUID, so it never begins with `/`;
//   - the legacy form additionally requires its first segment to parse as a UUID, not merely to
//     be followed by a slash;
//   - bare base64 runs last and must both decode and yield a JSON object.
func importForms() []importForm {
	return []importForm{
		{
			name:    "file reference",
			example: `@./import/example.json`,
			detect:  func(rawID string) bool { return strings.HasPrefix(rawID, importFileSigil) },
			decode:  decodeImportFile,
		},
		{
			name:    "raw JSON",
			example: `{"method":"GET","path":"/posts/1"}`,
			detect: func(rawID string) bool {
				return strings.HasPrefix(strings.TrimSpace(rawID), importJSONSigil)
			},
			decode: decodeImportJSON,
		},
		{
			name:    "method and path",
			example: `GET /posts/1`,
			detect:  func(rawID string) bool { return strings.ContainsAny(rawID, " \t") },
			decode:  decodeImportMethodPath,
		},
		{
			name:    "bare path",
			example: `/posts/1`,
			detect:  func(rawID string) bool { return strings.HasPrefix(rawID, importPathSigil) },
			decode:  decodeImportPath,
		},
		{
			name:    "UUID and base64 payload",
			example: `1e1a3f3c-0d2a-4a1f-9b1e-2f2c9b0f6a51/eyJtZXRob2QiOiJHRVQifQ`,
			detect:  detectImportLegacy,
			decode:  decodeImportLegacy,
		},
		{
			name:    "bare base64 payload",
			example: `eyJtZXRob2QiOiJHRVQiLCJwYXRoIjoiL3Bvc3RzLzEifQ`,
			detect:  detectImportBase64,
			decode:  decodeImportBase64,
		},
	}
}

// decodeImportID resolves an import identifier into a payload, or reports an actionable error
// listing every accepted shape.
func decodeImportID(rawID string, diagnostics *diag.Diagnostics) *importPayload {
	trimmed := strings.TrimSpace(rawID)
	if trimmed == "" {
		addUnrecognisedImportIDError(diagnostics, "the import identifier is empty")
		return nil
	}

	for _, form := range importForms() {
		if form.detect(trimmed) {
			return form.decode(trimmed, diagnostics)
		}
	}

	addUnrecognisedImportIDError(
		diagnostics,
		"it does not match any accepted shape",
	)

	return nil
}

// addUnrecognisedImportIDError reports the accepted shapes with one example each.
func addUnrecognisedImportIDError(diagnostics *diag.Diagnostics, cause string) {
	var builder strings.Builder
	builder.WriteString("The import identifier could not be recognised because ")
	builder.WriteString(cause)
	builder.WriteString(".\n\nAccepted forms:\n")

	for _, form := range importForms() {
		fmt.Fprintf(&builder, "  - %s: %s\n", form.name, form.example)
	}

	builder.WriteString(
		"\nOnly `method` and `path` are required. Every other argument left out of the payload " +
			"is adopted from your configuration on the first plan after the import, so the " +
			"shortest form is usually enough.",
	)

	diagnostics.AddError("Invalid import identifier", builder.String())
}

// decodeImportFile reads the JSON payload from the file named after the `@` sigil.
func decodeImportFile(rawID string, diagnostics *diag.Diagnostics) *importPayload {
	name := strings.TrimSpace(strings.TrimPrefix(rawID, importFileSigil))
	if name == "" {
		diagnostics.AddError(
			"Invalid import identifier",
			"The `@` prefix must be followed by a file path, for example `@./import/example.json`.",
		)

		return nil
	}

	// #nosec G304 -- the path is typed by the practitioner into their own `terraform import`
	// invocation and is read with their own credentials, which is the same trust level as the
	// built-in `file()` function in a Terraform configuration.
	content, err := os.ReadFile(name)
	if err != nil {
		diagnostics.AddError(
			"Unable to read the import payload file",
			fmt.Sprintf("Failed to read %q: %v", name, err),
		)

		return nil
	}

	return unmarshalImportJSON(content, diagnostics)
}

// decodeImportJSON decodes a raw JSON object supplied directly as the identifier.
func decodeImportJSON(rawID string, diagnostics *diag.Diagnostics) *importPayload {
	return unmarshalImportJSON([]byte(rawID), diagnostics)
}

// decodeImportMethodPath decodes the `GET /posts/1` shorthand.
func decodeImportMethodPath(rawID string, diagnostics *diag.Diagnostics) *importPayload {
	fields := strings.Fields(rawID)
	if len(fields) != importIDParts {
		diagnostics.AddError(
			"Invalid import identifier",
			fmt.Sprintf(
				"Expected a method and a path separated by a space, for example `GET /posts/1`, got %q.",
				rawID,
			),
		)

		return nil
	}

	method := strings.ToUpper(fields[0])
	if !isKnownHTTPMethod(method) {
		diagnostics.AddError(
			"Invalid import identifier",
			fmt.Sprintf("%q is not a recognised HTTP method.", fields[0]),
		)

		return nil
	}

	return newShorthandImportPayload(method, fields[1], true)
}

// decodeImportPath decodes the bare `/posts/1` shorthand, defaulting the method to GET. This
// mirrors the identifier accepted by the Mastercard `restapi` provider.
func decodeImportPath(rawID string, _ *diag.Diagnostics) *importPayload {
	return newShorthandImportPayload(http.MethodGet, rawID, false)
}

// detectImportLegacy reports whether the identifier is the `<id>/<base64>` form.
//
// The split is bounded to two segments and the *second* one has to decode to a JSON object. A
// looser test would be wrong in both directions: the standard base64 alphabet contains `/`, so
// counting slashes would reject a legitimate payload, while accepting any two segments would
// swallow identifiers meant for another form. The leading segment is deliberately not required to
// be a UUID -- earlier releases documented one but never enforced it, and identifiers already in
// use must keep working.
func detectImportLegacy(rawID string) bool {
	parts := strings.SplitN(rawID, importPathSigil, importIDParts)
	if len(parts) != importIDParts || parts[0] == "" {
		return false
	}

	return decodesToJSONObject(parts[1])
}

// decodeImportLegacy decodes the `<id>/<base64>` form kept for backwards compatibility.
func decodeImportLegacy(rawID string, diagnostics *diag.Diagnostics) *importPayload {
	parts := strings.SplitN(rawID, importPathSigil, importIDParts)

	decoded, err := decodeAnyBase64(parts[1])
	if err != nil {
		diagnostics.AddError(
			"Invalid import identifier",
			fmt.Sprintf("The payload after the identifier is not valid base64: %v", err),
		)

		return nil
	}

	payload := unmarshalImportJSON(decoded, diagnostics)
	if payload == nil {
		return nil
	}

	payload.id = parts[0]

	return payload
}

// detectImportBase64 reports whether the identifier is a bare base64 payload. It is the last form
// tried, so it may only claim identifiers that both decode and hold a JSON object.
func detectImportBase64(rawID string) bool {
	return decodesToJSONObject(rawID)
}

// decodesToJSONObject reports whether the value is base64 that decodes to a JSON object. Requiring
// the object -- not merely a successful decode -- is what keeps the base64 forms from claiming
// arbitrary text that happens to sit in the alphabet.
func decodesToJSONObject(value string) bool {
	decoded, err := decodeAnyBase64(value)
	if err != nil {
		return false
	}

	return json.Valid(decoded) &&
		strings.HasPrefix(strings.TrimSpace(string(decoded)), importJSONSigil)
}

// decodeImportBase64 decodes a bare base64 payload.
func decodeImportBase64(rawID string, diagnostics *diag.Diagnostics) *importPayload {
	decoded, err := decodeAnyBase64(rawID)
	if err != nil {
		diagnostics.AddError(
			"Invalid import identifier",
			fmt.Sprintf("The identifier is not valid base64: %v", err),
		)

		return nil
	}

	return unmarshalImportJSON(decoded, diagnostics)
}

// decodeAnyBase64 decodes standard and URL-safe base64, padded or not.
//
// Identifiers are copied by hand out of shells and CI logs, where padding is easy to lose and the
// standard alphabet's `/` is awkward, so all four spellings are accepted. Documentation emits the
// URL-safe form, which never contains a slash.
func decodeAnyBase64(value string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}

	var lastErr error

	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("%w", lastErr)
}

// newShorthandImportPayload builds a payload from the method/path shorthands. Everything else is
// left to adoption.
func newShorthandImportPayload(method, path string, methodSpecified bool) *importPayload {
	specified := map[string]struct{}{attrPath: {}}
	if methodSpecified {
		specified[attrMethod] = struct{}{}
	}

	return &importPayload{
		native:    &HTTPRequestResourceModelNative{Method: method, Path: path},
		id:        uuid.NewString(),
		specified: specified,
	}
}

// unmarshalImportJSON decodes a JSON payload into both the native model and the set of attribute
// names it mentioned.
//
// The document is decoded twice on purpose. `encoding/json` cannot report which keys were present
// -- an absent key and an explicit null are indistinguishable after unmarshalling -- so the raw
// key set is captured separately and is what adoption keys off.
func unmarshalImportJSON(data []byte, diagnostics *diag.Diagnostics) *importPayload {
	var native HTTPRequestResourceModelNative
	if err := json.Unmarshal(data, &native); err != nil {
		diagnostics.AddError(
			"Invalid import payload",
			fmt.Sprintf("The payload is not a valid JSON representation of the resource: %v", err),
		)

		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		diagnostics.AddError(
			"Invalid import payload",
			fmt.Sprintf("The payload is not a JSON object: %v", err),
		)

		return nil
	}

	id := native.ID
	if id == "" {
		id = uuid.NewString()
	}

	return &importPayload{
		native:    &native,
		id:        id,
		specified: adoptableKeysOf(raw),
	}
}

// adoptableKeysOf keeps only the payload keys that name an attribute adoption can act on. Keys
// such as `response_body` are computed, and `import_read_path` is import-only, so neither can be
// adopted from configuration.
func adoptableKeysOf(raw map[string]json.RawMessage) map[string]struct{} {
	adoptable := getSupportedIgnoreAttributes()
	specified := make(map[string]struct{}, len(raw))

	for key := range raw {
		if _, ok := adoptable[key]; ok {
			specified[key] = struct{}{}
		}
	}

	return specified
}

// isKnownHTTPMethod reports whether the value is one of the methods the resource can issue.
func isKnownHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace,
		http.MethodConnect:
		return true
	default:
		return false
	}
}

// isSafeHTTPMethod reports whether replaying the method during import is free of side effects.
// Only these are ever issued automatically; anything else needs an explicit `import_read_path`.
func isSafeHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

// buildImportID renders the identifier that re-imports this exact resource.
//
// It emits the `<uuid>/<base64>` form with URL-safe, unpadded base64, which is the only spelling
// that never contains a `/` and so cannot be mistaken for the separator.
//
// Only the arguments a configuration can set are encoded. Three groups are deliberately left out:
//
//   - `basic_auth`, because the identifier lives in plain state and is meant to be copied into
//     shells and CI logs, where a credential would leak. The configuration supplies it on import.
//   - the captured response (`response_code`, `response_body`, and the `response_body_id` /
//     `response_body_json` derived from it). A response body is frequently the most sensitive
//     thing the resource ever holds -- creation endpoints routinely return tokens and personal
//     data -- and it is already in state under its own attribute, so embedding a base64 copy would
//     both leak it into logs and inflate the state by a third of its size for nothing. A re-import
//     captures it live instead, which is also the only way to get a *current* one.
//   - the write-only destroy controls, for the same reason they are absent from state.
//
// Re-reading is only automatic for the safe methods, so for anything else the resolved refresh
// path is carried as `import_read_path` when one is configured -- a plain URL, not response data,
// and one already visible in state as `delete_resolved_path`.
func buildImportID(
	ctx context.Context,
	model HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) types.String {
	native := HTTPRequestResourceModelNative{
		Method:               model.Method.ValueString(),
		Path:                 model.Path.ValueString(),
		Headers:              stringMapOf(ctx, model.Headers, diagnostics),
		RequestBody:          model.RequestBody.ValueString(),
		IsResponseBodyJSON:   boolValueToPtr(model.IsResponseBodyJSON),
		ResponseBodyIDFilter: model.ResponseBodyIDFilter.ValueString(),
		QueryParameters:      stringMapOf(ctx, model.QueryParameters, diagnostics),
		ToleratedStatusCodes: int32SliceOf(ctx, model.ToleratedStatusCodes, diagnostics),
		IgnoreChanges:        stringSliceOf(ctx, model.IgnoreChanges, diagnostics),
		BaseURL:              model.BaseURL.ValueString(),
		IgnoreTLS:            boolValueToPtr(model.IgnoreTLS),
		RequestTimeoutMs:     int64ValueToPtr(model.RequestTimeoutMs),
		Retry:                retryNativeFromObject(model.Retry),
		IsRefreshEnabled:     boolValueToPtr(model.IsRefreshEnabled),
		RefreshPath:          model.RefreshPath.ValueString(),
		ImportReadPath:       importReadPathForIdentifier(model),
	}

	if diagnostics.HasError() {
		return types.StringNull()
	}

	encoded, err := json.Marshal(native)
	if err != nil {
		diagnostics.AddError(
			"Unable to build the import identifier",
			fmt.Sprintf("Failed to encode the resource: %v", err),
		)

		return types.StringNull()
	}

	return types.StringValue(
		model.ID.ValueString() + importPathSigil + base64.RawURLEncoding.EncodeToString(encoded),
	)
}

// importReadPathForIdentifier returns the path a re-import should read to recapture the response,
// or an empty string when there is none to offer.
//
// Only a resource created with an unsafe method needs it: a safe one is re-read automatically. The
// configured `refresh_path` is resolved against the captured body here, while that body is still at
// hand, so the identifier carries a concrete URL rather than the JSONPath tokens a fresh import
// would have nothing to resolve against.
func importReadPathForIdentifier(model HTTPRequestResourceModel) string {
	if isSafeHTTPMethod(model.Method.ValueString()) || !isNonEmptyString(model.RefreshPath) {
		return ""
	}

	// Failure here is not worth surfacing: the identifier is a convenience, and an import without
	// a read path still succeeds with a warning explaining how to supply one.
	var ignored diag.Diagnostics

	resolved, ok := resolveDeletePathTokens(
		model.RefreshPath.ValueString(),
		model.ResponseBody.ValueString(),
		&ignored,
	)
	if !ok {
		return ""
	}

	return resolved
}

// retryNativeFromObject converts the `retry` block back into its native representation.
func retryNativeFromObject(object types.Object) *retryNative {
	if object.IsNull() || object.IsUnknown() {
		return nil
	}

	attributes := object.Attributes()
	native := &retryNative{}

	if value, ok := attributes[attrAttempts].(types.Int64); ok {
		native.Attempts = int64ValueToPtr(value)
	}

	if value, ok := attributes[attrMinDelayMs].(types.Int64); ok {
		native.MinDelayMs = int64ValueToPtr(value)
	}

	if value, ok := attributes[attrMaxDelayMs].(types.Int64); ok {
		native.MaxDelayMs = int64ValueToPtr(value)
	}

	return native
}

// stringMapOf converts a framework map into its native form, yielding nil when unset so the JSON
// encoder omits the key entirely.
func stringMapOf(ctx context.Context, value types.Map, diagnostics *diag.Diagnostics) map[string]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var converted map[string]string
	diagnostics.Append(value.ElementsAs(ctx, &converted, false)...)

	return converted
}

// stringSliceOf converts a framework set of strings into its native form.
func stringSliceOf(ctx context.Context, value types.Set, diagnostics *diag.Diagnostics) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var converted []string
	diagnostics.Append(value.ElementsAs(ctx, &converted, false)...)

	return converted
}

// int32SliceOf converts a framework set of int32 into its native form.
func int32SliceOf(ctx context.Context, value types.Set, diagnostics *diag.Diagnostics) []int32 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var converted []int32
	diagnostics.Append(value.ElementsAs(ctx, &converted, false)...)

	return converted
}

// boolValueToPtr converts a framework bool into an optional native bool.
func boolValueToPtr(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	converted := value.ValueBool()

	return &converted
}

// int64ValueToPtr converts a framework int64 into an optional native int64.
func int64ValueToPtr(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	converted := value.ValueInt64()

	return &converted
}

// buildModelFromNativeData converts a decoded payload into the resource model.
func buildModelFromNativeData(
	nativeModel *HTTPRequestResourceModelNative,
	diagnostics *diag.Diagnostics,
) *HTTPRequestResourceModel {
	model := createBaseModel(nativeModel)

	setOptionalStringFields(model, nativeModel)
	setResourceLevelFields(model, nativeModel, diagnostics)
	if diagnostics.HasError() {
		return nil
	}

	setMapFields(model, nativeModel, diagnostics)
	if diagnostics.HasError() {
		return nil
	}

	setToleratedStatusCodesField(model, nativeModel, diagnostics)
	if diagnostics.HasError() {
		return nil
	}

	setIgnoreChangesField(model, nativeModel, diagnostics)
	if diagnostics.HasError() {
		return nil
	}

	return model
}

// createBaseModel seeds the model with the always-present attributes.
//
// Every optional value stays null unless the payload named it. A concrete zero here would diff
// against a configuration that leaves the argument out and, for a RequiresReplace attribute,
// force a destroy and a create on the first plan after the import.
func createBaseModel(nativeModel *HTTPRequestResourceModelNative) *HTTPRequestResourceModel {
	model := &HTTPRequestResourceModel{
		Method:               types.StringValue(nativeModel.Method),
		Path:                 types.StringValue(nativeModel.Path),
		IsResponseBodyJSON:   boolPtrToValue(nativeModel.IsResponseBodyJSON),
		ResponseCode:         int32PtrToValue(nativeModel.ResponseCode),
		IsDeleteEnabled:      boolPtrToValue(nativeModel.IsDeleteEnabled),
		IsRefreshEnabled:     boolPtrToValue(nativeModel.IsRefreshEnabled),
		ToleratedStatusCodes: types.SetNull(types.Int32Type),
		IgnoreChanges:        types.SetNull(types.StringType),
	}

	return model
}

// setOptionalStringFields copies the optional string attributes, leaving absent ones null.
func setOptionalStringFields(model *HTTPRequestResourceModel, nativeModel *HTTPRequestResourceModelNative) {
	assignments := []struct {
		target *types.String
		value  string
	}{
		{&model.DeleteMethod, nativeModel.DeleteMethod},
		{&model.DeletePath, nativeModel.DeletePath},
		{&model.DeleteRequestBody, nativeModel.DeleteRequestBody},
		{&model.RefreshPath, nativeModel.RefreshPath},
		{&model.RequestBody, nativeModel.RequestBody},
		{&model.ResponseBodyIDFilter, nativeModel.ResponseBodyIDFilter},
		{&model.ResponseBody, nativeModel.ResponseBody},
		{&model.ResponseBodyID, nativeModel.ResponseBodyID},
		{&model.BaseURL, nativeModel.BaseURL},
	}

	for _, assignment := range assignments {
		if assignment.value == "" {
			*assignment.target = types.StringNull()

			continue
		}

		*assignment.target = types.StringValue(assignment.value)
	}
}

// setResourceLevelFields copies the per-resource overrides of the provider configuration.
func setResourceLevelFields(
	model *HTTPRequestResourceModel,
	nativeModel *HTTPRequestResourceModelNative,
	diagnostics *diag.Diagnostics,
) {
	model.IgnoreTLS = boolPtrToValue(nativeModel.IgnoreTLS)

	// Operational tuning knobs (optional). retry must always be a typed object so the imported
	// state is plannable; a typed null is used when absent.
	model.RequestTimeoutMs = int64PtrToValue(nativeModel.RequestTimeoutMs)
	model.Retry = retryObjectFromNative(nativeModel.Retry, diagnostics)

	setBasicAuthField(model, nativeModel, diagnostics)
}

// setBasicAuthField rebuilds the `basic_auth` nested object, always with its declared type so the
// imported state stays plannable.
func setBasicAuthField(
	model *HTTPRequestResourceModel,
	nativeModel *HTTPRequestResourceModelNative,
	diagnostics *diag.Diagnostics,
) {
	attributeTypes := map[string]attr.Type{
		attrUsername: types.StringType,
		attrPassword: types.StringType,
	}

	if len(nativeModel.BasicAuth) == 0 {
		model.BasicAuth = types.ObjectNull(attributeTypes)

		return
	}

	values := make(map[string]attr.Value, len(attributeTypes))
	for _, name := range []string{attrUsername, attrPassword} {
		if value, ok := nativeModel.BasicAuth[name]; ok {
			values[name] = types.StringValue(value)

			continue
		}

		values[name] = types.StringNull()
	}

	object, diags := types.ObjectValue(attributeTypes, values)
	if diags.HasError() {
		diagnostics.Append(diags...)

		return
	}

	model.BasicAuth = object
}

// setMapFields copies the map attributes. A nil Go map converts to a typed null map, which is the
// correct representation of an argument the configuration leaves out.
func setMapFields(
	model *HTTPRequestResourceModel,
	nativeModel *HTTPRequestResourceModelNative,
	diagnostics *diag.Diagnostics,
) {
	assignments := []struct {
		target *types.Map
		value  map[string]string
	}{
		{&model.Headers, nativeModel.Headers},
		{&model.QueryParameters, nativeModel.QueryParameters},
		{&model.DeleteHeaders, nativeModel.DeleteHeaders},
		{&model.ResponseBodyJSON, nativeModel.ResponseBodyJSON},
	}

	for _, assignment := range assignments {
		value, diags := types.MapValueFrom(context.Background(), types.StringType, assignment.value)
		if diags.HasError() {
			diagnostics.Append(diags...)

			return
		}

		*assignment.target = value
	}
}

// setToleratedStatusCodesField copies `tolerated_status_codes`, leaving it null when absent.
func setToleratedStatusCodesField(
	model *HTTPRequestResourceModel,
	nativeModel *HTTPRequestResourceModelNative,
	diagnostics *diag.Diagnostics,
) {
	if len(nativeModel.ToleratedStatusCodes) == 0 {
		return
	}

	value, diags := types.SetValueFrom(
		context.Background(),
		types.Int32Type,
		nativeModel.ToleratedStatusCodes,
	)
	diagnostics.Append(diags...)

	if diagnostics.HasError() {
		return
	}

	model.ToleratedStatusCodes = value
}

// setIgnoreChangesField copies `ignore_changes`, leaving it null when absent.
func setIgnoreChangesField(
	model *HTTPRequestResourceModel,
	nativeModel *HTTPRequestResourceModelNative,
	diagnostics *diag.Diagnostics,
) {
	if len(nativeModel.IgnoreChanges) == 0 {
		return
	}

	value, diags := types.SetValueFrom(
		context.Background(),
		types.StringType,
		nativeModel.IgnoreChanges,
	)
	diagnostics.Append(diags...)

	if diagnostics.HasError() {
		return
	}

	model.IgnoreChanges = value
}

// boolPtrToValue converts an optional native bool into its framework representation.
func boolPtrToValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}

	return types.BoolValue(*value)
}

// int32PtrToValue converts an optional native int32 into its framework representation.
func int32PtrToValue(value *int32) types.Int32 {
	if value == nil {
		return types.Int32Null()
	}

	return types.Int32Value(*value)
}

// int64PtrToValue converts an optional native int64 into its framework representation.
func int64PtrToValue(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

// retryObjectFromNative reconstructs a typed `retry` object from the import payload, returning a
// typed null when no retry data is present.
func retryObjectFromNative(nativeRetry *retryNative, diagnostics *diag.Diagnostics) types.Object {
	if nativeRetry == nil {
		return types.ObjectNull(retryObjectAttrTypes())
	}

	object, diags := types.ObjectValue(retryObjectAttrTypes(), map[string]attr.Value{
		attrAttempts:   int64PtrToValue(nativeRetry.Attempts),
		attrMinDelayMs: int64PtrToValue(nativeRetry.MinDelayMs),
		attrMaxDelayMs: int64PtrToValue(nativeRetry.MaxDelayMs),
	})
	if diags.HasError() {
		diagnostics.Append(diags...)

		return types.ObjectNull(retryObjectAttrTypes())
	}

	return object
}
