**Title:** Implement URL field validation using string validators

**Labels:** enhancement, validation

**Body:**
## Description

Add proper string validators for the URL field in the provider configuration schema to ensure URL validity.

## Acceptance Criteria

- [ ] Uncomment and implement the TODO validators line in provider.go:59
- [ ] Use the NewStringNotEmpty validator for URL validation
- [ ] Add comprehensive tests for URL validation
- [ ] Ensure proper error messages for invalid URLs

## Context

This TODO is located in the provider schema definition where URL validation is currently commented out.

**Location:** `internal/provider/provider.go:59`  
**Priority:** Low  
**Component:** Provider schema validation

## Technical Notes

```go
// TODO: Validators: []validator.String{validators.NewStringNotEmpty("url")},
```

The validation should ensure the URL field is not empty and potentially add additional URL format validation.