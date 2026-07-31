# Changes in this branch

This branch follows a project-wide review of NACP's admission pipeline, policy integrations, configuration handling, runtime security, tests, CI/release setup, container image, and maintained documentation.

## Admission pipeline correctness

### Validators now receive the final mutated job

`pkg/admissionctrl/controller.go` now passes the result of the mutator chain into validation. Previously, `ApplyAdmissionControllers` ran validators with the original payload even though the mutators returned an updated job.

Validators now receive isolated copies of the final mutated job. This ensures that:

- Validators evaluate the exact job that will be forwarded to Nomad.
- A validator cannot mutate the forwarded job.
- One validator cannot alter the input seen by another validator.
- Missing payloads or jobs return controlled errors instead of causing nil-pointer panics.

Regression tests in `pkg/admissionctrl/controller_test.go` verify that validators see mutations, validator-side modifications do not leak, and missing jobs are rejected safely.

### Malformed job handling

Tracing and logging no longer assume that every incoming job has a non-nil ID. Admission helpers use safe job-ID access and reject missing jobs with an actionable error rather than dereferencing nil pointers.

## Token resolution and policy context

### Token lookup failures stop admission

When a controller enables `resolve_token`, failure of Nomad's `/v1/acl/token/self` lookup now immediately stops request processing. NACP no longer writes an error and then continues through admission or upstream proxying without the required identity context.

A regression assertion in `cmd/nacp/nacp_test.go` verifies that the Nomad admission endpoint is not called after token resolution fails.

### ACL token context is sanitized

`pkg/config/config.go` introduces `ACLTokenContext` and `SanitizeACLToken`. Policy input no longer serializes the complete `api.ACLToken` object.

The sanitized token context retains policy-relevant metadata such as:

- Accessor ID
- Name and type
- Policies and role links
- Global status
- Creation and expiration information
- Nomad indexes

It deliberately excludes `SecretID` and other unnecessary token-creation fields. Tests verify that the serialized policy context cannot contain the original secret.

`RequestContext.ResolveToken` is now populated to reflect whether token resolution was requested.

## OPA SDK and bundle validation

`pkg/admissionctrl/validator/opa_bundle_validator.go` was rewritten to parse bundle decisions strictly and with lower cognitive complexity.

Changes include:

- Non-object decisions now fail closed instead of silently passing validation.
- Invalid `errors` and `warnings` collections are rejected as policy contract failures.
- Invalid individual entries are rejected.
- Policy warnings are preserved when validation errors are also returned.
- Message parsing is shared through a focused helper.

The bundle validator tests now cover malformed decisions, malformed message collections, combined errors and warnings, and non-object result values.

## Webhook reliability and security

### Shared HTTP behavior

`pkg/admissionctrl/remoteutil/remoteutil.go` now provides common webhook behavior:

- A shared instrumented transport instead of constructing a new transport for every call.
- A 30-second request timeout.
- A 10 MiB response-body limit.
- Strict 2xx status enforcement.
- Bounded JSON response decoding.
- Validation that webhook endpoints are absolute HTTP or HTTPS URLs.

### Adapter hardening

The validation webhook, JSON-Patch webhook mutator, and legacy webhook mutator now:

- Close every HTTP response body.
- Send consistent `Content-Type: application/json` and `Accept: application/json` headers.
- Reject non-success HTTP responses.
- Use bounded shared response decoding.
- Return contextual errors for malformed responses.

The JSON-Patch webhook mutator now enforces its documented `errors` response field. Previously, webhook errors were parsed into the response structure but ignored, allowing a rejected mutation to continue.

Tests cover policy-declared mutation errors, non-2xx responses, and JSON request headers.

## Configuration validation

`pkg/config/config.go` now provides a comprehensive `Config.Validate` preflight. Configuration is validated after HCL decoding and again when constructing a server programmatically.

Validation covers:

- Port range and non-empty bind address.
- Required Nomad configuration and an absolute HTTP(S) Nomad address.
- Listener TLS certificate/key requirements.
- Upstream TLS certificate/key pairing.
- OPA SDK ID and configuration path.
- Known mutator and validator types.
- Required `opa_rule`, `opa_sdk_rule`, `webhook`, and `notation` blocks.
- Required OPA filename/query and bundle decision paths.
- Absolute HTTP(S) webhook endpoints and valid HTTP methods.
- Required Notation trust policy/store paths and positive signature-attempt limits.
- Duplicate mutator and validator names.
- Bundle controllers requiring a configured OPA SDK.

Notation defaults are now applied consistently to top-level validators and nested OPA-rule Notation configurations, including mutators.

Controller factories retain defensive nil checks so invalid programmatic configurations return errors rather than panicking.

An explicitly supplied but missing `-config` path now fails startup. Defaults are used only when no config path is supplied.

Tests cover invalid ports, missing webhook blocks, bundle controllers without an SDK, relative webhook URLs, and missing explicit configuration files.

## Proxy and runtime hardening

Changes in `cmd/nacp/nacp.go` include:

- Typed context keys replace the string key used for request context.
- `X-Forwarded-For` values are trimmed before use.
- Invalid `RemoteAddr` values fall back safely.
- Token lookups have an explicit timeout.
- Admission request bodies are limited to 32 MiB.
- Job update and plan route matching supports any non-slash Nomad job ID segment, including IDs beginning with numbers or containing underscores and dots.
- Route tests cover broader valid IDs and reject unrelated nested endpoints.
- Response rewriting is limited to successful upstream responses so Nomad error bodies are preserved.
- A latent validation-response panic now uses the actual validation error instead of a nil local error.
- Gzip readers now close both the decompressor and original upstream response body.
- Gzip writes and closes return errors instead of being ignored.
- Listener and upstream TLS configurations require TLS 1.2 or newer.
- Invalid PEM CA files are rejected rather than creating empty trust pools silently.
- Upstream TLS now supports CA-only, client-certificate, and system-root configurations instead of always requiring all three files.
- The server handles both `SIGINT` and `SIGTERM`.
- Graceful shutdown has a 30-second deadline and propagates shutdown failures.
- `ReadHeaderTimeout` and `IdleTimeout` are configured.
- Normal `http.ErrServerClosed` shutdown is not reported as a runtime failure.

## Notation

- Removed an impossible/dead error branch in `pkg/admissionctrl/notation/notation.go`.
- Docker-backed Notation tests now use the `integration` build tag so the default local test suite does not require Docker.
- CI explicitly enables the integration tag, preserving full integration coverage.

## OpenTelemetry test reliability

Race testing exposed concurrent access to Testcontainers log slices in `testutil/otel.go`.

The log consumer now:

- Protects log state with an RW mutex.
- Keeps internal slices private.
- Provides a synchronized `Contains` query.

`pkg/otel/otel_test.go` uses this synchronized API. The focused race suite now passes.

## Container image

`Dockerfile` now:

- Uses a version-pinned Alpine certificate stage instead of `ubuntu:latest`.
- Installs and copies the system CA certificate bundle into the scratch runtime image.
- Continues to use a minimal scratch final image.
- Runs as numeric non-root UID/GID `10001:10001` without requiring an `/etc/passwd` entry.

Including CA certificates allows HTTPS webhooks, OPA bundle services, registries, and upstream Nomad endpoints using public roots to work in the release image.

`.dockerignore` was cleaned up to exclude repository metadata, editor state, coverage/build output, examples, generated documentation, tests, templates, and test fixtures from the Docker build context.

## CI and automation

### Go workflow

`.github/workflows/go.yml` now:

- Uses explicit read-only repository permissions.
- Correctly quotes the formatting check.
- Runs `go vet ./...`.
- Runs tests with `-tags=integration` so Docker-backed Notation tests execute in CI.
- Waits for the Sonar quality gate instead of only submitting an analysis.

### OpenWiki workflow

OpenWiki remains pinned to version `0.2.4`.

The workflow intentionally allows npm lifecycle scripts because OpenWiki depends on `better-sqlite3`, whose native SQLite module must be built during installation. A workflow comment documents why `--ignore-scripts` cannot be used for this trusted pinned dependency. The corresponding Sonar finding should be treated as an intentional reviewed exception rather than addressed by breaking OpenWiki installation.

Generated files under `openwiki/` were not edited manually. They should be refreshed by the OpenWiki automation after source and maintained documentation changes land.

## README and maintained documentation

### Root README

`README.md` was replaced with a current onboarding and operations guide covering:

- What NACP does.
- Every supported mutator and validator integration.
- The admission payload and sanitized token context.
- Mutation and validation ordering.
- Intercepted Nomad endpoints and failure behavior.
- Installation from releases, Go, and GHCR.
- A runnable quickstart.
- Current top-level HCL syntax without the obsolete `server` wrapper.
- Configuration validation behavior.
- TLS and OpenTelemetry configuration.
- Security and availability guidance.
- Unit versus Docker-backed integration testing.
- Links to OpenWiki, examples, contribution guidance, security policy, and changelog.

The broken payload source link and obsolete feature/configuration claims were removed.

### Other documentation

- `CHANGELOG.md` now shows the current nested `otel { enabled = true }` logging syntax.
- `example/readme.md` fixes grammar, broken anchors, incorrect example configuration paths, incorrect example 4 references, and Nomad CLI commands.
- Added `CONTRIBUTING.md` with development checks, test organization, pull-request expectations, and OpenWiki guidance.
- Added `SECURITY.md` with private reporting guidance and deployment-boundary recommendations.

## Validation performed

The following checks completed successfully after the changes:

```text
gofmt -l .
go vet ./...
go build ./...
go test ./...
go test -tags=integration ./...
go test -tags=integration ./pkg/admissionctrl/notation -count=1
go test -race ./cmd/nacp ./pkg/admissionctrl ./pkg/admissionctrl/mutator/... ./pkg/admissionctrl/opa ./pkg/admissionctrl/remoteutil ./pkg/admissionctrl/validator ./pkg/config ./pkg/logutil ./pkg/otel
mise exec -- actionlint
go mod tidy -diff
git diff --check
```

Local Markdown links in `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, and `example/readme.md` were also checked and no broken local links were found.

The editor may report that `pkg/admissionctrl/notation/notation_test.go` is excluded when gopls is not configured with `-tags=integration`; this is expected for the opt-in integration test file, not a code error.

## Repository state note

The pre-existing untracked `.zed/` directory was not modified as part of this work.
