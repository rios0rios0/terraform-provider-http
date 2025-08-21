**Title:** Add UUID validation for resource import operations

**Labels:** enhancement, validation

**Body:**
## Description

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

The import ID format includes a UUID component: `<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>`. Validation should ensure the UUID part follows proper UUID format (RFC 4122).