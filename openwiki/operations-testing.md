---
type: Operations Guide
title: Operations and testing
description: How to run, secure, observe, test, release, and safely evolve the NACP admission proxy.
tags: [operations, testing, telemetry, tls, release]
openwiki:
  roles: [operations, testing, delivery]
  change_kinds: [tls, telemetry, generated-artifact, release]
  source_paths: [cmd/nacp/nacp.go, pkg/otel/otel.go, pkg/o11y/nacp.yaml, .github/workflows/go.yml, .goreleaser.yaml]
  test_paths: [cmd/nacp/nacp_test.go, cmd/nacp/nacp_otel_test.go, pkg/otel/otel_test.go]
  validation_commands: [go test ./cmd/nacp ./pkg/otel -count=1, go test -tags=integration ./pkg/admissionctrl/notation]
---

# Operations and testing

This page covers the operational envelope of the [proxy and admission pipeline](architecture.md), including the policy adapters configured in [policy integrations](policy-integrations.md). Runtime and automation configuration are source of truth; use the [source map](source-map.md) for exact locations.

## Transport and telemetry

NACP has separate TLS boundaries:

- Top-level `tls` config secures the NACP listener with certificate/key material and can verify client certificates using `ca_file` unless `no_client_cert` is set.
- `nomad { tls { ... } }` configures the upstream Nomad transport with CA/client certificate/key and optional `insecure_skip_verify`.

Keep these boundaries distinct: listener TLS controls whether callers reach NACP, while upstream TLS controls whether NACP forwards admitted requests. `cmd/nacp/nacp.go` sets 310-second server and Nomad-transport timeouts. Its token-resolution subrequest is separately limited to 30 seconds.

Telemetry independently enables logs, metrics, and tracing. `pkg/otel/otel.go` configures OTLP/HTTP exporters, resource attributes including `service.name=nacp`, and W3C TraceContext/Baggage propagation. `pkg/logutil/logutil.go` fans `slog` records to text, JSON, and OTel handlers. `pkg/o11y/nacp.yaml` defines controller-labeled warning/error/mutation metrics used by `JobHandler`; `pkg/o11y/metric.go` is generated output.

## Focused verification

Run the smallest relevant package test without cached results. The full repository command is deliberately not the default for a local, isolated change.

| Change area | Start with | Narrow command | Escalate when |
| --- | --- | --- | --- |
| Routes, request/response rewrite, gzip, TLS, token lookup | `cmd/nacp/nacp_test.go` | `go test ./cmd/nacp -count=1` | a proxy change also alters controller lifecycle or telemetry |
| Controller sequencing and adapter contracts | `pkg/admissionctrl/controller_test.go` and colocated adapter tests | `go test ./pkg/admissionctrl/... -count=1` | shared configuration or executable wiring changes |
| HCL schema/defaults/startup failures | `pkg/config/config_test.go` | `go test ./pkg/config -count=1` | controller construction is changed too |
| OTel provider/export and HTTP instrumentation | `pkg/otel/otel_test.go`, `cmd/nacp/nacp_otel_test.go` | `go test ./pkg/otel ./cmd/nacp -count=1` | exporter behavior or metrics schema changes |
| Notation task selection/errors | `TestNotationValidatorValidate` in `pkg/admissionctrl/validator/notation_validator_test.go` | `go test ./pkg/admissionctrl/validator -count=1` | the verifier, registry, trust-store, credential-store, or image-client boundary changes |
| Notation registry/signature behavior | `TestVerifyImage` in `pkg/admissionctrl/notation/notation_test.go` | `go test -tags=integration ./pkg/admissionctrl/notation` | the image verification contract crosses packages |

### Notation validation dependency boundary

The production verifier in `pkg/admissionctrl/notation/notation.go` uses Notation and ORAS to verify a remote artifact; it does not use a Docker client. Its Docker-backed integration test, `TestVerifyImage`, launches a registry, builds and pushes an image, signs it, and verifies both unauthenticated and authenticated registry cases. That test imports the split Moby API/client modules declared in `go.mod` (`github.com/moby/moby/api` and `github.com/moby/moby/client`), which replace the legacy `github.com/docker/docker` module named by the current visible dependency-change commit.

For a validator-only change, run the ordinary validator package test first. Run the integration command only when changing registry/signature behavior, the test's Docker/Moby client boundary, or the dependency declarations; it requires a usable Docker environment. The result then feeds the `notation` controller contract described in [policy integrations](policy-integrations.md#notation-image-verification).

CI in `.github/workflows/go.yml` runs format, `go vet ./...`, build, and `go test -tags=integration -coverprofile=cov.all.out -v ./...`, then excludes generated `o11y/metric.go` from the coverage report before SonarCloud analysis. `sonar-project.properties` also excludes that generated source through `**/o11y/metric.go`. Use that broader path before a cross-package or release handoff, not as the first check for a narrow change.

## Generated metrics

`pkg/o11y/metric.go` is derived from the registry schema under `pkg/o11y/`; `generate.go` declares `go generate` with `weaver registry generate --registry ./o11y go --future ./o11y`. For a metric change, edit the schema, regenerate, and review the generated output alongside the consumer in `pkg/admissionctrl/controller.go`. Do not hand-edit the generated file. Then run the controller and OTel-focused tests; add a broader test only if the generated API or instrumentation wiring changes across packages.

## Demo and developer workflow

`example/readme.md` maps demonstrations for cost-center validation, metadata mutation, PostgreSQL/Vault injection, and oauth2-proxy injection. The Nomad/Vault/Terraform material in `example/infra/` is development infrastructure, not production manifests. `mise.toml` pins developer tools and supplies tasks to start a Nomad dev agent, build a development NACP image, and deploy the demo job.

## Release and artifact delivery

`.github/workflows/release.yml` runs for pushed tags, imports its GPG key from GitHub Actions secrets, logs in to GHCR, and invokes GoReleaser. `.goreleaser.yaml` runs `go mod tidy`, builds static `./cmd/nacp` binaries for Linux, Windows, and macOS, creates/signs checksums, and publishes `ghcr.io/mxab/nacp:latest` plus tag images.

`Dockerfile` packages `/nacp` in a `scratch` image under a non-root UID. This differs from `dev.Dockerfile`, which supports development image builds. Release changes must review workflow and GoReleaser configuration together; do not record or expose signing credentials.

## Change navigation and operational cautions

- Consult [architecture](architecture.md) before changing a timeout or transport behavior that could alter admission outcomes; request and token paths are fail-closed on their errors.
- Policies may consume first-hop `X-Forwarded-For` and sanitized ACL data. Put NACP behind trusted proxy layers that sanitize forwarding headers.
- Webhooks have a 30-second client timeout and a 10 MiB response cap; their detailed failure contract is canonical in [policy integrations](policy-integrations.md#webhooks).
- Bundle support has examples but no code-backed production refresh/availability guarantee. Validate readiness and degraded behavior in the target environment.
- The scheduled OpenWiki workflow creates documentation PRs. Generated wiki content belongs under `openwiki/`.
