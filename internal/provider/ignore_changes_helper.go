package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IgnoreAttributeKind represents the type of attribute for ignore_changes handling.
type IgnoreAttributeKind int

const (
	// IgnoreKindScalar represents scalar attributes (string, bool).
	IgnoreKindScalar IgnoreAttributeKind = iota
	// IgnoreKindMap represents map attributes.
	IgnoreKindMap
	// IgnoreKindBody represents JSON body attributes.
	IgnoreKindBody
	// IgnoreKindObject represents nested object attributes.
	IgnoreKindObject
)

// IgnoreEntry represents a single ignore_changes entry with attribute and optional subpath.
type IgnoreEntry struct {
	Attribute string
	SubPath   []string
}

// NewIgnoreEntry creates a new IgnoreEntry for testing purposes.
func NewIgnoreEntry(attribute string, subPath []string) IgnoreEntry {
	return IgnoreEntry{Attribute: attribute, SubPath: subPath}
}

type ignoreApplier func(
	context.Context,
	[]IgnoreEntry,
	*HTTPRequestResourceModel,
	*HTTPRequestResourceModel,
	*diag.Diagnostics,
) bool

type (
	stringFieldAccessor func(*HTTPRequestResourceModel) *types.String
	boolFieldAccessor   func(*HTTPRequestResourceModel) *types.Bool
	mapFieldAccessor    func(*HTTPRequestResourceModel) *types.Map
	objectFieldAccessor func(*HTTPRequestResourceModel) *types.Object
)

// getSupportedIgnoreAttributes returns the map of supported attributes for ignore_changes.
// Note: is_delete_enabled, delete_method, delete_path, delete_headers, delete_request_body
// are NOT included here because they never trigger replacement (they only affect destroy behavior).
func getSupportedIgnoreAttributes() map[string]IgnoreAttributeKind {
	return map[string]IgnoreAttributeKind{
		"method":                  IgnoreKindScalar,
		"path":                    IgnoreKindScalar,
		"headers":                 IgnoreKindMap,
		"request_body":            IgnoreKindBody,
		"query_parameters":        IgnoreKindMap,
		"base_url":                IgnoreKindScalar,
		"basic_auth":              IgnoreKindObject,
		"ignore_tls":              IgnoreKindScalar,
		"is_response_body_json":   IgnoreKindScalar,
		"response_body_id_filter": IgnoreKindScalar,
	}
}

// getIgnoreAppliers returns the map of ignore appliers for each supported attribute.
// Note: delete_* fields are NOT included here because they never trigger replacement
// (they only affect destroy behavior and use NoReplace schema modifiers).
func getIgnoreAppliers() map[string]ignoreApplier {
	methodGetter := func(m *HTTPRequestResourceModel) *types.String { return &m.Method }
	pathGetter := func(m *HTTPRequestResourceModel) *types.String { return &m.Path }
	baseURLGetter := func(m *HTTPRequestResourceModel) *types.String { return &m.BaseURL }
	responseBodyIDFilterGetter := func(m *HTTPRequestResourceModel) *types.String { return &m.ResponseBodyIDFilter }
	ignoreTLSGetter := func(m *HTTPRequestResourceModel) *types.Bool { return &m.IgnoreTLS }
	isResponseBodyJSONGetter := func(m *HTTPRequestResourceModel) *types.Bool { return &m.IsResponseBodyJSON }
	headersGetter := func(m *HTTPRequestResourceModel) *types.Map { return &m.Headers }
	queryParametersGetter := func(m *HTTPRequestResourceModel) *types.Map { return &m.QueryParameters }
	requestBodyGetter := func(m *HTTPRequestResourceModel) *types.String { return &m.RequestBody }
	basicAuthGetter := func(m *HTTPRequestResourceModel) *types.Object { return &m.BasicAuth }

	return map[string]ignoreApplier{
		"method": makeStringApplier(
			methodGetter,
			methodGetter,
		),
		"path": makeStringApplier(
			pathGetter,
			pathGetter,
		),
		"base_url": makeStringApplier(
			baseURLGetter,
			baseURLGetter,
		),
		"response_body_id_filter": makeStringApplier(
			responseBodyIDFilterGetter,
			responseBodyIDFilterGetter,
		),
		"ignore_tls": makeBoolApplier(
			ignoreTLSGetter,
			ignoreTLSGetter,
		),
		"is_response_body_json": makeBoolApplier(
			isResponseBodyJSONGetter,
			isResponseBodyJSONGetter,
		),
		"headers": makeMapApplier(
			headersGetter,
			headersGetter,
		),
		"query_parameters": makeMapApplier(
			queryParametersGetter,
			queryParametersGetter,
		),
		"request_body": makeBodyApplier(
			requestBodyGetter,
			requestBodyGetter,
		),
		"basic_auth": makeObjectApplier(
			basicAuthGetter,
			basicAuthGetter,
		),
	}
}

// GetSupportedIgnoreAttributes returns the map of supported attributes for ignore_changes (exported for testing).
func GetSupportedIgnoreAttributes() map[string]IgnoreAttributeKind {
	return getSupportedIgnoreAttributes()
}

// ParseIgnoreEntries parses ignore_changes entries from a types.Set (exported for testing).
func ParseIgnoreEntries(
	ctx context.Context,
	value types.Set,
	diagnostics *diag.Diagnostics,
) []IgnoreEntry {
	return parseIgnoreEntries(ctx, value, diagnostics)
}

// ApplyIgnoreEntries applies ignore rules to the plan model (exported for testing).
func ApplyIgnoreEntries(
	ctx context.Context,
	entries []IgnoreEntry,
	plan *HTTPRequestResourceModel,
	state *HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) bool {
	return applyIgnoreEntries(ctx, entries, plan, state, diagnostics)
}

func parseIgnoreEntries(
	ctx context.Context,
	value types.Set,
	diagnostics *diag.Diagnostics,
) []IgnoreEntry {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var rawEntries []string
	diagnostics.Append(value.ElementsAs(ctx, &rawEntries, false)...)
	if diagnostics.HasError() {
		return nil
	}

	supportedAttrs := getSupportedIgnoreAttributes()
	seen := make(map[string]struct{})
	entries := make([]IgnoreEntry, 0, len(rawEntries))

	for _, raw := range rawEntries {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		parts := strings.Split(trimmed, ".")
		attribute := strings.ToLower(strings.TrimSpace(parts[0]))

		kind, ok := supportedAttrs[attribute]
		if !ok {
			diagnostics.AddWarning(
				"Unsupported ignore_changes entry",
				fmt.Sprintf(
					"Entry %q references attribute %q which is not supported. This entry will be ignored.",
					trimmed,
					attribute,
				),
			)
			continue
		}

		subPath := make([]string, 0, len(parts)-1)
		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			subPath = append(subPath, part)
		}

		if err := validateIgnorePath(kind, subPath); err != nil {
			diagnostics.AddError("Invalid ignore_changes entry", fmt.Sprintf("Entry %q is invalid: %s", trimmed, err))
			continue
		}

		key := attribute + "|" + strings.Join(subPath, ".")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		entries = append(entries, IgnoreEntry{Attribute: attribute, SubPath: subPath})
	}

	return entries
}

func validateIgnorePath(kind IgnoreAttributeKind, subPath []string) error {
	switch kind {
	case IgnoreKindScalar, IgnoreKindObject:
		if len(subPath) > 0 {
			return errors.New("nested paths are not supported for this attribute")
		}
	case IgnoreKindMap:
		if len(subPath) > 1 {
			return errors.New("only a single map key can be referenced (e.g. headers.Authorization)")
		}
	case IgnoreKindBody:
		// Allow zero-length (entire body) or nested paths
	default:
		return errors.New("unsupported ignore attribute type")
	}
	return nil
}

func applyIgnoreEntries(
	ctx context.Context,
	entries []IgnoreEntry,
	plan *HTTPRequestResourceModel,
	state *HTTPRequestResourceModel,
	diagnostics *diag.Diagnostics,
) bool {
	if len(entries) == 0 {
		return false
	}

	grouped := make(map[string][]IgnoreEntry)
	for _, entry := range entries {
		grouped[entry.Attribute] = append(grouped[entry.Attribute], entry)
	}

	appliers := getIgnoreAppliers()
	changed := false

	for attribute, attrEntries := range grouped {
		if handler, ok := appliers[attribute]; ok {
			changed = handler(ctx, attrEntries, plan, state, diagnostics) || changed
			continue
		}
		diagnostics.AddWarning(
			"Unsupported ignore_changes entry",
			fmt.Sprintf("Attribute %q does not have a registered ignore handler", attribute),
		)
	}

	return changed
}

func makeStringApplier(planGetter, stateGetter stringFieldAccessor) ignoreApplier {
	return func(_ context.Context, _ []IgnoreEntry, plan, state *HTTPRequestResourceModel, _ *diag.Diagnostics) bool {
		return setStringAttribute(planGetter(plan), *stateGetter(state))
	}
}

func makeBoolApplier(planGetter, stateGetter boolFieldAccessor) ignoreApplier {
	return func(_ context.Context, _ []IgnoreEntry, plan, state *HTTPRequestResourceModel, _ *diag.Diagnostics) bool {
		return setBoolAttribute(planGetter(plan), *stateGetter(state))
	}
}

func makeObjectApplier(planGetter, stateGetter objectFieldAccessor) ignoreApplier {
	return func(_ context.Context, _ []IgnoreEntry, plan, state *HTTPRequestResourceModel, _ *diag.Diagnostics) bool {
		return setObjectAttribute(planGetter(plan), *stateGetter(state))
	}
}

func makeMapApplier(planGetter, stateGetter mapFieldAccessor) ignoreApplier {
	return func(ctx context.Context, entries []IgnoreEntry, plan, state *HTTPRequestResourceModel, diagnostics *diag.Diagnostics) bool {
		return applyIgnoreForMap(ctx, entries, planGetter(plan), *stateGetter(state), diagnostics)
	}
}

func makeBodyApplier(planGetter, stateGetter stringFieldAccessor) ignoreApplier {
	return func(_ context.Context, entries []IgnoreEntry, plan, state *HTTPRequestResourceModel, diagnostics *diag.Diagnostics) bool {
		return applyIgnoreForBody(entries, planGetter(plan), *stateGetter(state), diagnostics)
	}
}

func applyIgnoreForMap(
	ctx context.Context,
	entries []IgnoreEntry,
	planValue *types.Map,
	stateValue types.Map,
	diagnostics *diag.Diagnostics,
) bool {
	if len(entries) == 0 {
		return false
	}

	// Check for full map ignore first
	if hasFullMapIgnore(entries, planValue, stateValue) {
		return true
	}

	planMap, stateMap := extractMaps(ctx, planValue, stateValue, diagnostics)
	if diagnostics.HasError() {
		return false
	}

	touched := applyMapKeyIgnores(entries, planMap, stateMap)
	if !touched {
		return false
	}

	return updateMapValue(ctx, planValue, planMap, stateValue, diagnostics)
}

func hasFullMapIgnore(entries []IgnoreEntry, planValue *types.Map, stateValue types.Map) bool {
	for _, entry := range entries {
		if len(entry.SubPath) == 0 {
			if planValue.Equal(stateValue) {
				return false
			}
			*planValue = stateValue
			return true
		}
	}
	return false
}

func extractMaps(
	ctx context.Context,
	planValue *types.Map,
	stateValue types.Map,
	diagnostics *diag.Diagnostics,
) (map[string]string, map[string]string) {
	planMap := map[string]string{}
	if !planValue.IsNull() && planValue.Elements() != nil {
		diagnostics.Append(planValue.ElementsAs(ctx, &planMap, true)...)
	}

	stateMap := map[string]string{}
	if !stateValue.IsNull() && stateValue.Elements() != nil {
		diagnostics.Append(stateValue.ElementsAs(ctx, &stateMap, true)...)
	}

	return planMap, stateMap
}

func applyMapKeyIgnores(entries []IgnoreEntry, planMap, stateMap map[string]string) bool {
	touched := false
	for _, entry := range entries {
		if len(entry.SubPath) == 0 {
			continue
		}
		key := entry.SubPath[0]
		stateVal, hasState := stateMap[key]
		if hasState {
			if val, exists := planMap[key]; !exists || val != stateVal {
				planMap[key] = stateVal
				touched = true
			}
			continue
		}
		if _, exists := planMap[key]; exists {
			delete(planMap, key)
			touched = true
		}
	}
	return touched
}

func updateMapValue(
	ctx context.Context,
	planValue *types.Map,
	planMap map[string]string,
	stateValue types.Map,
	diagnostics *diag.Diagnostics,
) bool {
	if len(planMap) == 0 && (stateValue.IsNull() || stateValue.Elements() == nil) {
		*planValue = stateValue
		return true
	}

	newValue, diags := types.MapValueFrom(ctx, types.StringType, planMap)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return false
	}
	*planValue = newValue
	return true
}

func applyIgnoreForBody(
	entries []IgnoreEntry,
	planValue *types.String,
	stateValue types.String,
	diagnostics *diag.Diagnostics,
) bool {
	if len(entries) == 0 {
		return false
	}

	for _, entry := range entries {
		if len(entry.SubPath) == 0 {
			if planValue.Equal(stateValue) {
				return false
			}
			*planValue = stateValue
			return true
		}
	}

	if planValue.IsNull() || planValue.IsUnknown() || stateValue.IsNull() || stateValue.IsUnknown() {
		return false
	}

	planJSON, err := decodeJSONBody(planValue.ValueString())
	if err != nil {
		diagnostics.AddWarning("Unable to parse request_body for ignore_changes", err.Error())
		return false
	}

	stateJSON, err := decodeJSONBody(stateValue.ValueString())
	if err != nil {
		diagnostics.AddWarning("Unable to parse state request_body for ignore_changes", err.Error())
		return false
	}

	target := cloneJSONMap(planJSON)
	touched := false
	for _, entry := range entries {
		if len(entry.SubPath) == 0 {
			continue
		}
		if syncJSONPath(target, stateJSON, entry.SubPath) {
			touched = true
		}
	}

	if !touched {
		return false
	}

	if reflect.DeepEqual(target, stateJSON) {
		if !planValue.Equal(stateValue) {
			*planValue = stateValue
			return true
		}
	}

	return false
}

func decodeJSONBody(raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{}, nil
	}
	if strings.EqualFold(raw, "null") {
		return map[string]interface{}{}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func syncJSONPath(
	target map[string]interface{},
	source map[string]interface{},
	path []string,
) bool {
	if len(path) == 0 {
		return false
	}

	key := path[0]

	if len(path) == 1 {
		return syncJSONLeaf(target, source, key)
	}

	return syncJSONNested(target, source, key, path[1:])
}

func syncJSONLeaf(target, source map[string]interface{}, key string) bool {
	sourceVal, sourceExists := source[key]
	targetVal, targetExists := target[key]

	if sourceExists {
		if !targetExists || !reflect.DeepEqual(targetVal, sourceVal) {
			target[key] = cloneJSONValue(sourceVal)
			return true
		}
		return false
	}

	if targetExists {
		delete(target, key)
		return true
	}
	return false
}

func syncJSONNested(
	target, source map[string]interface{},
	key string,
	remainingPath []string,
) bool {
	sourceChild, sourceExists := source[key]
	targetChild, targetExists := target[key]

	if !sourceExists {
		if targetExists {
			delete(target, key)
			return true
		}
		return false
	}

	sourceMap, isSourceMap := sourceChild.(map[string]interface{})
	if !isSourceMap {
		if !targetExists || !reflect.DeepEqual(targetChild, sourceChild) {
			target[key] = cloneJSONValue(sourceChild)
			return true
		}
		return false
	}

	targetMap := getOrCreateTargetMap(target, targetChild, targetExists, sourceMap, key)
	if targetMap == nil {
		return true // Target was replaced with cloned source
	}

	return syncJSONPath(targetMap, sourceMap, remainingPath)
}

func getOrCreateTargetMap(
	target map[string]interface{},
	targetChild interface{},
	targetExists bool,
	sourceMap map[string]interface{},
	key string,
) map[string]interface{} {
	if targetExists {
		if existing, isMap := targetChild.(map[string]interface{}); isMap {
			return existing
		}
		// Target is not a map, replace with cloned source
		cloned := cloneJSONMap(sourceMap)
		target[key] = cloned
		return nil
	}

	// Create new map
	newMap := make(map[string]interface{})
	target[key] = newMap
	return newMap
}

func cloneJSONValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return cloneJSONMap(typed)
	case []interface{}:
		newSlice := make([]interface{}, len(typed))
		for i, v := range typed {
			newSlice[i] = cloneJSONValue(v)
		}
		return newSlice
	default:
		return typed
	}
}

func cloneJSONMap(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}
	target := make(map[string]interface{}, len(source))
	for k, v := range source {
		target[k] = cloneJSONValue(v)
	}
	return target
}

func setStringAttribute(dest *types.String, state types.String) bool {
	if dest.Equal(state) {
		return false
	}
	*dest = state
	return true
}

func setBoolAttribute(dest *types.Bool, state types.Bool) bool {
	if dest.Equal(state) {
		return false
	}
	*dest = state
	return true
}

func setObjectAttribute(dest *types.Object, state types.Object) bool {
	if dest.Equal(state) {
		return false
	}
	*dest = state
	return true
}
