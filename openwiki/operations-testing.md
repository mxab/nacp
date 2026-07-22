---
type: Operations Guide
title: Operations, observability, and testing
description: Practical guidance for running NACP, configuring transport and telemetry, validating changes, and navigating documentation drift.
tags: [operations, testing, telemetry, tls, go]
---

# Operations, observability, and testing

This page covers the operational envelope around the [proxy and admission pipeline](architecture.md) and the policy integrations it hosts. Current behavior is grounded in `cmd/nacp/nacp.go`, `pkg/otel/otel.go`, `pkg/logutil/logutil.go`, and CI configuration.

## Run and transport configuration

The executable accepts `-config` and `-bootstrap-json-logger`; normal operation is:

```bash
nacp -config config.hcl
```

It listens on the configured `bind`/`port` (defaults: `0.0.0.0:6464`) and reverse-proxies to `nomad.address` (default: `http://localhost:4646`). The command-line invocation in [quickstart](quickstart.md#run-locally) sends Nomad job operations through that listener.

Configuration supports two distinct TLS boundaries:

- Top-level `tls` config controls the NACP listener certificate/key and an optional CA/client-certificate requirement.
- `nomad { tls { ... } }` controls NACP’s upstream transport with CA, client certificate/key, and optional `insecure_skip_verify`.

Keep serving and upstream trust decisions distinct when debugging connectivity. Admission is only reached after a request arrives at NACP; upstream TLS affects whether an admitted request can be forwarded to Nomad.

## Observability

Telemetry configuration separately controls logs, metrics, and tracing. `pkg/otel/otel.go` uses OTLP/HTTP exporters configured through standard OpenTelemetry environment variables, sets `service.name=nacp` plus build version, and enables W3C TraceContext/Baggage propagation. `pkg/logutil/logutil.go` can fan logging to text, JSON, and OTel handlers using `slog`.

The [proxy pipeline](architecture.md) instruments inbound HTTP handling and outbound work, and `JobHandler` creates spans for admission, mutation, validation, and each configured controller. Generated metrics in `pkg/o11y/metric.go`, defined by `pkg/o11y/nacp.yaml`, count validator/mutator warnings and errors plus successful mutations, labeled by controller name.

For a working telemetry-oriented policy example, use `example/otel/`; for the broad demo configuration, use `example/demo/nacp.conf`. Validate export behavior with `pkg/otel/otel_test.go` and the OTel-related executable tests rather than relying solely on collector dashboards.

## Tests and CI

The main CI workflow at `.github/workflows/go.yml` runs on pushes and pull requests to `main`:

```bash
test -z "$(gofmt -l .)"
go build -v ./...
go test -coverprofile=cov.all.out -v ./...
grep -v 'o11y/metric.go' cov.all.out > cov.out
go tool cover -func=cov.out
```

SonarCloud analysis follows those checks. Run the format, build, and package-test steps before modifying production code.

| Change area | Primary tests | What to protect |
| --- | --- | --- |
| HTTP routes, rewrites, TLS, token resolution | `cmd/nacp/nacp_test.go`, `cmd/nacp/nacp_otel_test.go` | Registration, plan, validation, gzip responses, warning/error injection, upstream interactions. |
| Ordering and aggregation | `pkg/admissionctrl/controller_test.go` | Mutation chaining, mutation-before-validation, warnings, validator aggregation, and nil-job handling. |
| HCL parsing/defaults | `pkg/config/config_test.go` | Defaults, output validation, controller declarations, telemetry, SDK config. |
| Individual policy mechanisms | `pkg/admissionctrl/**/**/*_test.go` | OPA queries, JSON Patch, remote contracts, bundle decisions, and Notation trust behavior. |
| OTel exporters/providers | `pkg/otel/otel_test.go` | Export configuration and provider flushing. |

Notation integration tests can exercise registry/signature behavior, so distinguish a local unit-test failure from an integration dependency problem before weakening test coverage.

## Examples and developer resources

`example/readme.md` maps several capability demonstrations: a cost-center validator, a simple metadata mutator, PostgreSQL/Vault template injection, and oauth2-proxy injection. `example/infra/` contains supporting development infrastructure. These are demonstrations, not production deployment manifests; the examples README says so explicitly.

`templates/registry/` is a metric code-generation template source for `pkg/o11y/metric.go`, not a policy workflow system. Changes to the metric definition should consider the YAML specification, generated output, and its exclusion from the coverage report.

## Documentation and maintenance notes

- Prefer `pkg/config/config.go` over the README for the current HCL structure. The README’s `server {}` block conflicts with the source’s top-level `bind`, `port`, and `tls` fields.
- The `CHANGELOG.md` OTel HCL example uses an older `logging { type = "otel" }` form; current source models OTel as `telemetry { logging { otel { enabled = ... } } }`.
- Recent history is high-signal for rationale: package relocation, slog/OTel work, token-context handling, then OPA SDK and bundle JSON-Patch additions. Use targeted `git log`/`git show` for affected files rather than treating old README prose as authoritative.
- The working tree contained untracked OpenWiki/agent-workflow support files at initialization. They were not treated as application-source behavior; preserve local changes when running validation or release commands.

For policy contract details, return to [policy integrations](policy-integrations.md); for exact interception and result propagation behavior, return to [architecture](architecture.md).
