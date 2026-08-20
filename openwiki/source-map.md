---
type: Source Map
title: NACP source map
description: Practical navigation map for NACP runtime code, policy adapters, configuration, observability, tests, fixtures, and examples.
tags: [source-map, go, testing, opa, nomad]
openwiki:
  roles: [repository, testing]
  change_kinds: [navigation, configuration, policy-adapter]
  source_paths: [cmd/nacp/nacp.go, pkg/config/config.go, pkg/admissionctrl/controller.go]
  test_paths: [cmd/nacp/nacp_test.go, pkg/config/config_test.go, pkg/admissionctrl/controller_test.go]
---

# NACP source map

Use this page to jump from a question to the smallest source area that can answer it. The structural relationships among these areas are explained in [architecture](architecture.md), while supported policy contracts are canonical in [policy integrations](policy-integrations.md).

## Runtime and admission

| Area | Key paths | Use it for |
| --- | --- | --- |
| Executable and proxy | `cmd/nacp/nacp.go` | CLI startup, config wiring, TLS server/transport, route matching, Nomad request decoding, response rewriting, token lookup, and shutdown. |
| Admission orchestration | `pkg/admissionctrl/controller.go` | Mutator/validator interfaces, order, aggregation, telemetry counters, and traces. |
| Shared policy input | `pkg/admissionctrl/types/opa_payload.go` | The exact `{job, context}` schema sent to controllers. |
| HTTP-level regression tests | `cmd/nacp/nacp_test.go`, `cmd/nacp/nacp_otel_test.go` | Routes, register/plan/validate semantics, warning/error injection, gzip, TLS, token handling, and instrumentation. |

The runtime code implements the ordered lifecycle in [architecture](architecture.md#request-lifecycle-and-outcomes). Begin here for a change that affects Nomad compatibility or whether traffic is admitted at all.

## Policy and configuration

| Area | Key paths | Use it for |
| --- | --- | --- |
| HCL schema/defaults/validation | `pkg/config/config.go`, `pkg/config/config_test.go`, `pkg/config/testdata/` | Config structure, defaults, startup validation, duplicate-name rules, and decoding tests. |
| Embedded OPA query engine | `pkg/admissionctrl/opa/` | Rego module loading, query evaluation, result binding extraction, Notation builtin registration. |
| JSON Patch implementation | `pkg/admissionctrl/mutator/opa_json_patch.go`, `pkg/admissionctrl/mutator/jsonpatcher/` | Embedded OPA patching and patch application. |
| OPA SDK/bundle adapters | `pkg/admissionctrl/{validator,mutator}/opa_bundle_*.go` | SDK decision result handling for validation and patch mutation. |
| Remote policy adapters | `pkg/admissionctrl/{validator,mutator}/*webhook*.go`, `pkg/admissionctrl/remoteutil/` | Payload serialization, response parsing, forwarded context headers, outbound instrumentation. |
| Image verification | `pkg/admissionctrl/notation/`, `pkg/admissionctrl/notation/notation_test.go`, `pkg/admissionctrl/validator/notation_validator.go`, `pkg/admissionctrl/validator/notation_validator_test.go` | Trust policy/store loading, remote verification, Docker-task selection, and the Docker-backed registry integration test. |

These paths implement the controller types listed in [policy integrations](policy-integrations.md#configuration-and-startup-validation). Keep result shape tests adjacent to the adapter that interprets it. The Docker-backed registry test is an operational, conditional check; its prerequisites and scope are defined in [operations and testing](operations-testing.md#notation-validation-dependency-boundary).

## Operations and generated observability

| Area | Key paths | Use it for |
| --- | --- | --- |
| OpenTelemetry setup | `pkg/otel/` | OTLP exporters, resource metadata, propagation, and tests. |
| Logging | `pkg/logutil/` | slog formatting/output fan-out. |
| Metric definition/generated code | `pkg/o11y/nacp.yaml`, `pkg/o11y/metric.go`, `templates/registry/` | Metric schema, generated counters, and generator templates. |
| Build/test CI | `.github/workflows/go.yml` | Required formatting, build, test/coverage, and Sonar checks. |
| Release | `.github/workflows/release.yml`, `.goreleaser.yaml`, `Dockerfile` | Tag release, signing, binary archives, GHCR publishing, minimal runtime image. |
| Local tooling | `mise.toml`, `dev.Dockerfile` | Pinned tools and demo/development container workflow. |

These files operationalize the proxy rather than change admission semantics; see [operations and testing](operations-testing.md) for their runbook implications.

## Fixtures and examples

| Area | Key paths | What it demonstrates |
| --- | --- | --- |
| Unit/integration test assets | `testdata/`, `testutil/` | Job requests and Rego inputs used by proxy and policy tests. |
| Basic embedded OPA | `example/example1/`, `example/example2/` | Cost-center validation and metadata JSON-Patch mutation. |
| Rich mutation samples | `example/example3/`, `example/example4/` | Postgres/Vault template injection and oauth2-proxy injection. |
| Combined Nomad demo | `example/demo/` | NACP Nomad job, templated config, mock webhooks, OPA SDK config, and bundle policies. |
| Notation demo | `example/notation/` | Registry/trust/signature validation sequence. |
| Telemetry demo | `example/otel/` | OTel-oriented mutation configuration. |
| Supporting development environment | `example/infra/` | Nomad/Vault/Terraform setup referenced by older feature examples. |

Examples are capability demonstrations, not production deployment manifests (`example/readme.md`). They connect policy behavior to [policy integrations](policy-integrations.md) and should be updated alongside contract changes when they remain representative.

## History navigation

Use narrow history queries rather than reading unrelated commits when the checkout contains history:

```bash
git log -- cmd/nacp/nacp.go pkg/admissionctrl/controller.go
git log -- pkg/config/config.go pkg/admissionctrl/mutator/opa_bundle_json_patch.go
```

This checkout is shallow: the prior wiki `gitHead` is unavailable locally. Do not depend on historical commit IDs from this page; verify lifecycle and configuration claims directly against the current source and tests until a fuller clone is available. The pipeline's request context and two OPA execution models are documented by [architecture](architecture.md) and [policy integrations](policy-integrations.md).
