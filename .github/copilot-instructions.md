# Terraform Provider for HTTP Requests

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Prerequisites and Setup
- Install Go 1.23.4+ (currently using Go 1.24.6)
- Install Terraform 1.10.4+ (tested with 1.13.0)
- Install make for build automation

### Essential Build and Test Commands
- `make build` -- compiles the provider binary. Takes <1 second. NEVER CANCEL.
- `make install` -- builds and installs provider locally for testing. Takes ~1 second. 
- `make test` -- runs full test suite with coverage. Takes ~4 seconds. NEVER CANCEL. Set timeout to 30+ seconds.
- `make docs` -- generates provider documentation. Takes ~2 seconds. NEVER CANCEL.

### Linting and Code Quality
- `go fmt ./...` -- formats Go code
- `go vet ./...` -- runs Go static analysis
- Install golangci-lint: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /tmp v1.62.2`
- `/tmp/golangci-lint run ./...` -- runs comprehensive linting. Takes ~1 minute. NEVER CANCEL. Set timeout to 120+ seconds.

### Manual Provider Testing
- Always run `make install` before testing provider changes
- Create test Terraform configuration in `/tmp/tf-test/main.tf`:
```hcl
terraform {
  required_providers {
    http = {
      source = "hashicorp-local.com/rios0rios0/http"
      version = "2.1.0"
    }
  }
}

provider "http" {
  url = "https://jsonplaceholder.typicode.com"
  ignore_tls = true
}

resource "http_request" "test_request" {
  method = "GET"
  path   = "/posts/1"
  headers = {
    "Accept" = "application/json"
  }
  is_response_body_json   = true
  response_body_id_filter = "$.id"
}
```
- Run: `cd /tmp/tf-test && terraform init && terraform plan`
- For complete validation: `terraform apply -auto-approve` (requires network access)

## Validation Requirements

### Always Test These Scenarios After Changes
1. **Build and Install**: `make build && make install` -- must complete without errors
2. **Test Suite**: `make test` -- all tests must pass with coverage report
3. **Provider Installation**: Local provider must install to `~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/2.1.0/linux_amd64/`
4. **Terraform Integration**: `terraform init` and `terraform plan` must work with local provider
5. **Documentation**: `make docs` must generate clean documentation in `docs/` directory

### Code Quality Checks
- Run `go fmt ./...` and `go vet ./...` before committing
- Install and run golangci-lint for comprehensive static analysis
- All tests must maintain or improve coverage (currently 45.8% total coverage)

## Key Codebase Navigation

### Critical Files and Directories
- `main.go` -- provider entry point and version definition
- `internal/provider/` -- core provider implementation
  - `provider.go` -- provider configuration and setup
  - `resource_http_request.go` -- main HTTP request resource
  - `*_test.go` -- comprehensive test files with unit and integration tests
- `internal/domain/` -- domain entities and business logic
- `test/infrastructure/builders/` -- test utilities for building configurations
- `examples/` -- working example configurations for testing
- `Makefile` -- build automation with all essential targets

### Important Provider Features
- HTTP methods: GET, POST, PUT, DELETE with full header support
- JSON response parsing with JSONPath filtering (`response_body_id_filter`)
- Basic authentication and TLS options
- Query parameters and request body support
- State management with response storage
- Delete operations with path resolution

### Version and Configuration
- Current version: 2.1.0 (defined in `main.go`)
- Provider address: `registry.terraform.io/rios0rios0/http`
- Local testing address: `hashicorp-local.com/rios0rios0/http`

## Common Issues and Solutions

### Build Issues
- If `make docs` fails with PATH error: Run `export GOBIN=$PWD/bin && export PATH=$GOBIN:$PATH` first
- Missing dependencies: Run `go mod download` and `go mod tidy`

### Test Issues  
- Network connectivity issues in tests: Tests use jsonplaceholder.typicode.com for integration testing
- Provider namespace errors: Ensure `TF_ACC_PROVIDER_NAMESPACE=rios0rios0` is set for acceptance tests

### Provider Installation Issues
- Local provider not found: Ensure `make install` completed successfully
- Version mismatches: Check version in `main.go` matches version in test configurations

## External Dependencies
- Uses external CI pipeline: `rios0rios0/pipelines/.github/workflows/go-binary.yaml@main`
- Test infrastructure requires access to jsonplaceholder.typicode.com for integration tests
- Documentation generation requires `terraform` binary in PATH

Always build and exercise your changes with the validation scenarios above before considering work complete.