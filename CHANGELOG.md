# Changelog

## [Unreleased]

## [v0.12.0] - 2026-10-24

### Added

- **OPA Bundle Server & Controller Wiring**
  Wired the OPA SDK (`opa_sdk` config block) into server initialization (`setupOpaSDK` with 30-second readiness check and graceful shutdown) and enabled the `opa_bundle` validator and `opa_bundle_json_patch` mutator in the admission pipeline.
- **Configuration Preflight Validation**
  Added comprehensive `Config.Validate` startup checks executed after HCL decoding and during server construction. Validates port ranges, bind addresses, absolute HTTP(S) Nomad and webhook URLs, listener and upstream TLS certificate/key pairings, known controller types, required rule/webhook/notation blocks, duplicate mutator/validator names, and `opa_sdk` presence when bundle controllers are configured.
- **Documentation & Governance**
  Rewrote `README.md` with updated architecture, payload contracts, and HCL syntax; added `CONTRIBUTING.md`, `SECURITY.md`, and [update_and_nomad_dep_removal.md](update_and_nomad_dep_removal.md).
- **Tracing Span Attribute**
  Introduced `attrNomadJobID` (`nomad.job.id`) constant for tracing spans.

### Breaking Changes

- **Jobs that previously bypassed admission control are now evaluated**
  The route patterns matching `/v1/job/:id` and `/v1/job/:id/plan` were malformed and only matched a subset of legal Nomad job IDs. Job IDs containing underscores or dots, starting with a digit, or using uppercase letters were proxied straight to Nomad with **no mutators or validators applied**.
  - These requests now go through the full admission pipeline. Jobs that silently slipped past your policies before may now be mutated or rejected.
  - Review your policies against existing workloads before upgrading.
- **`tokenInfo` in policy input is now sanitized**
  When `resolve_token` is enabled, the ACL token passed to OPA policies and webhooks is no longer the full `api.ACLToken`. It is now a reduced `ACLTokenContext` that omits `SecretID` and the creation-only `ExpirationTTL`.
  - All other fields keep their names and JSON shape (`AccessorID`, `Policies`, `Roles`, `Global`, `CreateTime`, `ExpirationTime`, `CreateIndex`, `ModifyIndex`).
  - Any policy or webhook reading `SecretID` or `ExpirationTTL` must be updated.
  - `RequestContext.ResolveToken` is now populated to reflect whether token resolution was requested; it was previously always `false`.
- **OPA bundle validator fails closed on malformed decisions**
  A bundle policy decision that is not a JSON object, or whose `errors`/`warnings` are not lists of strings, is now treated as a policy contract failure and rejects the job. Previously such decisions silently passed validation.
- **Configuration is validated at startup**
  Invalid configurations that previously started (or panicked at request time) now refuse to start. An explicitly supplied but missing `-config` path now fails startup instead of silently falling back to the default configuration (defaults still apply when no `-config` is given).
- **`json_patch_webhook` mutators honour the `errors` response field**
  The documented `errors` field was previously parsed and discarded, allowing a rejected mutation to proceed. Errors returned by the webhook now reject the request.
- **Upstream error responses are passed through untouched**
  Warning and validation-error injection is now limited to successful (2xx) upstream responses, so Nomad's own error bodies reach the client intact instead of being unmarshalled into a response struct and mangled.
- **Stricter Rego assignment safety in OPA 1.19**
  OPA now treats the right-hand side of `:=` as a read that must be made safe by other expressions, and it can no longer be satisfied through the left-hand side (`rego_unsafe_var_error`). Custom policies should be verified against OPA 1.19.
- **Error attributes in exported logs now use the `exception.*` semantic conventions**
  The `otelslog` bridge records an error-valued log attribute on the log record's error field rather than as a regular attribute, exporting it as `exception.message` / `exception.type`. Collector pipelines, queries, or dashboards matching on an `error` attribute must be updated.

### Security

- **Validators now evaluate the job that is actually forwarded to Nomad**
  `ApplyAdmissionControllers` previously ran validators against the caller's original payload while returning the mutator chain's output, so on the register and plan paths a mutator could introduce content that validators never inspected. Validators now receive the final mutated job, each as an isolated copy, so one validator cannot alter what another validator or Nomad sees.
- **Token resolution failures now stop the request**
  With `resolve_token` enabled, a failed `/v1/acl/token/self` lookup previously logged an error and continued through admission and upstream proxying without the required identity context. Such failures now abort the request immediately.
- **ACL token secrets are no longer sent to policy engines**
  `SecretID` was previously serialized into every OPA and webhook admission payload; it is now stripped via `SanitizeACLToken`.
- TLS 1.2 is now the minimum version for both the listener and upstream Nomad connections.
- Invalid PEM CA files are rejected instead of silently producing an empty trust pool.
- Webhook endpoints must be absolute HTTP(S) URLs, validated at configuration load.
- Admission request bodies are limited to 32 MiB and webhook response bodies to 10 MiB.
- Webhook calls now have a 30-second timeout; previously they could hang indefinitely.

### Fixed

- Fixed a nil-pointer panic when rendering a validation error on the `/v1/validate/job` response path.
- Safe nil-checked Job ID resolution in admission logging and tracing to prevent panics on malformed payloads.
- Webhook mutators and validators now close their HTTP response bodies, send consistent `Content-Type` / `Accept: application/json` headers, and share a single instrumented transport instead of allocating a new one per call.
- Gzip response handling now closes both the decompressor and the underlying upstream body, and no longer discards write and close errors.
- Non-2xx webhook responses are now rejected instead of being decoded as if they were successful.
- Upstream Nomad TLS now supports CA-only, client-certificate, and system-root configurations instead of requiring all three files.
- The server handles `SIGTERM` in addition to `SIGINT`, shuts down with a 30-second deadline, and propagates shutdown failures. A normal `http.ErrServerClosed` is no longer reported as a runtime failure.
- `ReadHeaderTimeout` and `IdleTimeout` are now configured on the HTTP server.
- `X-Forwarded-For` values are trimmed, and an unparseable `RemoteAddr` no longer yields an empty client IP.
- Notation defaults (`max_sig_attempts`) are now applied to nested `opa_rule` notation blocks on both validators and mutators, not just top-level validators.
- Fixed a data race on the Testcontainers log consumer used by the OpenTelemetry tests.

### Dependencies & Licensing

- **Removed the `github.com/hashicorp/nomad` main module**
  The main module has been licensed under BUSL-1.1 since Nomad v1.6.6, while NACP and `nomad/api` are both MPL-2.0. NACP now depends only on `github.com/hashicorp/nomad/api`.
  - `MergeMultierrorWarnings` is now vendored at `pkg/helper/warning.go`, taken verbatim from Nomad v1.6.3 (MPL-2.0), preserving `X-Nomad-Warnings` formatting byte-for-byte.
  - Nomad's `helper/tlsutil` and `lib/file` test helpers were replaced with `crypto/x509` and `os.WriteFile` equivalents in `cmd/nacp/tlshelper_test.go`.
  - Removes all BUSL-1.1 code and unused `hashicorp/go-hclog` from the dependency tree and roughly halves the module graph (510 → 258 modules). See [update_and_nomad_dep_removal.md](update_and_nomad_dep_removal.md) for the full rationale.
- **OPA upgraded from 1.14.1 to 1.19.0**
  The Wasm engine switched from `wasmtime-go` to the pure-Go `wazero`, removing the CGO dependency from the build.
- **OpenTelemetry upgraded to 1.45.0 / 0.21.0 (contrib 0.70.0)**
  Adapted test assertions to use shared `go.opentelemetry.io/otel/attribute` types; emitted OTLP telemetry is unchanged.
- **Replaced `github.com/docker/docker`** with split `github.com/moby/moby/api` and `github.com/moby/moby/client` modules.
- **Updated Go toolchain** to Go 1.27 (`1.27.3`).

### Container Image

- The release image now ships the system CA certificate bundle sourced from `alpine:3.24.3` into the `scratch` runtime image (replacing `ubuntu:latest`). Previously `scratch` had no roots, causing HTTPS webhooks, OPA bundle servers, and registries to fail with `x509: certificate signed by unknown authority`.
- The runtime image runs as numeric non-root UID/GID `10001:10001` without requiring an `/etc/passwd` entry.

### Improvements

- Docker-backed Notation tests are behind an `integration` build tag, so `go test ./...` no longer requires a running Docker daemon. CI runs with `-tags=integration` to preserve coverage.

## [v0.11.0] - 2026-03-22

### Added
- **Initial OPA Bundle Implementations & Configuration**:
  - Added `OpaBundleValidator` (`pkg/admissionctrl/validator/opa_bundle_validator.go`) and `OpaBundleMutator` (`pkg/admissionctrl/mutator/opa_bundle_json_patch.go`).
  - Added `opa_sdk` and `opa_sdk_rule` HCL configuration blocks in `pkg/config`.
  - Added mock OPA bundle server test utilities in `testutil`.
- Updated dependencies (OPA 1.14.1, OTel 1.42.0, Go 1.26, `oras-credentials-go`, and notation archive handling).

## [v0.10.0] - 2025-10-03

### Breaking Changes
- **Upgraded to OPA v1**: Updated OPA imports and Rego syntax to OPA v1 (`contains ... if` syntax).

### Changed & Fixed
- **Package Structure**: Moved internal packages (`admissionctrl`, `config`, `logutil`, `o11y`, `otel`) under `pkg/` (#46).
- **Selective Token Resolution**: Fixed token resolution so `/v1/acl/token/self` is only queried on admission-controlled endpoints (`register`, `plan`, `validate`) rather than all proxied requests (#49).
- Fixed Rego policies in `testdata` according to Regal linter rules.

## [v0.9.0] - 2025-09-04

### Added & Changed
- **Reworked Logging Configuration (#45)**:
  - Replaced `telemetry.logging.type` with dedicated `slog` (`text`, `text_out`, `json`, `json_out`) and `otel` (`enabled`) blocks, allowing simultaneous logging to multiple outputs/handlers:
    ```hcl
    telemetry {
        logging {
            level = "info"
            slog {
                text     = true
                text_out = "stdout"
                json     = false
                json_out = "stdout"
            }
            otel {
                enabled = true
            }
        }
    }
    ```
  - Added minimum severity filtering (`minsev`) for OpenTelemetry log exports.
- Documented `types.Payload` admission input in `README.md`.
- Updated Go version to 1.25.0.

## [v0.8.0] - 2025-07-04

### Added
- **OpenTelemetry with OTLP Support (#26, #36)**: Added OpenTelemetry instrumentation for traces, metrics, and logs with OTLP HTTP exporters, configured via standard OpenTelemetry environment variables and the `telemetry` HCL block:
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
- **Resource Version Attribution**: Added `service.name` (`nacp`) and `service.version` to OpenTelemetry resource attributes.
- **`log/slog` for Logging**: Switched from `hashicorp/go-hclog` to standard library `log/slog`.
- **Initial Logging Flag**: Added `-bootstrap-json-logger` to configure the bootstrap logger format (`text` vs `json`).

### Breaking Changes
- **Switch from `hashicorp/go-hclog` to `log/slog` for Logging**: May affect log formatting and structure.
- **Move `log_level` config to `telemetry/logging/level`**: Relocated `log_level` under `telemetry.logging.level`.

### Improvements
- Improved controller handling of mutation jobs so mutations are aggregated across mutators.

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
