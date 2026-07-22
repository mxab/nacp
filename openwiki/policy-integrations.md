---
type: Integration Guide
title: Policy integrations and configuration
description: HCL configuration and input/output contracts for embedded OPA, OPA SDK bundles, webhooks, and Notation admission controllers.
tags: [opa, webhooks, notation, configuration, admission-control]
---

# Policy integrations and configuration

The HCL schema in `pkg/config/config.go` selects controller implementations at startup in `cmd/nacp/nacp.go`. Every selected controller participates in the [proxy and admission pipeline](architecture.md): configured mutators run first, then validators process the rendered job.

## Configuration model

Top-level configuration fields are `bind`, `port`, `tls`, `nomad`, repeated `validator` and `mutator` blocks, `telemetry`, and optional `opa_sdk`. Defaults are `0.0.0.0:6464`, upstream Nomad `http://localhost:4646`, info-level text logs to stdout, and disabled OTel exports.

A controller is a labeled HCL block with a type and name. Startup wires these types:

| Role | Type | Supporting block | Implementation area |
| --- | --- | --- | --- |
| Validator | `opa` | `opa_rule` | `pkg/admissionctrl/validator/opa_validator.go` |
| Validator | `opa_sdk` | `opa_sdk_rule` | `pkg/admissionctrl/validator/opa_bundle_validator.go` |
| Validator | `webhook` | `webhook` | `pkg/admissionctrl/validator/webhook_validator.go` |
| Validator | `notation` | `notation` | `pkg/admissionctrl/validator/notation_validator.go` |
| Mutator | `opa_json_patch` | `opa_rule` | `pkg/admissionctrl/mutator/opa_json_patch.go` |
| Mutator | `opa_bundle_json_patch` | `opa_sdk_rule` | `pkg/admissionctrl/mutator/opa_bundle_json_patch.go` |
| Mutator | `json_patch_webhook` | `webhook` | `pkg/admissionctrl/mutator/json_patch_webhook.go` |

`pkg/config/config_test.go` and its `testdata/` fixtures protect decoding/default behavior. `example/demo/nacp.conf` is the repository’s only combined configuration example; it exercises all but Notation.

## Shared policy payload

All adapters receive the same `types.Payload` (`pkg/admissionctrl/types/opa_payload.go`):

```json
{
  "job": { "...": "Nomad api.Job" },
  "context": {
    "clientIP": "...",
    "accessorID": "...",
    "resolveToken": false,
    "tokenInfo": { "...": "optional Nomad ACL token" }
  }
}
```

`context` is omitted when unavailable. The proxy adds client IP; token data is populated for actionable routes when a configured controller requests token resolution. This payload replaced a job-only controller input in v0.7.0 (see `CHANGELOG.md`), so Rego and webhook consumers must not assume `input` or an HTTP body is a raw job.

## Embedded OPA

Embedded OPA reads the configured Rego source file, prepares the configured query, and evaluates the shared payload as OPA `input` (`pkg/admissionctrl/opa/opa.go`). Queries should bind `errors` and `warnings`; JSON-Patch mutators also bind `patch`.

```hcl
validator "opa" "costcenter" {
  opa_rule {
    filename = "costcenter.rego"
    query = <<EOH
errors = data.costcenter.errors
warnings = data.costcenter.warnings
EOH
  }
}
```

The embedded JSON-Patch mutator expects JSON Patch operations and applies them to the job. The README and `example/example1/`–`example/example4/` provide Rego examples; focused adapter and patch behavior lives under `pkg/admissionctrl/{opa,mutator,validator}/`.

An `opa_rule` may include a `notation` block. When configured, embedded OPA registers `notation_verify_image(string) -> bool`, connecting Rego decisions to the trust material described in [Notation](#notation-image-verification).

## OPA SDK and bundles

`opa_sdk "<id>" { config_path = ... }` creates an OPA SDK instance during NACP startup. The executable waits up to 30 seconds for the SDK to become ready before serving. `opa_sdk_rule { path = ... }` selects the decision path used by either an `opa_sdk` validator or `opa_bundle_json_patch` mutator.

Both adapters pass the same payload to `sdk.OPA.Decision`. Validators consume `errors` and `warnings`; bundle mutators also consume a JSON Patch `patch`. `example/demo/nacp.conf`, `example/demo/opa.yml`, and the bundle Rego directories are the grounded reference for current decision paths and results.

Bundle work was added after the embedded adapters and the latest implementation commit is explicitly a “working poc.” Treat bundle availability, refresh, and rollout behavior as an operational question to validate in your deployment; it is not a documented guarantee of this codebase. The pipeline still enforces the same ordering and error rules in [architecture](architecture.md#request-lifecycle-and-outcomes).

## Webhooks

Webhook adapters serialize the complete payload to their configured endpoint. Validators expect errors/warnings; JSON-Patch mutators additionally expect patch operations. Remote calls project selected context into `X-Forwarded-For`, `NACP-Client-IP`, and, when present, `NACP-Accessor-ID` (`pkg/admissionctrl/remoteutil/remoteutil.go`). The full token object remains in JSON `context.tokenInfo`.

```hcl
validator "webhook" "external-policy" {
  webhook {
    endpoint = "https://policy.example/validate"
    method   = "POST"
  }
}
```

Webhook availability and output-schema compatibility are part of the admission path: adapter errors become admission errors under the endpoint behavior in [architecture](architecture.md). The inspected adapters do not establish a dedicated per-hook timeout, retry policy, or TLS/client-auth configuration, so define those controls at the surrounding network/service layer or extend and test the adapters deliberately.

## Notation image verification

The `notation` validator examines Docker-driver task images and verifies them using Notary Project trust policy and trust store material. Its config accepts `trust_policy_file`, `trust_store_dir`, optional `repo_plain_http`, `credential_store_file`, and `max_sig_attempts`. Keep credential-store contents out of source and configuration documentation; NACP consumes only a file path.

Notation can be used directly as `validator "notation"` or from embedded Rego through `notation_verify_image`. The latter lets policy choose which image references require verification, but still depends on the configured trust materials. Use `example/notation/` and its integration-focused tests for the expected signed-registry workflow.

## Configuration cautions

- Prefer `pkg/config/config.go` and runnable examples to README snippets; the README’s `server {}` wrapper is stale relative to the current top-level fields.
- `LoadConfig` explicitly validates slog output destinations, but it does not provide a comprehensive preflight for each controller’s required nested block. Invalid minimal configs can therefore fail late during startup.
- The default `max_sig_attempts` value is applied while iterating top-level Notation validators. Add a focused test before relying on the same default for nested OPA-rule Notation config.

For where each adapter, test, and sample lives, continue to the [source map](source-map.md#policy-and-configuration).
