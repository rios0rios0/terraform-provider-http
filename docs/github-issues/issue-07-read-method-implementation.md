**Title:** Implement Read method for drift detection and state comparison

**Labels:** enhancement, high-priority

**Body:**
## Description

Implement the Read method to enable drift detection and comparison between the original source and the Terraform state.

## Acceptance Criteria

- [ ] Implement the Read method in HTTPRequestResource
- [ ] Enable drift detection capabilities
- [ ] Compare current state with original source when possible
- [ ] Add comprehensive tests for Read functionality
- [ ] Update documentation with drift detection behavior

## Context

This is a high-priority TODO for proper Terraform resource lifecycle management and state synchronization.

**Location:** `internal/provider/resource_http_request.go:335`  
**Priority:** High  
**Component:** Resource lifecycle management

## Technical Notes

```go
// TODO: should be implemented to be able to read the original source and compare with the TF state
```

The Read method is crucial for Terraform's refresh and plan operations. It should check if the remote resource still exists and matches the expected state.