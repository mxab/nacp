---
type: Architecture Guide
title: Proxy and admission pipeline
description: How NACP selectively intercepts Nomad job APIs, orders mutation and validation, and returns Nomad-compatible outcomes.
tags: [architecture, nomad, proxy, admission-control]
---

# Proxy and admission pipeline

`cmd/nacp/nacp.go` builds NACP around Go’s `httputil.NewSingleHostReverseProxy`. The proxy forwards ordinary Nomad API traffic, but decodes job-bearing requests that match its admission routes. It sends their payloads to the configured controllers in [policy integrations](policy-integrations.md), replaces the job in the original Nomad request envelope, and forwards it upstream when the endpoint’s admission rules allow it.

## Intercepted surface

Only `PUT` and `POST` requests matching the following shapes go through admission:

| Intent | Path | Request handling |
| --- | --- | --- |
| Create/register | `/v1/jobs` | Decode `api.JobRegisterRequest`, admit its `Job`, then forward rewritten request. |
| Update/register | `/v1/job/<id>` | Same register flow; matching uses the job-path regex in `cmd/nacp/nacp.go`. |
| Plan | `/v1/job/<id>/plan` | Decode `api.JobPlanRequest`, admit its `Job`, then forward rewritten request. |
| Validate | `/v1/validate/job` | Decode `api.JobValidateRequest`, admit its `Job`, forward it, then rewrite Nomad’s validation response. |

Any other method/path is proxy traffic, not admission traffic. The regex also makes update and plan matching narrower than a generic “all Nomad job IDs” claim, so route changes must update endpoint tests.

## Request lifecycle and outcomes

1. NACP derives `RequestContext.ClientIP` from the first `X-Forwarded-For` value, falling back to `RemoteAddr`.
2. If any configured controller enables `resolve_token`, an actionable admission request triggers `GET /v1/acl/token/self` against Nomad with the incoming `X-Nomad-Token`. Returned ACL data is attached to the payload context.
3. NACP constructs `types.Payload` as `{ "job": <api.Job>, "context": <optional request context> }` and invokes `JobHandler`.
4. The handler runs mutators in configuration order. Each mutator receives the prior mutator’s output. A mutator error or a nil returned job fails fast.
5. The handler then runs validators, aggregating validation errors while retaining warnings. Mutators therefore must accept imperfect input; validators see the final mutated job.
6. NACP rewrites the original Nomad request with the admitted `Job`. It stores warnings (and, for the validate endpoint, validator errors) in request context so `ModifyResponse` can merge them into Nomad’s response, including gzip-compressed bodies.

For **register** and **plan**, any admission error produces NACP’s local HTTP 500 and the request is not proxied. For **validate**, mutator errors still stop locally, but validator errors are forwarded as part of the rewritten Nomad validation response. This endpoint distinction is implemented in `handleRegister`, `handlePlan`, `handleValidate`, and the corresponding response handlers in `cmd/nacp/nacp.go`.

## Controller contract

`pkg/admissionctrl/controller.go` defines the extension seam:

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

Mutators report a resulting job, whether it changed, warnings, and one fatal error. Validators report warnings and errors; `JobHandler` combines validator failures with `go-multierror`. This contract is implemented by the concrete OPA, SDK/bundle, webhook, and Notation adapters mapped in [policy integrations](policy-integrations.md#controller-families).

## Security and transport boundaries

The proxy has independent listener and upstream-Nomad TLS configuration; [operations and testing](operations-testing.md#transport-and-telemetry) explains both boundaries. Client-IP context is policy-visible, so deployments that trust it must ensure upstream proxy layers sanitize `X-Forwarded-For`. Resolved token data is in the serialized payload context, not a dedicated remote-policy header.

Inbound handling, controller spans, and controller metrics surround this lifecycle. The [operations guide](operations-testing.md#transport-and-telemetry) explains how those signals are configured and where their tests live. The [source map](source-map.md#runtime-and-admission) locates the endpoint tests that protect this behavior.

## Change checklist

- Preserve mutation-before-validation ordering and configured slice order.
- Test every changed endpoint flow in `cmd/nacp/nacp_test.go`; include response rewriting and gzip when applicable.
- Test controller sequencing, error aggregation, warnings, and nil-job handling in `pkg/admissionctrl/controller_test.go`.
- Inspect targeted history before changing sensitive proxy behavior: `8793e10` narrowed ACL self-token lookups to actionable routes; later work added OPA SDK and bundle controllers.
