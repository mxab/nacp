# Changelog

## [Unreleased]

### OpenTelemetry
OpenTelemetry with OTLP support has been added, allowing for better observability and monitoring of admission requests and responses.

The otlp exporter is configured by the common otel environment variables.

```hcl
telemetry {
    logging {
        otel {
            enabled = true
        }
    }
    metrics {
        enabled = true
    }
    tracing {
        enabled = true
    }
}
```


### `log/slog` for logging

Switch from `hashicorp/go-hclog` to `log/slog` for logging.

### Initial logging config
Allow configuring the initial logger from `text` to `json` formats by adding `-bootstrap-json-logger`

### Breaking Changes

- **Switch from `hashicorp/go-hclog` to `log/slog` for Logging**
  The logging library has been changed from `hashicorp/go-hclog` to the standard library's `log/slog`.
  - This change may affect how logs are formatted and structured.

- **Move `log_level` config to `telemetry/logging/level`**
  The `log_level` setting has been moved under the `telemetry/logging/level` path to better organize telemetry-related configurations.

- **Jobs that previously bypassed admission control are now evaluated**
  The route patterns matching `/v1/job/:id` and `/v1/job/:id/plan` were malformed and only matched a subset of legal Nomad job IDs. Job IDs containing underscores or dots, starting with a digit, or using uppercase letters were proxied straight to Nomad with **no mutators or validators applied**.
  - These requests now go through the full admission pipeline. Jobs that silently slipped past your policies before may now be mutated or rejected.
  - Review your policies against existing workloads before upgrading.

- **`tokenInfo` in policy input is now sanitized**
  When `resolve_token` is enabled, the ACL token passed to OPA policies and webhooks is no longer the full `api.ACLToken`. It is now a reduced `ACLTokenContext` that omits `SecretID` and the creation-only `ExpirationTTL`.
  - All other fields keep their names and JSON shape, so policies reading `AccessorID`, `Policies`, `Roles`, `Global`, `CreateTime`, `ExpirationTime`, `CreateIndex`, or `ModifyIndex` continue to work unchanged.
  - Any policy or webhook reading `SecretID` or `ExpirationTTL` must be updated.
  - `RequestContext.ResolveToken` is now populated to reflect whether token resolution was requested; it was previously always `false`.

- **OPA bundle validator fails closed on malformed decisions**
  A bundle policy decision that is not a JSON object, or whose `errors`/`warnings` are not lists of strings, is now treated as a policy contract failure and rejects the job. Previously such decisions silently passed validation.
  - Policies whose configured decision path can yield a scalar or `undefined` value must be updated to return an object.

- **Configuration is validated at startup**
  Configuration is now checked after HCL decoding and when building a server, instead of failing later at request time or panicking. Invalid configurations that previously started now refuse to start.
  - Covers port range and bind address, an absolute HTTP(S) Nomad address, listener TLS requiring both `cert_file` and `key_file`, upstream TLS cert/key pairing, known mutator and validator types, required `opa_rule`/`opa_sdk_rule`/`webhook`/`notation` blocks, absolute HTTP(S) webhook endpoints and valid methods, duplicate mutator and validator names, and bundle controllers requiring a configured `opa_sdk` block.
  - An explicitly supplied but missing `-config` path now fails startup instead of silently falling back to the default configuration. Defaults still apply when no `-config` is given.

- **`json_patch_webhook` mutators honour the `errors` response field**
  The documented `errors` field was parsed and then discarded, so a webhook rejecting a mutation was ignored and the job proceeded. Errors returned by the webhook now reject the request.

- **Upstream error responses are passed through untouched**
  Warning and validation-error injection is now limited to successful (2xx) upstream responses, so Nomad's own error bodies reach the client intact instead of being unmarshalled into a response struct and mangled.

### Security

- **Validators now evaluate the job that is actually forwarded to Nomad**
  `ApplyAdmissionControllers` ran validators against the caller's original payload while returning the mutator chain's output, so on the register and plan paths a mutator could introduce content that validators never inspected. Validators now receive the final mutated job, each as an isolated copy, so one validator cannot alter what another validator or Nomad sees.

- **Token resolution failures now stop the request**
  With `resolve_token` enabled, a failed `/v1/acl/token/self` lookup wrote an error and then continued through admission and upstream proxying without the required identity context. Such failures now abort the request.

- **ACL token secrets are no longer sent to policy engines**
  See the `tokenInfo` breaking change above; `SecretID` was previously serialized into every OPA and webhook admission payload.

- TLS 1.2 is now the minimum version for both the listener and upstream Nomad connections.
- Invalid PEM CA files are rejected instead of silently producing an empty trust pool.
- Webhook endpoints must be absolute HTTP(S) URLs, validated at configuration load.
- Admission request bodies are limited to 32 MiB and webhook response bodies to 10 MiB.
- Webhook calls now have a 30 second timeout; previously they could hang indefinitely.

### Fixed

- Fixed a nil-pointer panic when rendering a validation error on the `/v1/validate/job` response path.
- Webhook mutators and validators now close their HTTP response bodies, and share a single instrumented transport instead of allocating a new one per call.
- Gzip response handling now closes both the decompressor and the underlying upstream body, and no longer discards write and close errors.
- Non-2xx webhook responses are now rejected instead of being decoded as if they were successful.
- Upstream Nomad TLS now supports CA-only, client-certificate, and system-root configurations instead of requiring all three files.
- The server handles `SIGTERM` in addition to `SIGINT`, shuts down with a 30 second deadline, and propagates shutdown failures. A normal `http.ErrServerClosed` is no longer reported as a runtime failure.
- `ReadHeaderTimeout` and `IdleTimeout` are now configured on the HTTP server.
- `X-Forwarded-For` values are trimmed, and an unparseable `RemoteAddr` no longer yields an empty client IP.
- Notation defaults (`max_sig_attempts`) are now applied to nested `opa_rule` notation blocks on both validators and mutators, not just top-level validators.
- Fixed a data race on the Testcontainers log consumer used by the OpenTelemetry tests.

### Container image

- The release image now ships the system CA certificate bundle. The previous `scratch` image had none, so HTTPS webhooks, OPA bundle servers, and registries failed with `x509: certificate signed by unknown authority`.
- The certificate stage is a version-pinned Alpine image instead of `ubuntu:latest`, and the runtime image runs as numeric UID/GID `10001:10001` without requiring an `/etc/passwd` entry.

### Improvements

- Improved the controller handling of mutation jobs so it can be ensured that jobs mutations are aggregated
- Docker-backed Notation tests are behind an `integration` build tag, so `go test ./...` no longer requires a running Docker daemon. CI runs with `-tags=integration` to preserve the coverage.

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
