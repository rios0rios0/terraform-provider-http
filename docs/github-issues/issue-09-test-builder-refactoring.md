**Title:** Refactor test builder pattern to fluent API

**Labels:** enhancement, testing

**Body:**
## Description

Refactor the test infrastructure to use a fluent API builder pattern instead of the current approach for better test readability and maintainability.

## Acceptance Criteria

- [ ] Implement fluent API builder pattern for provider value construction
- [ ] Replace current map-based approach with chainable builder methods
- [ ] Update all existing tests to use the new builder pattern
- [ ] Ensure test coverage is maintained or improved
- [ ] Document the new builder pattern usage

## Context

This TODO suggests improving test infrastructure by implementing a fluent builder pattern.

**Location:** `test/infrastructure/builders/provider_value_builder.go:73` and `90`  
**Priority:** Low  
**Component:** Test infrastructure

## Technical Notes

```go
// TODO: this should be used to produce the builder above
// builders.NewProviderValueBuilder().
//   WithURL("https://jsonplaceholder.typicode.com").
//   WithIgnoreTLS(false).
//   WithUsername("user").
//   WithPassword("pass").
```

The suggested fluent API would improve test readability and provide better IntelliSense support.