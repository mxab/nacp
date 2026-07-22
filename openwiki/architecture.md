---
type: Architecture Guide
title: Proxy and admission pipeline
description: How NACP intercepts Nomad job API requests, applies ordered admission controllers, and carries results back through Nomad-compatible responses.
tags: [architecture, nomad, proxy, admission-control]
---

# Proxy and admission pipeline

NACP’s executable in `cmd/nacp/nacp.go` constructs an `httputil.NewSingleHostReverseProxy` for the configured Nomad address. The proxy is the system boundary: unrecognized Nomad endpoints pass through, while supported job endpoints are decoded, admitted, re-encoded, and forwarded. The policy mechanisms that this boundary invokes are detailed in [policy integrations](policy-integrations.md).

## Intercepted surface

The handler recognizes these request forms:

| Intent | Methods and path | Processing |
| --- | --- | --- |
| Register/create or update a job | `PUT`/`POST /v1/jobs`, `PUT`/`POST /v1/job/<id>` | Decode the Nomad registration request, mutate and validate its `Job`, then forward the rewritten request. |
| Plan a job | `PUT`/`POST /v1/job/<id>/plan` | Decode plan input, apply admission, and forward the rewritten request. |
| Validate a job | `PUT`/`POST /v1/validate/job` | Decode and admit the job, then forward a Nomad validation request whose response may be augmented with policy failures. |

The endpoint predicates and response hooks are in `cmd/nacp/nacp.go` (`isRegister`, `isPlan`, `isValidate`, `ModifyResponse`). Path matching is intentionally narrow; do not infer that all Nomad APIs receive admission enforcement.

## Request lifecycle and outcomes

1. The handler derives a `RequestContext` with client IP. When at least one configured controller enables `resolve_token`, it also requests Nomad’s `/v1/acl/token/self` using the incoming `X-Nomad-Token` and records the returned ACL token/accessor data.
2. It decodes the Nomad job-bearing request and creates a `types.Payload`: `{ "job": ..., "context": ... }`. This payload is the shared contract for embedded policies and remote integrations; see [payload and configuration](policy-integrations.md#payload-and-configuration).
3. `JobHandler.ApplyAdmissionControllers` runs every mutator in configured slice order, passing each mutator the preceding job result. It then runs validators in slice order against the payload. The orchestration lives in `pkg/admissionctrl/controller.go`.
4. A mutator error or a nil job stops processing immediately. Validator errors are aggregated so later validators still run. In register and plan flows, an admission error produces NACP’s local error response rather than forwarding the request. The validation flow carries validator failures through request context and adds them to Nomad’s `JobValidateResponse` after forwarding.
5. Admission warnings are retained in request context. The proxy merges them into Nomad’s register, plan, or validate response; response rewriting handles gzip as well as uncompressed bodies.

This ordering means policy authors must make mutators robust enough to handle imperfect input, while validators see the post-mutation job. Changes to this behavior must be covered by `pkg/admissionctrl/controller_test.go` and the endpoint-level cases in `cmd/nacp/nacp_test.go`; [operations and testing](operations-testing.md#tests-and-ci) names the CI commands.

## Controller contract

`pkg/admissionctrl/controller.go` supplies the small integration boundary:

```go
type JobMutator interface {
    Name() string
    Mutate(context.Context, *types.Payload) (*api.Job, bool, []error, error)
}

type JobValidator interface {
    Name() string
    Validate(context.Context, *types.Payload) (warnings []error, err error)
}
```

Mutators report the resulting job, whether it changed, warnings, and a fatal error. Validators report warnings plus an error that the handler aggregates with `go-multierror`. This contract links controller behavior to [policy integrations](policy-integrations.md), which maps configuration types to concrete implementations.

## Transport and observability boundaries

The proxy serves HTTP or TLS according to configuration and has separately configured upstream Nomad TLS transport. Both the inbound handler and selected outbound HTTP work are instrumented. Admission spans and per-controller warning/error/mutation counters share the observability setup described in [operations and testing](operations-testing.md#observability).

The server sets a 310-second timeout constant for its Nomad-facing server/transport use in `cmd/nacp/nacp.go`; keep long-running plan/register workflows in mind when modifying proxy transport behavior.

## Maintenance notes

- Recent history shows the proxy and controller were reorganized into `pkg/`, then gained token-resolution fixes, OPA SDK support, and bundle JSON-Patch work. Review targeted history (`git log -- cmd/nacp/nacp.go pkg/admissionctrl/controller.go`) when altering those paths.
- The handler derives `ClientIP` from the first `X-Forwarded-For` entry when present, falling back to `RemoteAddr`. Deployments must ensure this header is supplied only by trusted proxy layers if policies depend on it.
- This page describes actual flow, not an assurance that every malformed HCL configuration fails gracefully. The configuration model has limited validation; see [documentation and maintenance notes](operations-testing.md#documentation-and-maintenance-notes).
