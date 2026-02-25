# Contributing

Contributions are welcome. By participating, you agree to maintain a respectful and constructive environment.

For coding standards, testing patterns, architecture guidelines, commit conventions, and all
development practices, refer to the **[Development Guide](https://github.com/rios0rios0/guide/wiki)**.

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.10+
- [Make](https://www.gnu.org/software/make/)

## Development Workflow

1. Fork and clone the repository
2. Create a branch: `git checkout -b feat/my-change`
3. Build the provider:
   ```bash
   make build
   ```
4. Make your changes
5. Validate:
   ```bash
   make lint
   make test
   make sast
   ```
6. Regenerate documentation if schemas changed:
   ```bash
   make docs
   ```
7. Update `CHANGELOG.md` under `[Unreleased]`
8. Commit following the [commit conventions](https://github.com/rios0rios0/guide/wiki/Life-Cycle/Git-Flow)
9. Open a pull request against `main`

## Testing Locally

Install the provider to the local plugin directory and use it in a Terraform configuration:

```bash
make install
```

Then reference the local provider in your `main.tf`:

```hcl
terraform {
  required_providers {
    http = {
      source  = "hashicorp-local.com/rios0rios0/http"
      version = "2.2.0"
    }
  }
}
```
