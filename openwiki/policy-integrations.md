---
type: Integration Guide
title: Policy integrations and configuration
description: HCL configuration and payload contracts for NACP embedded OPA, OPA SDK/bundles, webhook, and Notation admission controllers.
tags: [opa, webhooks, notation, configuration, admission-control]
---

# Policy integrations and configuration

NACP binds labeled HCL `validator` and `mutator` blocks to policy implementations during executable startup (`cmd/nacp/nacp.go`). Those implementations participate in the ordered [proxy and admission pipeline](architecture.md): all configured mutators run before configured validators.

## Payload and configuration

The source schema is `pkg/config/config.go`. It declares top-level `bind`, `port`, `tls`, `nomad`, `validator`, `mutator`, `telemetry`, and `opa_sdk` configuration. Defaults are bind `0.0.0.0`, port `6464`, upstream Nomad `http://localhost:4646`, info-level text logging to stdout, and disabled OTel exports.

Every controller receives the same serialized payload shape from `pkg/admissionctrl/types/opa_payload.go`:

```json
{
  "job": { "...": "Nomad API job" },
  "context": {
    "clientIP": "...",
    "accessorID": "...",
    "resolveToken": false,
    "tokenInfo": { "...": "optional Nomad ACL token" }
  }
}
```

`context` is omitted when unavailable. Client IP is always built by the proxy; ACL data is populated only when token resolution is requested by an applicable controller. This richer payload replaced job-only policy input in v0.7.0 (recorded in `CHANGELOG.md`), so webhook and Rego policy changes must not assume a bare job object.

The available labeled types are:

| Kind | Types wired at startup | Required supporting block |
| --- | --- | --- |
| Validator | `opa`, `opa_sdk`, `webhook`, `notation` | `opa_rule`, `opa_sdk_rule`, `webhook`, or `notation` as appropriate |
| Mutator | `opa_json_patch`, `opa_bundle_json_patch`, `json_patch_webhook` | `opa_rule`, `opa_sdk_rule`, or `webhook` as appropriate |

Read `pkg/config/config_test.go` when changing defaulting or HCL decoding, and use `example/demo/nacp.conf` for a multi-integration example.

## Embedded OPA

The embedded OPA integration in `pkg/admissionctrl/opa/opa.go` reads a Rego file, prepares the configured query, evaluates it against `types.Payload`, and extracts bindings. Standard validators emit error and warning collections. The JSON-Patch mutator expects patch, error, and warning bindings and applies the patch to the job.

A typical validator configuration is:

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

For mutation, use `mutator "opa_json_patch" ...` and bind `patch` as well. The README contains longer Rego examples, while runnable cases live under `example/example1/` through `example/example4/` and relevant OPA tests under `pkg/admissionctrl/opa/` and `pkg/admissionctrl/mutator/`.

An OPA rule can also carry a `notation` block, enabling the Notation helper described below. Embedded policy decisions therefore depend on the same trust material as the [Notation integration](#notation-image-verification).

## OPA SDK and bundles

`opa_sdk` declares an SDK configuration file and is initialized during startup; the executable waits for SDK readiness before serving. `opa_sdk_rule { path = ... }` selects a decision path for an `opa_sdk` validator or `opa_bundle_json_patch` mutator. These paths allow policy bundles to supply validation or JSON-Patch decisions without embedding each Rego file directly in NACP configuration.

Bundle support is a recent evolution: commits in the current history add SDK setup, bundle initialization, and bundle JSON-Patch work. Use `example/demo/opa.yml`, its bundle directories, and `example/demo/nacp.conf` as the source-backed reference when modifying decision contracts. Its outcomes still flow through the same warning/error and ordering rules in the [admission pipeline](architecture.md#request-lifecycle-and-outcomes).

## Webhooks

Webhook validators send the payload to their configured endpoint and expect errors/warnings. JSON-Patch webhook mutators expect a patch response and can also return warnings. `pkg/admissionctrl/remoteutil/remoteutil.go` implements remote call behavior; remote requests propagate caller metadata through `X-Forwarded-For`, `NACP-Client-IP`, and, when available, `NACP-Accessor-ID`.

```hcl
validator "webhook" "external-policy" {
  webhook {
    endpoint = "https://policy.example/validate"
    method   = "POST"
  }
}
```

Because webhooks receive the combined payload rather than a job alone, downstream services should parse both `job` and `context`. Treat endpoint availability and response-schema compatibility as part of the admission path: remote errors become controller errors under the lifecycle defined in [architecture](architecture.md#request-lifecycle-and-outcomes).

## Notation image verification

The `notation` validator finds Docker-driver task images in the job and verifies them with Notary Project trust policy and trust store material. `pkg/admissionctrl/notation/notation.go` and `pkg/admissionctrl/validator/notation_validator.go` contain this behavior. Configuration supports `trust_policy_file`, `trust_store_dir`, optional plain-HTTP registry access, an optional credential-store file, and `max_sig_attempts`.

Notation can be direct (`validator "notation"`) or exposed to embedded OPA via `notation_verify_image(imageRef)`. The latter lets Rego decide which images require verification but does not eliminate the need for trust configuration. The end-to-end reference is `example/notation/`, whose tests exercise a registry and signed-image cases.

Keep credential files out of source and documentation. NACP’s configuration only names their paths; the project does not require secrets to be placed in HCL.

## Configuration cautions

- `LoadConfig` validates text/JSON slog destinations as `stdout` or `stderr`, but the schema does not appear to comprehensively validate every required controller sub-block before startup uses it. Treat example-backed configurations as safer than minimally specified blocks.
- The code applies the `max_sig_attempts` default (50) while iterating top-level `validator` Notation blocks. Do not assume this post-load default is applied to every nested OPA-rule Notation configuration without testing it.
- The README’s historical `server {}` wrapper is inconsistent with the current top-level `bind`, `port`, and `tls` fields. See [operations and testing](operations-testing.md#documentation-and-maintenance-notes) before copying legacy snippets.
