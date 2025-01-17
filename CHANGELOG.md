# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

When a new release is proposed:

1. Create a new branch `bump/x.x.x` (this isn't a long-lived branch!!!);
2. The Unreleased section on `CHANGELOG.md` gets a version number and date;
3. Update the `version` constant in the `main.go` file;
4. Open a Pull Request with the bump version changes targeting the `main` branch;
5. When the Pull Request is merged, a new git tag must be created using [GitHub environment](https://github.com/rios0rios0/terraform-provider-http/tags).

Releases to productive environments should run from a tagged version.
Exceptions are acceptable depending on the circumstances (critical bug fixes that can be cherry-picked, etc.).

## [Unreleased]

### Changed

- **BREAKING CHANGE:** corrected import state using two parts in the ID to guarantee the resource consistency
- corrected validation inside the provider to avoid having empty URL when it's required

## [1.2.1] - 2024-12-09

### Removed

- removed `IsUnknown` from inside the `ValidateConfig` method to avoid issues in to apply without previous state

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
- corrected basic testing with basic checking with 3 cases
- corrected documentation to have examples in the official page
- corrected the structure to be more readable using DDD

## [1.0.0] - 2024-08-24

### Changed

- **BREAKING CHANGE:** changed the code to comply with the new Terraform SDK according to the [tutorial](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider-configure)

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
