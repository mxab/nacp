---
type: Project Guide
title: NACP quickstart
description: Entry point for the Nomad Admission Control Proxy, covering local use, policy configuration, runtime behavior, operations, tests, and source navigation.
tags: [nomad, admission-control, opa, go]
openwiki:
  roles: [repository, workflow]
  change_kinds: [configuration, lifecycle, policy-adapter, operations]
  source_paths: [cmd/nacp/nacp.go, pkg/config/config.go, pkg/admissionctrl/controller.go]
  test_paths: [cmd/nacp/nacp_test.go, pkg/config/config_test.go, pkg/admissionctrl/controller_test.go]
  validation_commands: [go test ./pkg/config, go test ./pkg/admissionctrl, go test ./cmd/nacp]
---

# NACP quickstart

NACP is a Go reverse proxy in front of HashiCorp Nomad. It intercepts selected job-submission requests after Nomad clients have rendered HCL as JSON, applies configured mutations and validations, and forwards permitted jobs to Nomad. This wiki documents the current runtime contract, policy adapters, operations, and focused engineering paths; it does not replace Nomad or Rego documentation.

The [proxy and admission pipeline](architecture.md) owns request handling and controller ordering. It supplies the shared payload consumed by [policy integrations and configuration](policy-integrations.md); [operations and testing](operations-testing.md) covers deployment boundaries and validation; the [source map](source-map.md) locates source, tests, and examples.

## Start locally

The module declares Go **1.26.5** in `go.mod`. Build the executable, then run the first embedded-OPA example from its directory so its relative Rego path resolves correctly:

```bash
go build -o nacp ./cmd/nacp
cd example/example1
../../nacp -config example1.conf.hcl
```

In another terminal, point the Nomad CLI at NACP:

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run example1.nomad
```

Without `-config`, runtime defaults bind NACP to `0.0.0.0:6464` and proxy Nomad at `http://localhost:4646` (`pkg/config/config.go`). An explicitly supplied config is decoded and validated before the server starts.

Useful runnable references:

- `example/example1/` — embedded OPA cost-center validation.
- `example/example2/` — embedded OPA JSON-Patch metadata mutation.
- `example/demo/` — embedded OPA, remote webhooks, OPA SDK/bundle decisions, and telemetry.
- `example/notation/` — signed-container-image validation.

`mise.toml` pins Go, Nomad, OPA, Regal, and linters and provides `nomad`, `build-dev-nacp`, and `deploy-nacp` tasks for the demo environment.

## What NACP can enforce

- **Mutate** a Nomad job with local OPA JSON Patch, OPA SDK/bundle JSON Patch, or a JSON-Patch webhook.
- **Validate** with embedded OPA, an OPA SDK/bundle decision, a webhook, or Notation container-image verification.
- Supply policy input as `{job, context}`. Context can carry client IP and, where any controller enables `resolve_token`, sanitized Nomad ACL-token details.
- Merge policy warnings into successful Nomad responses. Admission errors block registration and planning locally; the Nomad validation endpoint receives validator errors in its rewritten response. See [endpoint outcomes](architecture.md#request-lifecycle-and-outcomes).

## Engineer task routing

Use the narrow package command first; `-count=1` avoids a cached success while keeping successful output quiet. Run `go test ./...` only after a cross-package change or before a broad handoff.

| Change area or user intent | Relevant wiki page | Exact source entry points | Important symbols or types | Focused tests | Minimal validation command |
| --- | --- | --- | --- | --- | --- |
| Intercept a route, rewrite request/response, change body or token boundary | [Proxy and admission pipeline](architecture.md) | `cmd/nacp/nacp.go` | `newProxyHandler`, `handleRegister`, `handlePlan`, `handleValidate` | `TestProxy`, `TestJobUpdateProxy`, `TestAdmissionRouteMatching` in `cmd/nacp/nacp_test.go` | `go test ./cmd/nacp -count=1` |
| Change mutation ordering, validator behavior, warnings, or errors | [Proxy and admission pipeline](architecture.md#controller-contract) | `pkg/admissionctrl/controller.go` | `JobHandler`, `ApplyAdmissionControllers`, `AdmissionMutators`, `AdmissionValidators` | `TestJobHandler_ApplyAdmissionControllers`, `TestJobHandler_ValidatorsReceiveIsolatedCopyOfMutatedJob` | `go test ./pkg/admissionctrl -count=1` |
| Add a controller type or change HCL startup validation | [Policy integrations and configuration](policy-integrations.md#configuration-and-startup-validation) | `pkg/config/config.go`, `cmd/nacp/nacp.go` | `Config.Validate`, `validateMutator`, `validateValidator`, `createMutators`, `createValidators` | `TestConfigValidation`, `TestCreateMutatators`, `TestCreateValidators` | `go test ./pkg/config ./cmd/nacp -count=1` |
| Change embedded OPA, bundle, webhook, patch, or Notation result handling | [Policy integrations and configuration](policy-integrations.md) | `pkg/admissionctrl/{opa,mutator,validator,notation,remoteutil}/` | adapter constructors, `types.Payload`, `DecodeJSONResponse` | corresponding colocated `*_test.go` | `go test ./pkg/admissionctrl/... -count=1` |
| Change listener/upstream TLS, telemetry, CI, or release | [Operations and testing](operations-testing.md) | `cmd/nacp/nacp.go`, `pkg/otel/otel.go`, `.github/workflows/`, `.goreleaser.yaml` | `buildServer`, `buildCustomTransport`, OTel setup | `TestCreateTlsConfig`, `TestBuildCustomTransport`, `TestOtelInstrumentation` | `go test ./cmd/nacp ./pkg/otel -count=1` |
| Find an example, fixture, or generated metric source | [Source map](source-map.md) | `example/`, `testdata/`, `pkg/o11y/nacp.yaml` | generated `pkg/o11y/metric.go` | adjacent package tests | `go test ./pkg/o11y/... -count=1` when generator/runtime code changes |

## Scope and safety

The proxy and controller order are the core behavioral contract. Preserve mutation-before-validation, configured slice order, fail-fast mutators, validator aggregation, and validator isolation. Configuration now validates listener, Nomad URL/TLS pairing, controller type/block requirements, duplicate names within a kind, and OPA SDK prerequisites before startup; use the canonical configuration page rather than stale assumptions.

## Backlog

- **Historical progression audit** — source anchor: `openwiki/.last-update.json`, `.git/`; the recorded prior `gitHead` `e10faf62eb238c28902bf7bbab4e5579750b0faf` is unavailable in this shallow checkout, so a source-backed range history cannot be reconstructed here.
- **OPA bundle production lifecycle** — source anchor: `pkg/admissionctrl/{validator,mutator}/opa_bundle_*.go`, `example/demo/opa.yml`; bundle availability, refresh, and rollout behavior need deployment-level evidence beyond the current code and examples.
