---
type: Integration Guide
title: Policy integrations and configuration
description: HCL configuration, startup validation, and policy input/output contracts for embedded OPA, OPA SDK bundles, webhooks, and Notation controllers.
tags: [opa, webhooks, notation, configuration, admission-control]
openwiki:
  roles: [integration, domain, workflow]
  change_kinds: [configuration, policy-adapter, public-contract]
  source_paths: [pkg/config/config.go, cmd/nacp/nacp.go, pkg/admissionctrl/remoteutil/remoteutil.go, pkg/admissionctrl/types/opa_payload.go]
  symbols: [Config.Validate, validateMutator, validateValidator, createMutator, createValidator, Payload]
  test_paths: [pkg/config/config_test.go, cmd/nacp/nacp_test.go, pkg/admissionctrl/remoteutil/remoteutil_test.go]
  invariants: [A selected controller has a valid type-specific block before server startup, and all adapters receive the same payload shape.]
  validation_commands: [go test ./pkg/config ./cmd/nacp -count=1, go test ./pkg/admissionctrl/... -count=1]
---

# Policy integrations and configuration

The HCL schema in `pkg/config/config.go` selects and validates controller configuration before startup. `cmd/nacp/nacp.go` turns each validated block into a concrete controller; every controller then participates in the mutation-before-validation lifecycle documented in the [proxy and admission pipeline](architecture.md).

## Configuration and startup validation

Top-level fields are `bind`, `port`, `tls`, `nomad`, repeated `validator` and `mutator` blocks, `telemetry`, and optional `opa_sdk`. Defaults are `0.0.0.0:6464`, upstream Nomad `http://localhost:4646`, info-level text logs to stdout, and disabled OTel exports.

`LoadConfig` applies Notation defaults and calls `Config.Validate` before the server is built. Validation rejects invalid listener ports/binds, a non-HTTP(S) Nomad address, incomplete TLS pairs, incomplete `opa_sdk`, unknown controller types, missing controller names or type-specific blocks, invalid webhook endpoint/method, absent OPA SDK for bundle controllers, incomplete Notation trust settings, non-positive signature attempts, and duplicate names within mutators or validators. The duplicate namespaces are separate: a mutator and validator may share a name.

| Role | HCL type | Required block | Startup implementation |
| --- | --- | --- | --- |
| Validator | `opa` | `opa_rule` | `validator.NewOpaValidator` |
| Validator | `opa_bundle` | `opa_sdk_rule` and top-level `opa_sdk` | `validator.NewOpaBundleValidator` |
| Validator | `webhook` | `webhook` | `validator.NewWebhookValidator` |
| Validator | `notation` | `notation` | `validator.NewNotationValidator` |
| Mutator | `opa_json_patch` | `opa_rule` | `mutator.NewOpaJsonPatchMutator` |
| Mutator | `opa_bundle_json_patch` | `opa_sdk_rule` and top-level `opa_sdk` | `mutator.NewOpaBundleMutator` |
| Mutator | `json_patch_webhook` | `webhook` | `mutator.NewJsonPatchWebhookMutator` |

`pkg/config/config_test.go` protects loading/defaults and the validation matrix. `example/demo/nacp.conf` is the combined example; it exercises all listed controller families except Notation.

## Shared policy payload

All adapters receive `types.Payload` from `pkg/admissionctrl/types/opa_payload.go`:

```json
{
  "job": { "...": "Nomad api.Job" },
  "context": {
    "clientIP": "...",
    "accessorID": "...",
    "resolveToken": false,
    "tokenInfo": { "...": "sanitized Nomad ACL token" }
  }
}
```

`context` is present on admission requests, although `tokenInfo` is omitted when unavailable. The proxy supplies client IP and only resolves token data on actionable routes when a configured controller requests it. Policy authors must treat `input` or webhook request JSON as this envelope, not a raw job. `SanitizeACLToken` excludes `SecretID` before policy code receives it.

## Controller families

### Embedded OPA

Embedded OPA loads the configured Rego file, prepares its query, and evaluates the shared payload as OPA `input` (`pkg/admissionctrl/opa/opa.go`). Validator queries bind `errors` and `warnings`; JSON-Patch mutators additionally bind `patch`, whose operations are applied to the job.

An `opa_rule` can contain a `notation` block. This registers `notation_verify_image(string) -> bool`, allowing Rego to call the image verifier described in [Notation](#notation-image-verification). Use `example/example1/` and `example/example2/` for minimal contracts and colocated OPA/mutator tests for behavior changes.

### OPA SDK and bundles

`opa_sdk "<id>" { config_path = ... }` creates one SDK instance during startup. NACP waits up to 30 seconds for readiness before serving. `opa_sdk_rule { path = ... }` selects the decision passed to `sdk.OPA.Decision`; bundle validators consume `errors`/`warnings`, while bundle mutators also consume `patch`.

The current code and `example/demo/{nacp.conf,opa.yml,bundle/}` establish decision input and result shape. They do not establish a production guarantee for bundle availability, refresh, or rollout; verify those conditions in the deployment environment.

### Webhooks

Webhook adapters serialize the complete payload to the configured endpoint. Validators expect errors/warnings; patch mutators additionally expect JSON Patch operations. `remoteutil.ApplyContextHeaders` projects selected context into `X-Forwarded-For`, `NACP-Client-IP`, and, when present, `NACP-Accessor-ID`; the sanitized token object remains only in JSON `context.tokenInfo`.

`remoteutil.NewInstrumentedClient` sets a 30-second timeout. `DecodeJSONResponse` accepts only 2xx responses, limits bodies to 10 MiB, and requires valid JSON. Timeout, transport, status, oversized-body, malformed-response, or adapter-result errors enter the admission error path in the [proxy pipeline](architecture.md#request-lifecycle-and-outcomes). TLS/client authentication are not configurable per webhook block, so manage them at the endpoint/network layer or add an explicitly tested adapter capability.

### Notation image verification

The `notation` validator examines Docker-driver task images and verifies them with Notary Project trust policy and trust-store material. Configuration requires `trust_policy_file` and `trust_store_dir`; `max_sig_attempts` defaults to 50 wherever a Notation block is processed. Optional `repo_plain_http` and `credential_store_file` are intended for controlled use; retain credentials outside source and documentation.

Notation can be a direct `validator "notation"` or an embedded OPA builtin. The latter lets Rego choose which references to verify but still depends on configured trust material. Use `example/notation/` and `pkg/admissionctrl/{notation,validator}/` tests for the expected signed-registry workflow.

## Extension recipe: add a controller type

A new controller type crosses configuration and runtime-registration boundaries; an implementation unit test alone does not make the HCL feature usable.

1. Implement `admissionctrl.JobMutator` or `JobValidator` under `pkg/admissionctrl/`; preserve the payload and result contract above.
2. Add its labeled config support and type-specific validation in `pkg/config/config.go` (`validateMutator` or `validateValidator`). Decide required nested fields and add table cases to `TestConfigValidation`.
3. Register construction in `createMutator` or `createValidator` in `cmd/nacp/nacp.go`; if it changes token needs, ensure the returned `resolveToken` aggregation remains correct.
4. Add adapter result/failure tests next to the implementation and a startup-wiring case in `TestCreateMutatators` or `TestCreateValidators`. Add a representative example only when it remains a supported operational contract.
5. Run `go test ./pkg/config ./cmd/nacp -count=1` plus the adapter package test. Run `go test ./...` conditionally when shared types or multiple controller families changed.

Do not hand-edit generated observability output; if the new adapter needs metrics, change the schema/template path described in [operations and testing](operations-testing.md#transport-and-telemetry).
