#!/bin/bash

# Script to create GitHub issues from TODO migration
# Run: chmod +x create-github-issues.sh && ./create-github-issues.sh

echo "Creating 9 GitHub issues from TODO migration..."

# Issue 1: Delete functionality (High Priority)
gh issue create \
  --title "Implement delete functionality for HTTP request resources" \
  --body "## Description

Currently, the HTTP request resource does not implement proper delete functionality. When a resource is destroyed in Terraform, it should be removed from the state properly.

## Acceptance Criteria

- [ ] Implement the Delete method in HTTPRequestResource
- [ ] Ensure proper state cleanup when resources are destroyed
- [ ] Add tests to verify delete functionality works correctly
- [ ] Update documentation if needed

## Context

This was identified in the original TODO list as a high-priority feature for proper resource lifecycle management.

**Location:** Originally mentioned in README.md TODO section  
**Priority:** High  
**Component:** Resource lifecycle management

## Technical Notes

The Delete method should be implemented in the \`HTTPRequestResource\` struct in \`internal/provider/resource_http_request.go\`. Consider whether any cleanup of external resources is needed or if this is purely state management." \
  --label "enhancement,high-priority"

# Issue 2: UUID validation
gh issue create \
  --title "Add UUID validation for resource import operations" \
  --body "## Description

The resource import functionality needs proper UUID validation to ensure only valid UUIDs are accepted during import operations.

## Acceptance Criteria

- [ ] Add UUID validation for import operations
- [ ] Provide clear error messages for invalid UUIDs
- [ ] Add tests to verify UUID validation works correctly
- [ ] Update import documentation with UUID format requirements

## Context

This was identified in the original TODO list as a medium-priority validation enhancement.

**Location:** Originally mentioned in README.md TODO section  
**Priority:** Medium  
**Component:** Resource import and validation

## Technical Notes

The import ID format includes a UUID component: \`<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>\`. Validation should ensure the UUID part follows proper UUID format (RFC 4122)." \
  --label "enhancement,validation"

# Issue 3: URL validation
gh issue create \
  --title "Enhanced URL validation at provider and resource levels" \
  --body "## Description

Update \`ValidateConfig\` inside \`provider.go\` to catch when the URL is set on the resource as well, providing comprehensive URL validation across the provider.

## Acceptance Criteria

- [ ] Update ValidateConfig to handle URL validation at both provider and resource levels
- [ ] Ensure validation catches conflicting or invalid URL configurations
- [ ] Add comprehensive tests for URL validation scenarios
- [ ] Update documentation with URL configuration examples

## Context

This was identified in the original TODO list to improve configuration validation.

**Location:** Originally mentioned in README.md TODO section  
**Priority:** Medium  
**Component:** Provider configuration validation

## Technical Notes

The validation should check for URL conflicts between provider-level and resource-level URL settings, and ensure proper URL format validation throughout the configuration." \
  --label "enhancement,validation"

# Issue 4: URL field validators
gh issue create \
  --title "Implement URL field validation using string validators" \
  --body "## Description

Add proper string validators for the URL field in the provider configuration schema to ensure URL validity.

## Acceptance Criteria

- [ ] Uncomment and implement the TODO validators line in provider.go:59
- [ ] Use the NewStringNotEmpty validator for URL validation
- [ ] Add comprehensive tests for URL validation
- [ ] Ensure proper error messages for invalid URLs

## Context

This TODO is located in the provider schema definition where URL validation is currently commented out.

**Location:** \`internal/provider/provider.go:59\`  
**Priority:** Low  
**Component:** Provider schema validation

## Technical Notes

\`\`\`go
// TODO: Validators: []validator.String{validators.NewStringNotEmpty(\"url\")},
\`\`\`

The validation should ensure the URL field is not empty and potentially add additional URL format validation." \
  --label "enhancement,validation"

# Issue 5: JSON config parsing
gh issue create \
  --title "Evaluate JSON-based configuration parsing optimization" \
  --body "## Description

Evaluate whether it's worth using JSON marshaling/unmarshaling instead of getting configuration values individually for better performance and cleaner code.

## Acceptance Criteria

- [ ] Benchmark current approach vs JSON-based parsing
- [ ] Evaluate code maintainability implications  
- [ ] Consider error handling differences
- [ ] Make decision on implementation approach
- [ ] Document the decision and reasoning

## Context

This TODO questions whether the current value-by-value approach should be replaced with JSON serialization.

**Location:** \`internal/provider/provider.go:205\`  
**Priority:** Low  
**Component:** Configuration parsing

## Technical Notes

\`\`\`go
/* TODO: is it worth to use JSON instead of getting value per value?
var source Configuration
jsonData, _ := json.Marshal(model)
_ = json.Unmarshal(jsonData, &source) */
\`\`\`

Consider performance implications, error handling, and code readability when making this decision." \
  --label "enhancement,performance"

# Issue 6: Import documentation
gh issue create \
  --title "Improve import functionality documentation with examples" \
  --body "## Description

Add comprehensive documentation for the import functionality, including practical examples of how to import resources with the required ID format.

## Acceptance Criteria

- [ ] Document the import ID format with clear examples
- [ ] Explain the UUID and Base64 encoding components
- [ ] Provide step-by-step import examples
- [ ] Add troubleshooting section for common import issues
- [ ] Update provider documentation

## Context

This TODO highlights the need for better documentation around the import functionality.

**Location:** \`internal/provider/resource_http_request.go:154\`  
**Priority:** Low  
**Component:** Documentation

## Technical Notes

\`\`\`go
// TODO: how to document the import of ths ID with examples?
\`\`\`

The import ID format is: \`<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>\`. Documentation should explain how to construct this ID and provide practical examples." \
  --label "documentation,enhancement"

# Issue 7: Read method (High Priority)
gh issue create \
  --title "Implement Read method for drift detection and state comparison" \
  --body "## Description

Implement the Read method to enable drift detection and comparison between the original source and the Terraform state.

## Acceptance Criteria

- [ ] Implement the Read method in HTTPRequestResource
- [ ] Enable drift detection capabilities
- [ ] Compare current state with original source when possible
- [ ] Add comprehensive tests for Read functionality
- [ ] Update documentation with drift detection behavior

## Context

This is a high-priority TODO for proper Terraform resource lifecycle management and state synchronization.

**Location:** \`internal/provider/resource_http_request.go:335\`  
**Priority:** High  
**Component:** Resource lifecycle management

## Technical Notes

\`\`\`go
// TODO: should be implemented to be able to read the original source and compare with the TF state
\`\`\`

The Read method is crucial for Terraform's refresh and plan operations. It should check if the remote resource still exists and matches the expected state." \
  --label "enhancement,high-priority"

# Issue 8: Validator implementation
gh issue create \
  --title "Review and improve Terraform validator implementation" \
  --body "## Description

Review the current implementation of Terraform validators to ensure they follow best practices and provide proper error handling.

## Acceptance Criteria

- [ ] Review the current validator implementation approach
- [ ] Ensure proper error handling and diagnostic reporting
- [ ] Verify this is the correct way to implement TF validators
- [ ] Add comprehensive tests for validator behavior
- [ ] Update implementation if needed based on best practices

## Context

This TODO questions whether the current validator implementation follows Terraform plugin framework best practices.

**Location:** \`internal/infrastructure/validators/string_not_empty.go:37\`  
**Priority:** Low  
**Component:** Validation infrastructure

## Technical Notes

\`\`\`go
// TODO: is this the correct way to check for TF validators?
// this is just appending the error to the diagnostics and showing them at the end
\`\`\`

Review Terraform plugin framework documentation and ensure the validator implementation follows recommended patterns." \
  --label "enhancement,validation"

# Issue 9: Test builder refactoring
gh issue create \
  --title "Refactor test builder pattern to fluent API" \
  --body "## Description

Refactor the test infrastructure to use a fluent API builder pattern instead of the current approach for better test readability and maintainability.

## Acceptance Criteria

- [ ] Implement fluent API builder pattern for provider value construction
- [ ] Replace current map-based approach with chainable builder methods
- [ ] Update all existing tests to use the new builder pattern
- [ ] Ensure test coverage is maintained or improved
- [ ] Document the new builder pattern usage

## Context

This TODO suggests improving test infrastructure by implementing a fluent builder pattern.

**Location:** \`test/infrastructure/builders/provider_value_builder.go:73\` and \`90\`  
**Priority:** Low  
**Component:** Test infrastructure

## Technical Notes

\`\`\`go
// TODO: this should be used to produce the builder above
// builders.NewProviderValueBuilder().
//   WithURL(\"https://jsonplaceholder.typicode.com\").
//   WithIgnoreTLS(false).
//   WithUsername(\"user\").
//   WithPassword(\"pass\").
\`\`\`

The suggested fluent API would improve test readability and provide better IntelliSense support." \
  --label "enhancement,testing"

echo "✅ All 9 GitHub issues created successfully!"
echo ""
echo "Issues created:"
echo "🔴 High Priority: Delete functionality, Read method"
echo "🟡 Medium Priority: UUID validation, URL validation" 
echo "🟢 Low Priority: URL validators, JSON config, import docs, validator review, test builder"