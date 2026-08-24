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
- Delete-operation attributes (`is_delete_enabled`, `delete_method`, `delete_path`, `delete_headers`, `delete_request_body`) are WriteOnly (Terraform 1.11+). They are not persisted in Terraform state; values are stored in provider private state.
- Refresh-control attributes (`is_refresh_enabled`, `refresh_path`) are ordinary state-stored arguments, **not** WriteOnly — the `Read` RPC receives no configuration, so a write-only value would be unavailable exactly when refresh needs it.
- Schema version is 3. Upgraders are registered for versions 0, 1 and 2, and the framework hands every one of them a response state built from the *current* schema — so a new attribute must be added to **all** of them, as a TYPED null (`types.MapNull(types.StringType)`, not a zero value), or the next plan fails with `Value Conversion Error ... MISSING TYPE`.
- Import must never cause a replacement. `ImportState` records the attributes an identifier left unspecified under the `import_adopt` private-state key; the conditional `RequiresReplaceIf` modifiers in `internal/provider/import_adopt.go` read that key and suppress replacement for exactly those attributes, and `Update` settles them from configuration without issuing a request, then clears the key. Attribute-level plan modifiers run *before* `ModifyPlan` and `resp.RequiresReplace` is only ever appended to, so `ModifyPlan` cannot undo a replacement — the private-state flag is the only lever.
- Import identifier decoding lives in `internal/provider/import_payload.go`. Forms are dispatched on the first character so none can be read as another; `import_payload.go` also renders the `import_id` attribute, which deliberately omits `basic_auth`. `import_read_path` is import-only — not a schema attribute (see the struct in `import_payload.go`) — and opts a resource created with an unsafe method into a GET against the object it made rather than replaying the create.
- The provider-level `headers` map is merged BEFORE each resource's own `headers` at the single point every request is built, so it covers create, read, refresh, destroy, and the import read; a resource naming the same header overrides it (per-header, not all-or-nothing). It is `Sensitive` and has no env-var equivalent. Because the import read is built from the identifier alone (which omits `headers`), a provider-level header is the only way to authenticate an import against a header-auth API.
- `Makefile` targets delegate to scripts in the external `rios0rios0/pipelines` repo (cloned to `~/Development/github.com/rios0rios0/pipelines`).
- CI runs `rios0rios0/pipelines/.github/workflows/go-binary.yaml@main`. Releases use GoReleaser with GPG signing on `v*` tags.
- Tests hit `jsonplaceholder.typicode.com` for integration testing. Acceptance tests need `TF_ACC_PROVIDER_NAMESPACE=rios0rios0`.

## Requirements

- Go 1.27.0+
- Terraform 1.11+
