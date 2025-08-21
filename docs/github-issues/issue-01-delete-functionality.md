**Title:** Implement delete functionality for HTTP request resources

**Labels:** enhancement, high-priority

**Body:**
## Description

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

The Delete method should be implemented in the `HTTPRequestResource` struct in `internal/provider/resource_http_request.go`. Consider whether any cleanup of external resources is needed or if this is purely state management.