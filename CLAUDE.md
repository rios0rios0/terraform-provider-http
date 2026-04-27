# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

- `make build` — compiles the provider binary with ldflags version injection
- `make install` — builds and installs to `~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/`
- `make test` — runs full test suite with coverage (uses external script from `rios0rios0/pipelines`)
- `make lint` — runs golangci-lint via the pipelines project's shared config
- `make lint-fix` — lint with auto-fix
- `make docs` — generates Terraform plugin docs (requires `terraform` in PATH; may need `export GOBIN=$PWD/bin && export PATH=$GOBIN:$PATH`)
- `make semgrep` / `make gitleaks` — security scanning via pipelines scripts

VERSION is auto-detected from the latest git tag (`git describe --tags --abbrev=0`), falling back to `dev`.

## Architecture

Terraform provider using the Plugin Framework (not the older SDK). Follows a DDD-inspired layout:

- `main.go` — entry point; version injected via `-ldflags`
- `internal/provider/` — Terraform resource and provider schemas, CRUD logic, validators
- `internal/domain/entities/` — domain models (`Configuration`, `InternalContext`)
- `internal/infrastructure/helpers/` — HTTP client, mapper, and resource helper utilities
- `internal/infrastructure/validators/` — custom schema validators (e.g. `StringNotEmpty`)
- `test/infrastructure/builders/` — test helpers for constructing provider/resource configs
- `tools/tools.go` — tool dependencies pinned for `go generate` (e.g. `tfplugindocs`)

## Key Conventions

- Always update `CHANGELOG.md` under `[Unreleased]` when making changes.
- Delete-operation attributes (`is_delete_enabled`, `delete_method`, `delete_path`, `delete_headers`, `delete_request_body`) are WriteOnly (Terraform 1.11+). They are not persisted in Terraform state; values are stored in provider private state. Schema version is 1.
- `Makefile` targets delegate to scripts in the external `rios0rios0/pipelines` repo (cloned to `~/Development/github.com/rios0rios0/pipelines`).
- CI runs `rios0rios0/pipelines/.github/workflows/go-binary.yaml@main`. Releases use GoReleaser with GPG signing on `v*` tags.
- Tests hit `jsonplaceholder.typicode.com` for integration testing. Acceptance tests need `TF_ACC_PROVIDER_NAMESPACE=rios0rios0`.

## Requirements

- Go 1.26.2+
- Terraform 1.11+
