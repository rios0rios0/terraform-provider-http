**Title:** Enhanced URL validation at provider and resource levels

**Labels:** enhancement, validation

**Body:**
## Description

Update `ValidateConfig` inside `provider.go` to catch when the URL is set on the resource as well, providing comprehensive URL validation across the provider.

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

The validation should check for URL conflicts between provider-level and resource-level URL settings, and ensure proper URL format validation throughout the configuration.