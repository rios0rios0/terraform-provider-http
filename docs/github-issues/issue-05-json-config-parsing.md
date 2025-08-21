**Title:** Evaluate JSON-based configuration parsing optimization

**Labels:** enhancement, performance

**Body:**
## Description

Evaluate whether it's worth using JSON marshaling/unmarshaling instead of getting configuration values individually for better performance and cleaner code.

## Acceptance Criteria

- [ ] Benchmark current approach vs JSON-based parsing
- [ ] Evaluate code maintainability implications  
- [ ] Consider error handling differences
- [ ] Make decision on implementation approach
- [ ] Document the decision and reasoning

## Context

This TODO questions whether the current value-by-value approach should be replaced with JSON serialization.

**Location:** `internal/provider/provider.go:205`  
**Priority:** Low  
**Component:** Configuration parsing

## Technical Notes

```go
/* TODO: is it worth to use JSON instead of getting value per value?
var source Configuration
jsonData, _ := json.Marshal(model)
_ = json.Unmarshal(jsonData, &source) */
```

Consider performance implications, error handling, and code readability when making this decision.