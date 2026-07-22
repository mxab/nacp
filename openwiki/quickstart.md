---
type: Project Guide
title: NACP quickstart
description: "Entry point for the Nomad Admission Control Proxy: local use, architecture, policy integrations, operations, tests, and source navigation."
tags: [nomad, admission-control, opa, go]
---

# NACP quickstart

NACP is a Go reverse proxy in front of HashiCorp Nomad. It intercepts selected job-submission requests after Nomad clients have rendered HCL as JSON, applies configured mutations and validations, and forwards permitted requests to Nomad. Its purpose is to centralize job policy without modifying every Nomad client or embedding operational rules in each job.

The [proxy and admission pipeline](architecture.md) is the core runtime concept. It dispatches decoded jobs to the policy mechanisms in [policy integrations](policy-integrations.md), and its transport, telemetry, release, and verification practices are documented in [operations and testing](operations-testing.md). Use the [source map](source-map.md) to locate the implementation, tests, fixtures, and examples behind those concepts.

## Start locally

The module declares Go **1.26.5** in `go.mod`. Build and test before running the executable:

```bash
test -z "$(gofmt -l .)"
go build ./...
go test ./...
./nacp -config config.hcl
```

If `nacp` has not been built at the repository root, use `go run ./cmd/nacp -config config.hcl` instead. Without a config file, runtime defaults bind NACP to `0.0.0.0:6464` and proxy Nomad at `http://localhost:4646` (`pkg/config/config.go`). Point the Nomad CLI at NACP for a job operation:

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run job.hcl
```

Start from a runnable example rather than an older README configuration snippet:

- `example/example1/` — embedded OPA cost-center validation.
- `example/example2/` — embedded OPA JSON-Patch metadata mutation.
- `example/demo/` — Nomad deployment with embedded OPA, remote webhooks, OPA SDK/bundle decisions, and telemetry.
- `example/notation/` — signed-container-image validation.

The repository’s Go-managed developer tools and convenience tasks are in `mise.toml`; its `nomad`, `build-dev-nacp`, and `deploy-nacp` tasks support the demo environment.

## What NACP can enforce

- **Mutate** a Nomad job using local OPA JSON Patch, OPA SDK/bundle JSON Patch, or a JSON-Patch webhook.
- **Validate** a job using local OPA, OPA SDK/bundle decisions, a webhook, or Notation container-image verification.
- Supply policy input as `{job, context}`. Context can include client IP and, when a controller enables `resolve_token`, Nomad ACL-token details.
- Merge policy warnings into Nomad responses. Admission errors block registration and planning locally; the Nomad validation endpoint instead receives policy validation errors in its rewritten response. Exact endpoint semantics are in [architecture](architecture.md#request-lifecycle-and-outcomes).

## Engineer orientation

| If you need to… | Start at | Then verify |
| --- | --- | --- |
| Change an intercepted route or Nomad request/response rewriting | `cmd/nacp/nacp.go` and [architecture](architecture.md) | `cmd/nacp/nacp_test.go`, including gzip cases |
| Change admission ordering or controller contracts | `pkg/admissionctrl/controller.go` | `pkg/admissionctrl/controller_test.go` |
| Add/change policy configuration or output contracts | `pkg/config/config.go` and [policy integrations](policy-integrations.md) | config and focused adapter tests |
| Operate TLS, telemetry, CI, or release delivery | [operations and testing](operations-testing.md) | `.github/workflows/`, `.goreleaser.yaml` |
| Find a fixture or a working policy example | [source map](source-map.md) | `testdata/` and `example/` |

## Repository progression

Recent history shows a useful evolution path: major packages moved under `pkg/`; request context and token resolution enriched policy input; OpenTelemetry and structured logging were added; then OPA SDK, bundle validation, and bundle JSON-Patch support arrived. The current branch’s `feat(bundle): working poc` commit makes bundle behavior especially important to verify against `example/demo/` and its focused adapter tests instead of assuming a mature operational contract.

## Documentation caution

Prefer current source and runnable examples over legacy prose. In particular, `pkg/config/config.go` expects top-level `bind`, `port`, and `tls`; the README still contains an older `server {}` wrapper. The [operations page](operations-testing.md#maintenance-notes) lists other drift and production-readiness cautions.

## Backlog

- **Exhaustive HCL reference** — source anchor: `pkg/config/config.go`; deferred because controller sub-block requirements are enforced mainly during startup construction and need a dedicated schema audit.
- **OPA bundle production lifecycle** — source anchor: `pkg/admissionctrl/{validator,mutator}/opa_bundle_*.go`, `example/demo/opa.yml`; deferred because the latest bundle feature is explicitly a working POC and does not document refresh, availability, or rollout expectations.
