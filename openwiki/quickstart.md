---
type: Project Guide
title: NACP quickstart
description: Entry point for the Nomad Admission Control Proxy: its purpose, supported policy mechanisms, local operation, and documentation map.
tags: [nomad, admission-control, go, opa]
---

# NACP quickstart

NACP is a Go reverse proxy in front of HashiCorp Nomad. It intercepts selected job submission APIs after Nomad clients have converted HCL jobs to JSON, applies configured mutations and validations, then forwards acceptable requests to Nomad. It exists to centralize policy enforcement without changing Nomad clients or embedding policy rules in every job.

The runtime is centered on the [proxy and admission pipeline](architecture.md): it handles job registration, planning, and validation requests while preserving ordinary Nomad traffic as proxy traffic. Policy implementations and their HCL configuration are documented in [policy integrations](policy-integrations.md). Serving, observability, CI, and change guidance live in [operations and testing](operations-testing.md).

## What it can enforce

- **Mutate jobs** before validation using embedded OPA JSON Patch, OPA SDK/bundle JSON Patch, or a JSON Patch webhook.
- **Validate jobs** using embedded OPA, OPA SDK/bundle decisions, a webhook, or Notation image-signature verification.
- Carry request context—client IP and, when requested by a controller, resolved Nomad ACL-token information—inside the policy payload.
- Return policy warnings alongside Nomad responses; mutations and validation errors have endpoint-specific behavior described in the [request lifecycle](architecture.md#request-lifecycle-and-outcomes).

## Run locally

Build and test with the Go toolchain declared in `go.mod` (the current module declares Go 1.26.5):

```bash
go build ./...
go test ./...
nacp -config config.hcl
```

The code defaults to `0.0.0.0:6464` and upstream Nomad at `http://localhost:4646`. Point a Nomad CLI at the proxy:

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run job.hcl
```

Use one of the HCL configurations under `example/` as a starting point rather than assuming every README snippet matches current configuration. For a simple embedded-OPA setup, see `example/example1/` or `example/example2/`; `example/demo/nacp.conf` combines OPA, webhook, SDK/bundle, and telemetry features.

## Source landmarks

| Area | Start here | Why it matters |
| --- | --- | --- |
| Executable and HTTP boundary | `cmd/nacp/nacp.go` | Builds controllers, creates the reverse proxy, manages TLS, and rewrites supported requests/responses. |
| Admission orchestration | `pkg/admissionctrl/controller.go` | Defines controller interfaces and mutation-before-validation ordering. |
| Configuration contract | `pkg/config/config.go` | Defines HCL blocks, defaults, and the request-context schema. |
| Policy implementations | `pkg/admissionctrl/{opa,mutator,validator,notation,remoteutil}/` | Contains OPA, SDK/bundle, webhook, JSON Patch, and signature-verification behavior. |
| Observability | `pkg/otel/otel.go`, `pkg/logutil/logutil.go`, `pkg/o11y/` | Sets up OTLP exports, slog handlers, and generated controller metrics. |
| Tests and examples | `cmd/nacp/*_test.go`, `pkg/**/*_test.go`, `example/` | Establishes expected proxy, policy, and integration behavior. |

## Change orientation

- Changes to intercepted endpoints or Nomad request/response serialization begin in `cmd/nacp/nacp.go` and need proxy tests, including gzip-response cases where applicable.
- Changes to policy ordering, warning aggregation, or controller interfaces begin in `pkg/admissionctrl/controller.go`; preserve the invariant that mutators execute before validators.
- Changes to HCL shape must update `pkg/config/config.go`, its tests, and the relevant runnable example. The current README has configuration drift noted in [operations and testing](operations-testing.md#documentation-and-maintenance-notes).
- CI runs formatting, `go build -v ./...`, and full package tests with coverage; use the same checks before changing production paths.

## Backlog

- **Configuration reference** — `pkg/config/config.go`; deferred because a canonical, exhaustive HCL reference would require auditing each controller constructor and every example. This first pass points to the source schema and documents the supported integration families.
- **Release/distribution workflow** — `.goreleaser.yaml`, `Dockerfile`, `.github/workflows/release.yml`; deferred to retain a focused initial wiki centered on the admission runtime.
