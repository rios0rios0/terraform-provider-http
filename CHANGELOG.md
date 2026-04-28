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

### Added

- added `CLAUDE.md` with build commands, architecture overview, and key conventions for Claude Code sessions

### Changed

- refreshed `.github/copilot-instructions.md` to reflect v3.x state: Go 1.26.2, Terraform 1.11+, auto-detected version via ldflags, WriteOnly delete attributes, and `tolerated_status_codes`
- changed the Go module dependencies to their latest versions
- changed the Go module dependencies to their latest versions

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
