# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file is not edited by hand. Every change writes its own fragment under
`.changes/unreleased/` with [chlog](https://github.com/luizjhonata/chlog), and a release compiles
the pending fragments into a version section here — so two branches each adding an entry no
longer touch the same lines, and a rebase that used to conflict on this file now conflicts on
nothing.

When a new release is proposed:

1. Create a new branch `bump/x.x.x` (this isn't a long-lived branch!!!);
2. The fragments pending under `.changes/unreleased/` are compiled into a version section by `chlog batch auto && chlog merge` (AutoBump does this for you — it reads the fragments directly);
3. Verify the `VERSION` in the `Makefile` if applicable (it auto-detects from git tags by default);
4. Open a Pull Request with the bump version changes targeting the `main` branch;
5. When the Pull Request is merged, a new Git tag must be created using [GitHub environment](https://github.com/rios0rios0/terraform-provider-http/tags).

Releases to productive environments should run from a tagged version.
Exceptions are acceptable depending on the circumstances (critical bug fixes that can be cherry-picked, etc.).

## [Unreleased]

## [3.6.0] - 2026-08-26

### Added

- added a plan-time warning when a resource's `delete_headers` names a header the provider configuration also sets. The condition is objective and provider-knowable -- a key present in both maps -- and nothing inspects the value or guesses whether it is a credential. Names are compared canonically, because `http.Header.Set` canonicalises them, so `authorization` in one map really does overwrite `Authorization` in the other; the diagnostic reports the spelling the configuration used
- added a tailored `code-review` skill under `.github/skills/` so GitHub Copilot reviews changes against the [rios0rios0/guide](https://github.com/rios0rios0/guide/wiki) standards and this repository's own load-bearing invariants
- added eleven unit tests over the comparison and the diagnostic's name formatting (overlap, case-insensitive overlap, spelling preservation, no overlap, multiple names sorted, an unconfigured provider, a resource with no delete headers, and the four list renderings), plus `TestShadowedDeleteHeadersStayNonFatal`, which pins the severity: a great many existing configurations repeat the provider's credential in `delete_headers`, and that is a hazard rather than a defect -- it only bites once the credential is rotated. Raising it to an error would stop every one of those configurations from planning at all
- added the warning because the combination is a trap that is invisible until it fires, and then self-perpetuating. `delete_headers` is write-only and Terraform sends no configuration to a destroy (`resource.DeleteRequest` has no `Config` field, unlike `UpdateRequest`), so it is captured into private state at create/update and replayed on destroy -- where `makeDeleteModel` puts it exactly where a resource's own headers go, and `buildRequest` applies those AFTER the provider's. The resource value therefore wins. Fine for a header that describes the request; a trap for a credential, because rotating it means the destroy is sent with the value captured when the resource was created. If the endpoint rejects it the destroy fails, and when that destroy is the first half of a replacement -- which `headers` forces, being `RequiresReplace` as a whole map -- the replacement never completes, so state keeps the OLD value and every later plan proposes the same thing. No configuration edit reaches it, because the state write that would fix it is the one the failure prevents
- implemented the warning in `ModifyPlan` rather than `ValidateConfig`, for two reasons: the provider configuration it compares against is only populated after `Configure`, and a CREATE plan must be warned too -- which is why the call sits ahead of the state guard and behind the plan guard, so a destroy plan stays silent

### Changed

- changed the changelog to [chlog](https://github.com/luizjhonata/chlog) fragments: a change now writes its own YAML file under `.changes/unreleased/` through `chlog new --kind <Kind> --body "..."`, and `CHANGELOG.md` is GENERATED from them at release time by `chlog batch auto && chlog merge`. That is the one thing a single shared file cannot do — two branches each adding an entry no longer touch the same lines, so a rebase that used to conflict on `CHANGELOG.md` now conflicts on nothing. The `[Unreleased]` section was empty, so nothing had to be carried across. AutoBump already reads the fragments directly, so the release flow is unchanged.
- changed the Go module dependencies to their latest versions
- changed the provider documentation to state the rule the warning enforces: a credential that rotates belongs in the provider block's `headers` and nowhere else -- not in a resource's own `headers`, which forces replacement as a whole map, and not repeated in `delete_headers`. The existing `ignore_changes` note in the resource documentation said the delete fields "never trigger replacement", which is true and was reassuring in the wrong direction; it now also says they are write-only, that editing one alone produces no plan diff, and that the destroy keeps using the capture until some other state-stored attribute changes
- changed the provider-header integration tests to share `guardedConfigFor`, `withDeleteEnabled`, `applyThenDestroy` and `requireOnlyAuthorized`, so each case is left holding only its own given and then. The three cases repeated the whole guarded-server and POST-widget preamble, which is what turned pre-existing repetition into NEW-code duplication and failed the quality gate at `11.4%` against a `3%` ceiling. No behaviour change: every assertion is the one it was before

### Fixed

- fixed the `main` pipeline, which every repository's `sast:gitleaks` job had been failing since the code-review skill landed: the skill's own security bullet listed credential prefixes verbatim to warn against writing them, and the scanner's second pass matches those prefixes on their own, so the warning tripped the rule it was describing. The bullet now names the vendors instead, and the commit that carried the original wording is allowlisted by fingerprint in `.gitleaksignore`, because the scan walks the whole history reachable from `HEAD` and no edit at the tip can clear a past commit. No credential was ever committed.
- fixed the prerequisites in `CONTRIBUTING.md`, which still asked for Go 1.26+ and Terraform 1.10+ while `go.mod` requires `go 1.27.0` and the WriteOnly attributes the provider ships need Terraform 1.11+ (as `README.md` already states).

### Removed

- removed the two stale items from the bump pull request template that asked for a `version` constant in `main.go` and a `VERSION` variable in the `Makefile` to be hand-edited. The provider version is injected at build time through `-ldflags` — GoReleaser passes the tag and the `Makefile` derives it from `git describe` — so `main.go` deliberately stays at `"dev"` and editing either by hand would have been wrong.

## [3.5.6] - 2026-08-24

### Changed

- changed the Go module dependencies to their latest versions
- changed the Go version to `1.27.0` and updated all module dependencies
- refreshed `CLAUDE.md` and `.github/copilot-instructions.md` to reference the Go `1.27.0` version from `go.mod`

## [3.5.5] - 2026-08-17

### Changed

- changed the Go module dependencies to their latest versions

## [3.5.4] - 2026-08-15

### Changed

- changed the Go version to `1.26.6` and updated all module dependencies

## [3.5.3] - 2026-08-13

### Changed

- changed the Go module dependencies to their latest versions

## [3.5.2] - 2026-08-12

### Changed

- changed the Go module dependencies to their latest versions

## [3.5.1] - 2026-08-11

### Changed

- changed the Go module dependencies to their latest versions
- refreshed `CLAUDE.md` to document the provider-level `headers` merge order and the import-only `import_read_path`

## [3.5.0] - 2026-08-06

### Added

- added a provider-level `headers` map, sent on every request the provider makes and applied BEFORE
  each resource's own `headers`, so a resource naming the same header still overrides it. The
  override is case-insensitive because header names are: `http.Header.Set` canonicalises the name,
  so the two sides cannot disagree by casing alone. Merging is per header rather than
  all-or-nothing, so declaring resource headers does not discard the provider's, and the map is
  marked `Sensitive` because this is where a credential belongs when an API wants one in a header
  rather than in `basic_auth`
- added it because an import cannot authenticate any other way. A resource created with an unsafe
  method names `import_read_path` so the provider force-GETs the object instead of replaying the
  `POST` that would create a second one -- but `ImportState` is handed the import identifier and
  nothing else, since Terraform never shows it the configuration, so that GET is built from the
  identifier alone. An identifier deliberately omits `headers` (that is what keeps them adoptable:
  one that spells them out loses the in-place adoption and risks a `RequiresReplace` on the first
  plan), so a bearer token kept in the resource's `headers` is unavailable exactly when the import
  read needs it, and an API that authenticates through a header answers `401`. The remaining option
  was to spell the credential into the identifier, which works and prints it wherever plan output
  goes -- plan logs, review comments -- so there was no way to import against such an API without
  leaking the token. A provider-level header is the only place it can live and still be sent
- added the merge at the single point every request is built, so create, read, refresh, destroy and
  the import read are all covered by one call. The destroy is worth naming separately: it sends
  `delete_headers`, so a resource that declares none previously had no credential for its own
  teardown
- added no environment-variable equivalent, unlike the `basic_auth` scalars. A map has no
  unambiguous encoding for one, and the values belong in a Terraform variable; this is stated in the
  attribute description rather than left to be discovered
- added `TestProviderHeadersReachTheImportRead` and `TestProviderHeadersReachTheDestroyRequest`,
  which run against an `httptest` server that answers `401` to any request without the expected
  bearer, so a regression fails on the provider's own error rather than on a subtle assertion. Both
  were confirmed to fail without the feature, reproducing `Error performing HTTP request. Not
  expected status code... Response code: 401 Unauthorized`. Seven unit tests cover the merge itself:
  provider-only, resource override, case-insensitive override, per-header merge, the empty map, a
  provider `Content-Type` surviving the JSON defaults, and an unconfigured provider
- added nothing to the resource schema, so there is no state migration and no new upgrader: this is
  provider configuration, which has neither. Nothing is written to state, and no existing
  configuration changes behaviour -- the map is optional and absent by default

### Changed

- changed the `ValidateConfig` tests to build their provider value through `fullProviderValues`,
- changed the Go module dependencies to their latest versions
  `basicAuthValue` and `validateConfigOf` instead of spelling the whole six-attribute object out per
  case. Each of the four cases repeated all six attributes, so adding one attribute meant editing
  four near-identical blocks -- which is what turned pre-existing duplication into NEW-code
  duplication and failed the quality gate at `6.7%` against a `3%` ceiling. The new header tests got
  the same treatment through `resourceHeaderMap` and `buildTestRequest`, leaving each case holding
  only its own given and then. No behaviour change: every assertion is the one it was before, and it
  is a net 45 lines shorter

### Fixed

- fixed a latent nil-pointer dereference in the provider-level `basic_auth` fallback, which read
  `internal.Config` unguarded. Terraform sets provider data AFTER the `ConfigureProvider` RPC and
  `Configure` returns early while it is absent, so the field is reachable as nil -- `Configure`
  already checks for exactly that. Every consumer of the provider configuration now reads it through
  one nil-safe accessor, and `HasAuthentication` tolerates a nil receiver. Found by the unit test
  written for the new attribute, which segfaulted on the pre-existing line rather than on the new
  one

## [3.4.2] - 2026-08-04

### Changed

- changed the Go module dependencies to their latest versions

## [3.4.1] - 2026-08-01

### Changed

- changed the Go module dependencies to their latest versions

## [3.4.0] - 2026-07-31

### Added

- added import adoption, so importing a resource never destroys and recreates it. Terraform does
  not show a configuration to a provider during import, so an identifier that omits arguments
  produces a state that differs from the HCL, and that difference used to land on the
  `RequiresReplace` rules. The provider now records what the identifier left unsaid and adopts
  those values from the configuration on the first apply -- in place, without sending any HTTP
  request, and with a warning naming exactly what was adopted. Neither `terraform state rm` nor a
  `lifecycle` block is needed any more
- added five import identifier formats alongside the existing `<id>/<base64>` pair: a bare path
  (`/posts/1`, the method defaults to `GET`), a method and path (`POST /posts`), raw JSON, a JSON
  file reference (`@./import.json`), and a bare base64 payload. Each is distinguished by its first
  character, so no identifier can be read as two different forms. Only `method` and `path` are ever
  required
- added resource identity, so Terraform `1.12` and later can import with
  `import { identity = { method = ..., path = ... } }` instead of a stringly-typed identifier
- added `import_id`, a computed attribute holding the identifier that re-imports the resource with
  the arguments it was applied with. Neither `basic_auth` nor the captured response is encoded into
  it: a response body is often the most sensitive thing the resource holds and is already in state
  under its own attribute, so embedding a copy would leak it wherever the identifier is pasted and
  inflate the state for nothing. A re-import captures a current response instead, and for a method
  that cannot be replayed the resolved `refresh_path` travels as `import_read_path` so it still can
- added a live read during import: for `GET` and `HEAD` the endpoint is read so `response_code`,
  `response_body`, `response_body_id`, `response_body_json` and `delete_resolved_path` are captured
  from the real API. `POST`, `PUT`, `PATCH` and `DELETE` are never replayed, because re-sending one
  would repeat its side effect; the new `import_read_path` payload key names an object to `GET`
  instead. This is what makes a `delete_path` carrying JSONPath tokens resolvable after an import
- added opt-in drift detection through `is_refresh_enabled` and `refresh_path`, replacing the `Read`
  method that had been a no-op. When enabled, every refresh re-reads the resource and updates the
  captured response; a response that is neither successful nor tolerated removes the resource from
  state so it is planned for creation again. It is off by default, so no existing configuration
  changes behaviour

### Changed

- changed the resource schema to version `3` and added a `2` to `3` upgrader carrying the new
  attributes as typed nulls, following the precedent set by the version `0` upgrader
- changed the import identifier documentation and examples to emit URL-safe, unpadded base64, which
  cannot contain the `/` separator

### Fixed

- fixed import rejecting any `<id>/<base64>` identifier whose payload contained a `/`. The standard
  base64 alphabet includes that character, so a request path with a query string was enough to
  produce one; the identifier was split into three parts and refused outright. The split is now
  bounded to two segments, and standard, URL-safe, padded and unpadded base64 are all accepted
- fixed imported state recording `is_response_body_json` as `false` when the identifier omitted it.
  A configuration that also omits the argument leaves it null, so the two disagreed and the first
  plan after an import destroyed and recreated the resource. Omitted arguments now stay null, and
  an explicit `false` stays distinguishable from an absent one -- the same fix applies to
  `ignore_tls`, `is_delete_enabled` and `response_code`

## [3.3.9] - 2026-07-30

### Changed

- changed the Go module dependencies to their latest versions

### Fixed

- fixed `response_body_id`, `delete_resolved_path` and `response_body_json` recording whole JSON
  numbers in scientific notation (an id of `803554429` became `8.03554429e+08`), which broke every
  consumer that interpolated the value back into a request -- most visibly `delete_path`, leaving
  affected resources impossible to destroy. Whole numbers within the exactly-representable float64
  range are now rendered positionally; fractional numbers, booleans, strings and JSON null keep
  their previous rendering
- fixed already-written state carrying the notation above: the resource schema moves to version 2
  and a state upgrader rewrites the three attributes in place, driven by the numbers in each
  resource's own recorded `response_body` so nothing is guessed

## [3.3.8] - 2026-07-28

### Changed

- changed the Go module dependencies to their latest versions

## [3.3.7] - 2026-07-27

### Changed

- changed the Go module dependencies to their latest versions

## [3.3.6] - 2026-07-22

### Changed

- changed the Go module dependencies to their latest versions

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
