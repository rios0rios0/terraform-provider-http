#!/bin/bash

# Alternative script using curl and GitHub API
# Set your GitHub personal access token first:
# export GITHUB_TOKEN="your_token_here"

if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ Please set GITHUB_TOKEN environment variable"
    echo "Get a token from: https://github.com/settings/tokens/new"
    echo "Then run: export GITHUB_TOKEN='your_token_here'"
    exit 1
fi

REPO="rios0rios0/terraform-provider-http"
API_URL="https://api.github.com/repos/$REPO/issues"

echo "Creating 9 GitHub issues from TODO migration using GitHub API..."

# Issue 1: Delete functionality (High Priority)
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Implement delete functionality for HTTP request resources",
    "body": "## Description\n\nCurrently, the HTTP request resource does not implement proper delete functionality. When a resource is destroyed in Terraform, it should be removed from the state properly.\n\n## Acceptance Criteria\n\n- [ ] Implement the Delete method in HTTPRequestResource\n- [ ] Ensure proper state cleanup when resources are destroyed\n- [ ] Add tests to verify delete functionality works correctly\n- [ ] Update documentation if needed\n\n## Context\n\nThis was identified in the original TODO list as a high-priority feature for proper resource lifecycle management.\n\n**Location:** Originally mentioned in README.md TODO section\n**Priority:** High\n**Component:** Resource lifecycle management\n\n## Technical Notes\n\nThe Delete method should be implemented in the `HTTPRequestResource` struct in `internal/provider/resource_http_request.go`. Consider whether any cleanup of external resources is needed or if this is purely state management.",
    "labels": ["enhancement", "high-priority"]
  }'

echo -e "\n✅ Issue 1 created: Delete functionality"

# Issue 2: UUID validation
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Add UUID validation for resource import operations",
    "body": "## Description\n\nThe resource import functionality needs proper UUID validation to ensure only valid UUIDs are accepted during import operations.\n\n## Acceptance Criteria\n\n- [ ] Add UUID validation for import operations\n- [ ] Provide clear error messages for invalid UUIDs\n- [ ] Add tests to verify UUID validation works correctly\n- [ ] Update import documentation with UUID format requirements\n\n## Context\n\nThis was identified in the original TODO list as a medium-priority validation enhancement.\n\n**Location:** Originally mentioned in README.md TODO section\n**Priority:** Medium\n**Component:** Resource import and validation\n\n## Technical Notes\n\nThe import ID format includes a UUID component: `<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>`. Validation should ensure the UUID part follows proper UUID format (RFC 4122).",
    "labels": ["enhancement", "validation"]
  }'

echo -e "\n✅ Issue 2 created: UUID validation"

# Issue 3: URL validation 
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Enhanced URL validation at provider and resource levels",
    "body": "## Description\n\nUpdate `ValidateConfig` inside `provider.go` to catch when the URL is set on the resource as well, providing comprehensive URL validation across the provider.\n\n## Acceptance Criteria\n\n- [ ] Update ValidateConfig to handle URL validation at both provider and resource levels\n- [ ] Ensure validation catches conflicting or invalid URL configurations\n- [ ] Add comprehensive tests for URL validation scenarios\n- [ ] Update documentation with URL configuration examples\n\n## Context\n\nThis was identified in the original TODO list to improve configuration validation.\n\n**Location:** Originally mentioned in README.md TODO section\n**Priority:** Medium\n**Component:** Provider configuration validation\n\n## Technical Notes\n\nThe validation should check for URL conflicts between provider-level and resource-level URL settings, and ensure proper URL format validation throughout the configuration.",
    "labels": ["enhancement", "validation"]
  }'

echo -e "\n✅ Issue 3 created: URL validation"

# Issue 4: URL field validators
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Implement URL field validation using string validators",
    "body": "## Description\n\nAdd proper string validators for the URL field in the provider configuration schema to ensure URL validity.\n\n## Acceptance Criteria\n\n- [ ] Uncomment and implement the TODO validators line in provider.go:59\n- [ ] Use the NewStringNotEmpty validator for URL validation\n- [ ] Add comprehensive tests for URL validation\n- [ ] Ensure proper error messages for invalid URLs\n\n## Context\n\nThis TODO is located in the provider schema definition where URL validation is currently commented out.\n\n**Location:** `internal/provider/provider.go:59`\n**Priority:** Low\n**Component:** Provider schema validation\n\n## Technical Notes\n\n```go\n// TODO: Validators: []validator.String{validators.NewStringNotEmpty(\"url\")},\n```\n\nThe validation should ensure the URL field is not empty and potentially add additional URL format validation.",
    "labels": ["enhancement", "validation"]
  }'

echo -e "\n✅ Issue 4 created: URL field validators"

# Issue 5: JSON config parsing
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Evaluate JSON-based configuration parsing optimization",
    "body": "## Description\n\nEvaluate whether it'"'"'s worth using JSON marshaling/unmarshaling instead of getting configuration values individually for better performance and cleaner code.\n\n## Acceptance Criteria\n\n- [ ] Benchmark current approach vs JSON-based parsing\n- [ ] Evaluate code maintainability implications\n- [ ] Consider error handling differences\n- [ ] Make decision on implementation approach\n- [ ] Document the decision and reasoning\n\n## Context\n\nThis TODO questions whether the current value-by-value approach should be replaced with JSON serialization.\n\n**Location:** `internal/provider/provider.go:205`\n**Priority:** Low\n**Component:** Configuration parsing\n\n## Technical Notes\n\n```go\n/* TODO: is it worth to use JSON instead of getting value per value?\nvar source Configuration\njsonData, _ := json.Marshal(model)\n_ = json.Unmarshal(jsonData, &source) */\n```\n\nConsider performance implications, error handling, and code readability when making this decision.",
    "labels": ["enhancement", "performance"]
  }'

echo -e "\n✅ Issue 5 created: JSON config parsing"

# Issue 6: Import documentation
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Improve import functionality documentation with examples",
    "body": "## Description\n\nAdd comprehensive documentation for the import functionality, including practical examples of how to import resources with the required ID format.\n\n## Acceptance Criteria\n\n- [ ] Document the import ID format with clear examples\n- [ ] Explain the UUID and Base64 encoding components\n- [ ] Provide step-by-step import examples\n- [ ] Add troubleshooting section for common import issues\n- [ ] Update provider documentation\n\n## Context\n\nThis TODO highlights the need for better documentation around the import functionality.\n\n**Location:** `internal/provider/resource_http_request.go:154`\n**Priority:** Low\n**Component:** Documentation\n\n## Technical Notes\n\n```go\n// TODO: how to document the import of ths ID with examples?\n```\n\nThe import ID format is: `<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>`. Documentation should explain how to construct this ID and provide practical examples.",
    "labels": ["documentation", "enhancement"]
  }'

echo -e "\n✅ Issue 6 created: Import documentation"

# Issue 7: Read method (High Priority)
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Implement Read method for drift detection and state comparison",
    "body": "## Description\n\nImplement the Read method to enable drift detection and comparison between the original source and the Terraform state.\n\n## Acceptance Criteria\n\n- [ ] Implement the Read method in HTTPRequestResource\n- [ ] Enable drift detection capabilities\n- [ ] Compare current state with original source when possible\n- [ ] Add comprehensive tests for Read functionality\n- [ ] Update documentation with drift detection behavior\n\n## Context\n\nThis is a high-priority TODO for proper Terraform resource lifecycle management and state synchronization.\n\n**Location:** `internal/provider/resource_http_request.go:335`\n**Priority:** High\n**Component:** Resource lifecycle management\n\n## Technical Notes\n\n```go\n// TODO: should be implemented to be able to read the original source and compare with the TF state\n```\n\nThe Read method is crucial for Terraform'"'"'s refresh and plan operations. It should check if the remote resource still exists and matches the expected state.",
    "labels": ["enhancement", "high-priority"]
  }'

echo -e "\n✅ Issue 7 created: Read method"

# Issue 8: Validator implementation
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Review and improve Terraform validator implementation",
    "body": "## Description\n\nReview the current implementation of Terraform validators to ensure they follow best practices and provide proper error handling.\n\n## Acceptance Criteria\n\n- [ ] Review the current validator implementation approach\n- [ ] Ensure proper error handling and diagnostic reporting\n- [ ] Verify this is the correct way to implement TF validators\n- [ ] Add comprehensive tests for validator behavior\n- [ ] Update implementation if needed based on best practices\n\n## Context\n\nThis TODO questions whether the current validator implementation follows Terraform plugin framework best practices.\n\n**Location:** `internal/infrastructure/validators/string_not_empty.go:37`\n**Priority:** Low\n**Component:** Validation infrastructure\n\n## Technical Notes\n\n```go\n// TODO: is this the correct way to check for TF validators?\n// this is just appending the error to the diagnostics and showing them at the end\n```\n\nReview Terraform plugin framework documentation and ensure the validator implementation follows recommended patterns.",
    "labels": ["enhancement", "validation"]
  }'

echo -e "\n✅ Issue 8 created: Validator implementation"

# Issue 9: Test builder refactoring
curl -X POST "$API_URL" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d '{
    "title": "Refactor test builder pattern to fluent API",
    "body": "## Description\n\nRefactor the test infrastructure to use a fluent API builder pattern instead of the current approach for better test readability and maintainability.\n\n## Acceptance Criteria\n\n- [ ] Implement fluent API builder pattern for provider value construction\n- [ ] Replace current map-based approach with chainable builder methods\n- [ ] Update all existing tests to use the new builder pattern\n- [ ] Ensure test coverage is maintained or improved\n- [ ] Document the new builder pattern usage\n\n## Context\n\nThis TODO suggests improving test infrastructure by implementing a fluent builder pattern.\n\n**Location:** `test/infrastructure/builders/provider_value_builder.go:73` and `90`\n**Priority:** Low\n**Component:** Test infrastructure\n\n## Technical Notes\n\n```go\n// TODO: this should be used to produce the builder above\n// builders.NewProviderValueBuilder().\n//   WithURL(\"https://jsonplaceholder.typicode.com\").\n//   WithIgnoreTLS(false).\n//   WithUsername(\"user\").\n//   WithPassword(\"pass\").\n```\n\nThe suggested fluent API would improve test readability and provide better IntelliSense support.",
    "labels": ["enhancement", "testing"]
  }'

echo -e "\n✅ Issue 9 created: Test builder refactoring"

echo -e "\n🎉 All 9 GitHub issues created successfully!"
echo ""
echo "Issues created:"
echo "🔴 High Priority: Delete functionality, Read method"
echo "🟡 Medium Priority: UUID validation, URL validation" 
echo "🟢 Low Priority: URL validators, JSON config, import docs, validator review, test builder"