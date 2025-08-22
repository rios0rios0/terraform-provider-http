**Title:** Review and improve Terraform validator implementation

**Labels:** enhancement, validation

**Body:**
## Description

Review the current implementation of Terraform validators to ensure they follow best practices and provide proper error handling.

## Acceptance Criteria

- [ ] Review the current validator implementation approach
- [ ] Ensure proper error handling and diagnostic reporting
- [ ] Verify this is the correct way to implement TF validators
- [ ] Add comprehensive tests for validator behavior
- [ ] Update implementation if needed based on best practices

## Context

This TODO questions whether the current validator implementation follows Terraform plugin framework best practices.

**Location:** `internal/infrastructure/validators/string_not_empty.go:37`  
**Priority:** Low  
**Component:** Validation infrastructure

## Technical Notes

```go
// TODO: is this the correct way to check for TF validators?
// this is just appending the error to the diagnostics and showing them at the end
```

Review Terraform plugin framework documentation and ensure the validator implementation follows recommended patterns.