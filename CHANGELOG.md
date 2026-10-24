# Changelog

## [Unreleased]

## [v0.12.0] - 2026-10-24

### Security

- **Validators evaluate the final mutated job**
  `ApplyAdmissionControllers` now provides validators with an isolated copy of the final mutated job produced by the mutator chain rather than the original caller payload. This prevents mutations from escaping validation and ensures validators cannot mutate inputs.
- **Jobs previously bypassing admission control are now evaluated**
  Route patterns matching `/v1/job/:id` and `/v1/job/:id/plan` previously matched only a subset of legal Nomad job IDs. Job IDs containing underscores, dots, uppercase letters, or starting with digits bypassed admission hooks entirely and were proxied directly to Nomad. They now undergo full admission evaluation.
- **Sanitized `tokenInfo` in policy context**
  When `resolve_token` is enabled, the ACL token passed to OPA policies and webhooks is now an `ACLTokenContext` omitting `SecretID` and creation-only `ExpirationTTL`. Other fields (`AccessorID`, `Policies`, `Roles`, `Global`, `CreateTime`, `ExpirationTime`, `CreateIndex`, `ModifyIndex`) remain unchanged. `RequestContext.ResolveToken` now accurately reflects whether token resolution was requested.
- **Token resolution failures abort the request**
  When `resolve_token` is enabled, a failure to look up `/v1/acl/token/self` immediately halts request processing instead of proceeding unauthenticated.
- **TLS 1.2 minimum version enforced**
  Both the listener and upstream Nomad client TLS connections require at least TLS 1.2.
- **Strict PEM CA validation**
  Invalid PEM CA certificates now fail configuration/startup instead of silently producing an empty root pool.
- **Payload limits and timeouts**
  Admission request bodies are limited to 32 MiB, webhook responses to 10 MiB, and webhook calls timeout after 30 seconds.
- **Webhook endpoint constraints**
  Webhook targets must be absolute HTTP/HTTPS URLs, validated at startup.

### Breaking Changes

- **Configuration validated at startup**
  Invalid configurations (invalid port range/bind address, non-absolute Nomad or webhook URLs, missing TLS cert/key pairs, duplicate controller names, or missing explicit `-config` paths) fail startup immediately instead of panicking or falling back silently.
- **OPA bundle validator fails closed on malformed decisions**
  Decisions from OPA bundle policies that are not JSON objects, or whose `errors` / `warnings` fields are not arrays of strings, now fail closed and reject the job as a policy contract failure.
- **`json_patch_webhook` mutators honour the `errors` response field**
  Errors returned by JSON-Patch mutation webhooks now reject the request instead of being silently ignored.
- **Upstream error responses passed through untouched**
  Validation-error and warning injection is now restricted to successful (2xx) upstream Nomad responses so Nomad's own error payloads reach the client intact.
- **Stricter Rego assignment safety in OPA 1.19**
  OPA 1.19 treats the RHS of `:=` as a read that must be made safe by other expressions. Verify custom policies against OPA 1.19.
- **Error attributes in exported logs now use `exception.*` semantic conventions**
  The `otelslog` bridge records error log attributes using `exception.message` and `exception.type`. Collector pipelines and dashboards matching on an `error` attribute must be updated.

### Added & Improved

- **Refactored OPA SDK & Bundle Wiring**
  Refactored OPA SDK initialization and dependency injection, supporting cleaner test isolation and robust lifecycle management for dynamic policy bundles.
- **Tracing `nomad.job.id` Span Attribute**
  Introduced `attrNomadJobID` constant for tracing spans to cleanly correlate traces by Nomad Job ID.
- **Docker-backed Notation tests isolated**
  Notation tests requiring Docker are behind the `integration` build tag, allowing local `go test ./...` execution without Docker while maintaining full coverage in CI.

### Fixed

- Fixed a nil-pointer panic when rendering a validation error on the `/v1/validate/job` response path.
- Webhook mutators and validators now close HTTP response bodies and share a single instrumented transport.
- Gzip response handling now properly closes both the decompressor and upstream response body, correctly propagating write and close errors.
- Non-2xx webhook responses are rejected cleanly with contextual errors.
- Upstream Nomad TLS now supports CA-only, client-certificate, and system-root configurations.
- The server cleanly handles `SIGTERM` and `SIGINT` with a 30-second graceful shutdown deadline; `http.ErrServerClosed` is no longer logged as a failure.
- `ReadHeaderTimeout` and `IdleTimeout` are configured on the HTTP server.
- `X-Forwarded-For` header entries are trimmed, and unparseable `RemoteAddr` values fall back safely.
- Notation defaults (`max_sig_attempts`) are applied to nested `opa_rule` notation blocks on mutators and validators.
- Fixed a data race on the Testcontainers log consumer in OpenTelemetry tests.

### Dependencies & Licensing

- **Removed `github.com/hashicorp/nomad` main module**
  Removed the BUSL-1.1 licensed Nomad main module dependency, leaving only `github.com/hashicorp/nomad/api` (MPL-2.0). NACP is now entirely free of BUSL-1.1 code, cutting module dependencies in half (510 → 258 modules). `MergeMultierrorWarnings` has been vendored from Nomad v1.6.3 (MPL-2.0).
- **OPA upgraded from 1.16 to 1.19**
  Wasm engine migrated from `wasmtime-go` to pure-Go `wazero`, removing CGO build dependencies.
- **OpenTelemetry upgraded to 1.45.0 / 0.21.0 (contrib 0.70.0)**
  Updated to shared attribute APIs (`go.opentelemetry.io/otel/attribute`).
- **Dropped `github.com/docker/docker`** in favor of split moby modules.
- **Go toolchain updated** to latest Go LTS.

### Container Image

- Release image includes the system CA certificate bundle sourced from Alpine 3.24 (updated to latest release) into the minimal `scratch` image, allowing HTTPS webhooks, OPA bundle servers, and public registries to work out of the box with up-to-date roots.
- Container executes as non-root UID/GID `10001:10001`.

## [v0.11.0] - 2026-03-22

### Added
- **OPA Bundle Support (`opa_bundle`)**: Introduced OPA SDK integration allowing policy bundles to be loaded and evaluated for:
  - `opa_bundle` validator
  - `opa_bundle` JSON-Patch mutator
- Added mock OPA bundle server test utilities in `testutil`.
- Updated dependencies (oras-credentials-go, notation archive handling).

## [v0.10.0] - 2025-10-03

### Changed & Fixed
- Upgraded to OPA v1.
- Refactored internal directory layout and relocated packages into `pkg/`.
- Fixed token resolution error handling.
- Regal linting fixes across testdata mutators and validators.

## [v0.9.0] - 2025-09-04

### Added & Changed
- Reworked logging architecture.
- Go version updated to 1.25.
- Updated OpenTelemetry image and dependencies.

## [v0.8.0] - 2025-07-04

### Added
- **OpenTelemetry with OTLP Support**: Full OpenTelemetry instrumentation for traces, metrics, and logs with OTLP export, configured via standard OpenTelemetry environment variables or the `telemetry` block:
  ```hcl
  telemetry {
      logging {
          type = "otel"
      }
      metrics {
          enabled = true
      }
      tracing {
          enabled = true
      }
  }
  ```
- **`log/slog` Standard Structured Logging**: Switched logging from `hashicorp/go-hclog` to Go's standard library `log/slog`.
- **Startup JSON Logger Flag**: Added `-bootstrap-json-logger` flag to format initial startup logs as JSON.
- **Resource Version Header**: Added resource version support.

### Breaking Changes
- Switch from `hashicorp/go-hclog` to `log/slog`.
- Relocated `log_level` setting under `telemetry/logging/level`.

### Improvements
- Improved controller handling of mutation jobs so it can be ensured that job mutations are aggregated.

## [v0.7.0]

### Breaking Changes
- **Controller Signature Refactor**
  The `Job`-only signature in the admission controller has been replaced with a new `types.Payload` struct.
  - All mutators and validators now receive a `Payload` object containing both the `Job` definition and additional context (e.g., client IP, resolved token details).
  - Any custom integrations using the old `Job`-based method signatures must be updated to use `types.Payload`.

- **OPA Input Changes**
  The embedded OPA validator has been updated to accept a new input structure containing job and caller context.
  - Policies and data references relying on the previous input format must be updated accordingly.

- **Remote Webhook Contract Change**
  Webhook mutators and validators now receive a request body with the combined job and context data instead of job-only information.
  - Downstream services expecting the old JSON schema must be updated to parse the new `Payload` format.

### Added
- **Token Resolution & Context Passing**
  Hooks can now resolve Nomad tokens (with optional policy extraction) and pass the accessor ID, client IP, and other metadata through mutators and validators.
  - New configuration flag `resolveToken` enables token resolution for specific hooks to avoid unnecessary overhead when not required.
  - Enhanced support for use cases like CIDR-based validation, custom ACL logic, and extended audit logging.

- **Changelog Initialization**
  Introduced a `CHANGELOG.md` to track significant updates, especially breaking changes and added features.

### Rational

With these changes, you can now:
- Perform CIDR-based validations by leveraging the client IP.
- Create advanced ACL logic by passing resolved ACL token details (accessor ID, policies) to OPA or remote webhooks.
- Implement more granular auditing or custom workflows by integrating the new, richer `context` data available in each request.
