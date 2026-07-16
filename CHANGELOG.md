# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

When a new release is proposed:

1. Create a new branch `bump/x.x.x` (this isn't a long-lived branch!!!);
2. The Unreleased section on `CHANGELOG.md` gets a version number and date;
3. Verify the `VERSION` in the `Makefile` if applicable (it auto-detects from git tags by default);
4. Open a Pull Request with the bump version changes targeting the `main` branch;
5. When the Pull Request is merged, a new Git tag must be created using [GitHub environment](https://github.com/rios0rios0/terraform-provider-http/tags).

Releases to productive environments should run from a tagged version.
Exceptions are acceptable depending on the circumstances (critical bug fixes that can be cherry-picked, etc.).

## [Unreleased]

## [3.3.5] - 2026-07-16

### Changed

- changed the Go module dependencies to their latest versions

## [3.3.4] - 2026-07-14

### Changed

- changed the Go module dependencies to their latest versions

## [3.3.3] - 2026-07-13

### Changed

- changed the Go module dependencies to their latest versions
- refreshed `CLAUDE.md` and `.github/copilot-instructions.md` to reference the Go `1.26.5` version from `go.mod`

## [3.3.2] - 2026-07-10

### Changed

- changed the Go module dependencies to their latest versions
- changed the Go version to `1.26.5` and updated all module dependencies

## [3.3.1] - 2026-07-02

### Changed

- changed the Go module dependencies to their latest versions

### Security

- replaced `secrets: inherit` with an explicit `CLAUDE_CODE_OAUTH_TOKEN` pass-through in the Claude Code workflows to satisfy the `secrets-inherit` least-privilege check

## [3.3.0] - 2026-06-29

### Added

- added a `request_timeout_ms` argument and a `retry` block (`attempts`, `min_delay_ms`, `max_delay_ms`) to both the provider and the `http_request` resource, mirroring the upstream `hashicorp/http` provider. Configured at the provider level they apply to every request; the matching resource-level arguments override them. `request_timeout_ms` bounds each individual attempt -- an unset or `0` value preserves the previous behavior of waiting indefinitely -- and retries are attempted on connection errors and on 5xx (except 501) responses using an exponential backoff bounded by `min_delay_ms` and `max_delay_ms`. This addresses requests hanging indefinitely against a slow or unreachable endpoint, since the underlying HTTP client previously set no timeout and performed no retries

### Changed

- changed the Go module dependencies to their latest versions
- promoted `github.com/hashicorp/go-retryablehttp` to a direct dependency, used to implement the new request `retry` support

### Fixed

- preserved connection-pool reuse when `ignore_tls` is enabled at the provider level: the request client now reuses the provider's existing `http.Transport` instead of allocating a fresh one per request, avoiding goroutine and file-descriptor churn across repeated requests

## [3.2.2] - 2026-06-24

### Changed

- changed the Go module dependencies to their latest versions

## [3.2.1] - 2026-06-19

### Changed

- changed the Go module dependencies to their latest versions

## [3.2.0] - 2026-06-18

### Added

- added regression tests for the in-place update re-issue behavior: a unit test asserting that only request-defining attribute changes trigger a re-issue, and integration tests (driven by a local endpoint that returns a fresh id per request) proving that a client-side-only change neither re-issues the request nor fails apply, that a genuine request change still re-issues consistently, and that write-only destroy controls enabled in the same apply as a client-side-only change are still honored on destroy

### Changed

- changed the Go module dependencies to their latest versions

### Fixed

- fixed `http_request` failing an in-place update with `Error: Provider produced inconsistent result after apply` when only a client-side attribute changed (for example `tolerated_status_codes`). `Update` unconditionally re-issued the request, but the computed response attributes (`id`, `response_code`, `response_body`, `response_body_id`, `response_body_json`, `delete_resolved_path`) carry `UseStateForUnknown`, so the plan pinned them to their prior values while the re-issued request returned a different response — and for a non-idempotent method the request was repeated needlessly. The request is now re-issued only when an attribute that defines it actually changes (`method`, `path`, `headers`, `request_body`, `query_parameters`, `base_url`, `basic_auth`, `ignore_tls`); a change limited to response-interpretation attributes keeps the recorded response untouched. When the request does change, those computed attributes (including `delete_resolved_path`) are planned as unknown so the freshly captured response is accepted — which also resolves the same inconsistency for legitimate `request_body` / `headers` / etc. edits
- fixed `terraform destroy` not honoring write-only destroy controls (`is_delete_enabled`, `delete_method`, `delete_path`, `delete_headers`, `delete_request_body`) that were changed in the same apply as a client-side-only update. Because those attributes are read only from configuration and persisted in opaque private state, the short-circuiting update path now refreshes them into private state just like a full create/update, so a later destroy uses the current delete configuration instead of the stale one

## [3.1.10] - 2026-06-15

### Changed

- changed the Go module dependencies to their latest versions

### Fixed

- fixed `http_request` failing the first `plan` after a `2.x` state (schema `v0`) is upgraded to `3.x`, with `Error: Value Conversion Error ... Path: delete_headers` (`Map[!!! MISSING TYPE !!!]` / `tftypes.Map[tftypes.DynamicPseudoType]`). The schema `v0`->`v1` state upgrader rebuilt the model without the new WriteOnly delete-control attributes, so `delete_headers` took Go's zero-value `types.Map{}` (element-typeless) instead of a typed null. The upgrader now sets `is_delete_enabled`, `delete_method`, `delete_path`, `delete_headers` (`types.MapNull(types.StringType)`), and `delete_request_body` to typed nulls. Affects every configuration that upgrades pre-`3.0.0` state — the resource was un-plannable on `3.0.0`-`3.1.9` until the state was recreated

## [3.1.9] - 2026-06-09

### Changed

- changed the Go module dependencies to their latest versions
- refreshed `CLAUDE.md` and `.github/copilot-instructions.md` to reference Go `1.26.4` to match `go.mod`

## [3.1.8] - 2026-06-03

### Changed

- changed the Go module dependencies to their latest versions
- changed the Go version to `1.26.4` and updated all module dependencies

## [3.1.7] - 2026-05-28

### Changed

- changed the Go module dependencies to their latest versions

## [3.1.6] - 2026-05-25

### Changed

- changed the Go module dependencies to their latest versions

## [3.1.5] - 2026-05-22

### Changed

- changed the Go module dependencies to their latest versions

## [3.1.4] - 2026-05-20

### Changed

- changed the Go module dependencies to their latest versions

## [3.1.3] - 2026-05-19

### Changed

- changed the Go module dependencies to their latest versions

## [3.1.2] - 2026-05-08

### Changed

- changed the Go module dependencies to their latest versions
- changed the Go version to `1.26.3`
- extracted repeated string literals (`basic_auth`, `ignore_tls`, `username`, `password`) into package-level constants to satisfy `goconst` lint rule
- updated `.github/copilot-instructions.md` and `CLAUDE.md` to reference Go `1.26.3` to match `go.mod`

## [3.1.1] - 2026-04-29

### Changed

- changed the Go module dependencies to their latest versions

### Fixed

- fixed silent regression where every release since `3.0.0` (2026-03-31) shipped with zero assets, because the repo's `.goreleaser.yml` declares a `signs:` block referencing `{{ .Env.GPG_FINGERPRINT }}` but the shared `delivery-binary` action never imported a GPG key or populated that env var. GoReleaser failed at `signing artifacts` and uploaded nothing — leaving Terraform Registry with empty release pages for `3.0.0`, `3.0.1`, `3.0.2`, `3.0.3`, `3.0.4`, `3.0.5`, and `3.1.0`. Wired `gpg_sign: true` into the `go-binary.yaml` reusable workflow and forwarded the existing repo secrets `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE`. The new `crazy-max/ghaction-import-gpg@v6` step ([added in pipelines PR #388](https://github.com/rios0rios0/pipelines/pull/388)) imports the key and exposes its fingerprint to GoReleaser as `GPG_FINGERPRINT` at runtime, so no `GPG_FINGERPRINT` secret needs to be maintained in sync with the private key

## [3.1.0] - 2026-04-28

### Added

- added `CLAUDE.md` with build commands, architecture overview, and key conventions for Claude Code sessions

### Changed

- changed the Go module dependencies to their latest versions
- refreshed `.github/copilot-instructions.md` to reflect v3.x state: Go 1.26.2, Terraform 1.11+, auto-detected version via ldflags, WriteOnly delete attributes, and `tolerated_status_codes`

## [3.0.6] - 2026-04-24

### Changed

- changed the Go module dependencies to their latest versions

## [3.0.5] - 2026-04-23

### Changed

- changed the Go module dependencies to their latest versions

## [3.0.4] - 2026-04-21

### Changed

- changed the Go module dependencies to their latest versions

## [3.0.3] - 2026-04-17

### Changed

- changed the Go module dependencies to their latest versions

## [3.0.2] - 2026-04-15

### Changed

- changed the Go version to `1.26.2` and updated all module dependencies

## [3.0.1] - 2026-04-01

### Changed

- changed the Go module dependencies to their latest versions

## [3.0.0] - 2026-03-31

### Changed

- **BREAKING CHANGE:** existing resources that stored destroy parameters in state will have those values removed automatically when Terraform runs the built-in state upgrade to schema version `1`. The new values are read from the configuration at the next `terraform apply` that performs a Create or Update operation, at which point they are stored in opaque private state for use by `terraform destroy`. Resources that are destroyed before any subsequent apply after upgrading the provider will use the default behavior (`is_delete_enabled = false`), meaning no remote HTTP delete call is made
- changed `is_delete_enabled`, `delete_method`, `delete_path`, `delete_headers`, and `delete_request_body` to `WriteOnly` schema attributes (requires Terraform `1.11+`): they are no longer persisted in Terraform state and no longer produce plan diffs when changed in configuration. Values are stored in provider private state and read back during `terraform destroy`
- changed schema version from `0` to `1` to support the automatic state upgrade that removes the above destroy parameters from any existing state
- changed the Go module dependencies to their latest versions

## [2.4.2] - 2026-03-22

### Changed

- changed the Go module dependencies to their latest versions

## [2.4.1] - 2026-03-19

### Changed

- changed the Go module dependencies to their latest versions
- changed version injection to use `ldflags` at build time instead of a hardcoded constant

## [2.4.0] - 2026-03-12

### Added

- added `tolerated_status_codes` attribute to `http_request` resource, allowing specific non-2xx HTTP status codes (e.g. 404) to be treated as successful instead of causing errors

### Changed

- changed the Go version to `1.26.0` and updated all module dependencies
- changed the Go version to `1.26.1` and updated all module dependencies
- updated `.github/copilot-instructions.md` to reflect the current project state (v2.3.0, Go 1.26.0, new features)

### Fixed

- fixed `golangci-lint` findings: replaced `interface{}` with `any` (`modernize`), added `nolint` directives for expected SSRF and password field patterns (`gosec`), and wrapped long lines (`golines`)
- fixed `Makefile` script paths to match the updated pipelines repository structure
- fixed CodeQL SARIF upload by adding `security-events: write` permission to CI workflow

## [2.3.0] - 2025-12-23

### Added

- added `ignore_changes` feature to support ignoring specific attributes during updates
- added ability to use `count` and `for_each` with different APIs by specifying configuration at resource level instead of provider level
- added resource-level configuration support for `base_url`, `basic_auth`, and `ignore_tls` attributes in `http_request` resource
- added validation to ensure at least one base URL is configured (either at provider or resource level)

### Changed

- changed provider-level `url` attribute from required to optional (can now be provided at resource level)
- changed the state and plan flows to ignore delete control fields by default instead of destructing the resource when those fields were changed
- improved error handling with clear messages when no base URL is configured anywhere

## [2.2.0] - 2025-08-22

### Added

- added Copilot instructions on how to build and install this project
- added destruction mechanism (destroy method) to use additional and optional parameters in `http_request` resource
- added new documentation to explain how to use the optional destruction parameters

## [2.1.0] - 2025-08-12

### Added

- added the option to send query parameters when using the provider

### Fixed

- fixed supports for `?` query and `#` fragment characters in the `http_request` resource path parameter

## [2.0.2] - 2025-01-21

### Fixed

- fixed all arguments to make them force the resource recreation in the `http_request` resource, avoiding issues while changing the read-only (computed values) state

## [2.0.1] - 2025-01-20

### Fixed

- fixed provider's URL value assessment that was triggering empty when it was actually set
- fixed the ID generation instead of using `sha1` of timestamp (which is not unique), it's using the `uuid` to guarantee the uniqueness

## [2.0.0] - 2025-01-17

### Changed

- **BREAKING CHANGE:** corrected import state using two parts in the ID to guarantee the resource consistency
- corrected validation inside the provider to avoid having an empty URL when it's required

## [1.2.1] - 2024-12-09

### Removed

- removed `IsUnknown` from inside the `ValidateConfig` method to avoid issues when applying without a previous state

## [1.2.0] - 2024-12-09

### Added

- added more testing to cover cases in the provider configuration

### Changed

- upgraded all dependencies to the latest version
- upgraded to GoLang version 1.23.4

## [1.1.1] - 2024-11-18

### Fixed

- fixed null pointer error on the provider validation method

## [1.1.0] - 2024-11-18

### Added

- added JSON handling to perform better operations with the response
- added state importing feature with Base64 encoding

### Changed

- changed all the code styling to follow the standard proposed at [pipelines](https://github.com/rios0rios0/pipelines/blob/main/global/scripts/golangci-lint/.golangci.yml) repository
- corrected basic testing with basic checking with three cases
- corrected documentation to have examples in the official page
- corrected the structure to be more readable using DDD

## [1.0.0] - 2024-08-24

### Changed

- **BREAKING CHANGE:** changed the code to comply with the new Terraform SDK, according to the [tutorial](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider-configure)

## [0.0.6] - 2024-08-23

### Changed

- corrected the code to have JSON as response body conversion
- corrected the panic when applying the resource for the first time

## [0.0.5] - 2024-08-23

### Changed

- corrected the code to have a `request_body` field in the `http_request` resource

## [0.0.4] - 2024-08-23

### Added

- added features to handle TSL and Basic Auth in the provider

### Changed

- moved the responsibility to handle the URL from resource to provider

## [0.0.3] - 2024-08-23

### Changed

- corrected the missing `response_code` field in the output of `http_response` resource

## [0.0.2] - 2024-08-23

### Added

- added default publishing files recommended by [Terraform documentation](https://developer.hashicorp.com/terraform/registry/providers/publishing)

## [0.0.1] - 2024-08-23

### Added

- added the new code quickly to test and validate the new feature
