# Contributing

Contributions are welcome. By participating, you agree to maintain a respectful and constructive environment.

For coding standards, testing patterns, architecture guidelines, commit conventions, and all
development practices, refer to the **[Development Guide](https://github.com/rios0rios0/guide/wiki)**.

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+
- [GNU Make](https://www.gnu.org/software/make/)
- [Terraform](https://developer.hashicorp.com/terraform/install) (for running acceptance tests)

## Development Workflow

1. Fork and clone the repository
2. Create a branch: `git checkout -b feat/my-change`
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Build the provider binary:
   ```bash
   make build
   ```
5. Install the provider locally for testing:
   ```bash
   make install
   ```
6. Run linting:
   ```bash
   make lint
   ```
7. Run tests:
   ```bash
   make test
   ```
8. Run security analysis (SAST):
   ```bash
   make all
   ```
9. Generate provider documentation:
   ```bash
   make docs
   ```
10. Commit following the [commit conventions](https://github.com/rios0rios0/guide/wiki/Life-Cycle/Git-Flow)
11. Open a pull request against `main`
