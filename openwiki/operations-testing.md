---
type: Operations Guide
title: Operations and testing
description: How to run, secure, observe, test, release, and safely evolve the NACP admission proxy.
tags: [operations, testing, telemetry, tls, release]
---

# Operations and testing

This page covers the operational envelope of the [proxy and admission pipeline](architecture.md), including the policy adapters it runs. Source of truth is the runtime and automation configuration, with practical file locations in the [source map](source-map.md).

## Transport and telemetry

NACP has separate TLS boundaries:

- Top-level `tls` config secures the NACP listener with certificate/key material and can verify client certificates using `ca_file` unless `no_client_cert` is set.
- `nomad { tls { ... } }` configures the upstream Nomad transport with CA/client certificate/key and optional `insecure_skip_verify`.

Keep those trust boundaries distinct when diagnosing failures: listener TLS controls whether the caller reaches NACP; upstream TLS controls whether NACP can forward an admitted request. The server and upstream transport use 310-second timeout values in `cmd/nacp/nacp.go`, a historical choice for long-running Nomad workflows.

Telemetry config independently enables logs, metrics, and tracing. `pkg/otel/otel.go` configures OTLP/HTTP exporters, resource fields including `service.name=nacp`, and W3C TraceContext/Baggage propagation. `pkg/logutil/logutil.go` fans `slog` records to text, JSON, and OTel handlers. `pkg/o11y/nacp.yaml` defines controller-labeled warning/error/mutation metrics generated into `pkg/o11y/metric.go`.

The runtime wraps inbound HTTP and controller work in instrumentation, so telemetry follows the flow documented in [architecture](architecture.md). Use `example/otel/` for a policy-focused demonstration and `pkg/otel/otel_test.go` plus `cmd/nacp/nacp_otel_test.go` to verify code-level exporter/instrumentation behavior.

## Verification workflow

CI in `.github/workflows/go.yml` runs on pushes and pull requests to `main` using Go `^1.26.5`:

```bash
test -z "$(gofmt -l .)"
go build -v ./...
go test -coverprofile=cov.all.out -v ./...
grep -v 'o11y/metric.go' cov.all.out > cov.out
go tool cover -func=cov.out
```

SonarCloud analysis follows those steps. Run the equivalent format/build/test checks before changing production paths. The coverage post-processing intentionally removes generated `o11y/metric.go`.

| Change area | Tests to start with | Key behavior |
| --- | --- | --- |
| Routes, request/response rewrite, gzip, TLS, token lookup | `cmd/nacp/nacp_test.go`, `cmd/nacp/nacp_otel_test.go` | Admission boundary and Nomad-compatible responses |
| Ordering and aggregation | `pkg/admissionctrl/controller_test.go` | Chained mutation, validation ordering, errors/warnings |
| HCL schema/defaults | `pkg/config/config_test.go` | Default configuration, controller decoding, telemetry |
| OPA, bundles, patching, webhooks, Notation | colocated `pkg/admissionctrl/**/*_test.go` | Per-adapter result and failure contract |
| OTel providers/export | `pkg/otel/otel_test.go` | Exporter and provider setup |

Notation tests can exercise registry/signature integration; troubleshoot those dependencies rather than weakening broad coverage.

## Demo and developer workflow

`example/readme.md` maps demonstrations for cost-center validation, metadata mutation, PostgreSQL/Vault injection, and oauth2-proxy injection. The supporting Nomad/Vault/Terraform files in `example/infra/` are development infrastructure; the examples explicitly are not production manifests.

`mise.toml` pins developer tools (Go, Nomad, OPA, Regal, and linters) and supplies tasks to start a Nomad dev agent, build a development NACP image, and deploy the demo job. The [policy integrations guide](policy-integrations.md) explains the policy contracts exercised by those examples.

## Release and artifact delivery

`.github/workflows/release.yml` runs for pushed tags, imports a GPG key from GitHub Actions secrets, logs in to GHCR, and invokes GoReleaser. `.goreleaser.yaml` runs `go mod tidy`, builds static binaries from `./cmd/nacp` for Linux, Windows, and macOS, creates/signs checksums, and publishes `ghcr.io/mxab/nacp:latest` plus tag images.

The repository `Dockerfile` packages `/nacp` in a `scratch` image under a non-root UID. This is distinct from `dev.Dockerfile`, which supports development image builds. Release changes require reviewing both workflow and GoReleaser configuration; do not expose or record the workflow’s signing credentials.

## Maintenance notes

- **Source over stale prose:** current HCL fields are in `pkg/config/config.go`; the README’s `server {}` structure and an older changelog telemetry sample do not match the current model.
- **Token-context trust:** policies may use first-hop `X-Forwarded-For` and resolved ACL data. Deploy NACP behind trusted proxy layers and recognize that requesting token resolution adds a Nomad self-token lookup to actionable routes.
- **Bundle maturity:** OPA SDK/bundle support is recent and marked as a working POC in its latest feature commit. Exercise configured decision paths, readiness behavior, and degraded conditions in the target environment.
- **Wiki automation:** `.github/workflows/openwiki-update.yml` schedules an OpenWiki refresh and creates a documentation PR. It uses repository-managed CI secrets; generated content belongs under `openwiki/`.

The [source map](source-map.md) provides exact engineering starting points; the [architecture](architecture.md) supplies the behavioral invariants those paths must preserve.
