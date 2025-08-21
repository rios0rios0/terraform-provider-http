**Title:** Improve import functionality documentation with examples

**Labels:** documentation, enhancement

**Body:**
## Description

Add comprehensive documentation for the import functionality, including practical examples of how to import resources with the required ID format.

## Acceptance Criteria

- [ ] Document the import ID format with clear examples
- [ ] Explain the UUID and Base64 encoding components
- [ ] Provide step-by-step import examples
- [ ] Add troubleshooting section for common import issues
- [ ] Update provider documentation

## Context

This TODO highlights the need for better documentation around the import functionality.

**Location:** `internal/provider/resource_http_request.go:154`  
**Priority:** Low  
**Component:** Documentation

## Technical Notes

```go
// TODO: how to document the import of ths ID with examples?
```

The import ID format is: `<RANDOM UUID>/<PARAMETERS ENCODED IN BASE64>`. Documentation should explain how to construct this ID and provide practical examples.