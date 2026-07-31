# NACP — Nomad Admission Control Proxy

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=mxab_nacp&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=mxab_nacp)

NACP is a reverse proxy for the HashiCorp Nomad API. It intercepts job registration, planning, and validation requests, then runs configured mutation and validation policies before forwarding accepted jobs to Nomad.

![NACP admission flow](https://user-images.githubusercontent.com/1607547/224442234-685950f7-43ff-4570-91d1-fe004827caef.png)

## Capabilities

| Stage | Integration | Configuration type |
| --- | --- | --- |
| Mutation | Embedded OPA returning JSON Patch | `opa_json_patch` |
| Mutation | OPA SDK/bundle returning JSON Patch | `opa_bundle_json_patch` |
| Mutation | JSON-Patch webhook | `json_patch_webhook` |
| Validation | Embedded OPA | `opa` |
| Validation | OPA SDK/bundle | `opa_bundle` |
| Validation | Validation webhook | `webhook` |
| Validation | Notation container-image verification | `notation` |

Policies receive a shared payload containing the rendered Nomad job and optional request context:

```json
{
  "job": { "ID": "example" },
  "context": {
    "clientIP": "192.0.2.10",
    "accessorID": "optional-nomad-token-accessor",
    "resolveToken": true,
    "tokenInfo": {
      "Policies": ["example-policy"]
    }
  }
}
```

`tokenInfo` is deliberately sanitized and never includes the Nomad token `SecretID`. See [`types.Payload`](pkg/admissionctrl/types/opa_payload.go) and [`config.RequestContext`](pkg/config/config.go) for the source contract.

## Admission flow

NACP applies mutators in configuration order, then runs validators against the final mutated job. Validators receive isolated job copies and cannot alter the job forwarded to Nomad.

Admission runs for `PUT` and `POST` requests to:

- `/v1/jobs`
- `/v1/job/<id>`
- `/v1/job/<id>/plan`
- `/v1/validate/job`

Other Nomad API traffic is proxied without admission processing. Mutator or integration failures stop the request. Validation errors stop registration and planning; for Nomad's validation endpoint they are merged into the Nomad-compatible validation response. Policy warnings are merged into successful Nomad responses.

## Install

### Release binary

Download an archive and its signed checksum from [GitHub Releases](https://github.com/mxab/nacp/releases).

### Go

```bash
go install github.com/mxab/nacp/cmd/nacp@latest
```

### Container

Release images are published to `ghcr.io/mxab/nacp` and run as a non-root user:

```bash
docker run --rm -p 6464:6464 \
  -v "$PWD/config.hcl:/etc/nacp/config.hcl:ro" \
  ghcr.io/mxab/nacp:latest \
  -config /etc/nacp/config.hcl
```

Pin a version tag rather than `latest` for production deployments.

## Quickstart

The module's required Go version is declared in [`go.mod`](go.mod). Build and run the first embedded-OPA example from its directory so its relative Rego path resolves correctly:

```bash
go build -o nacp ./cmd/nacp
cd example/example1
../../nacp -config example1.conf.hcl
```

In another terminal, point the Nomad CLI at NACP:

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run example/example1/example1.nomad
```

Without `-config`, NACP listens on `0.0.0.0:6464` and proxies Nomad at `http://localhost:4646`.

## Configuration

Top-level server settings are not wrapped in a `server` block:

```hcl
bind = "0.0.0.0"
port = 6464

nomad {
  address = "http://localhost:4646"
}

validator "opa" "costcenter" {
  opa_rule {
    filename = "policies/costcenter.rego"
    query = <<EOH
errors = data.costcenter.errors
warnings = data.costcenter.warnings
EOH
  }
}
```

Configuration is validated before the server starts. Missing controller blocks, invalid HTTP endpoints, invalid ports, incomplete TLS key pairs, and bundle controllers without an `opa_sdk` block are rejected with actionable startup errors. An explicitly supplied but missing configuration file is also an error.

Current combined examples:

- [`example/example1`](example/example1) — embedded OPA validation
- [`example/example2`](example/example2) — embedded OPA JSON-Patch mutation
- [`example/demo`](example/demo) — webhooks, OPA SDK bundles, and telemetry
- [`example/notation`](example/notation) — signed image validation

## Transport and telemetry

NACP supports separate TLS configuration for its listener and upstream Nomad connection. OpenTelemetry logging, metrics, and tracing use the standard OTLP environment variables.

```hcl
telemetry {
  logging {
    level = "info"
    slog {
      json = true
    }
    otel {
      enabled = true
    }
  }
  metrics {
    enabled = true
  }
  tracing {
    enabled = true
  }
}
```

## Security and availability

- Run NACP only on a trusted network path and use TLS for production traffic.
- If policies consume `clientIP`, ensure the proxy in front of NACP sanitizes `X-Forwarded-For`; NACP uses its first value.
- Enabling `resolve_token` performs a Nomad `/v1/acl/token/self` lookup. Lookup failures stop admission rather than continuing without identity context.
- Webhooks receive the full job and sanitized request context. They should use HTTPS and be treated as trusted policy services.
- Webhook requests have a 30-second timeout and bounded responses. Network, timeout, malformed-response, and non-2xx failures fail admission closed.
- OPA SDK/bundle support is currently experimental. Validate bundle refresh and degraded-mode behavior in your environment before relying on it in production.
- `insecure_skip_verify` and `repo_plain_http` are intended only for controlled development environments.

## Development

Run the default checks:

```bash
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
```

Notation registry tests require Docker and are opt-in locally:

```bash
go test -tags=integration ./pkg/admissionctrl/notation
```

CI runs the full suite with the `integration` tag and publishes coverage to SonarCloud.

## Documentation

Start with the generated [OpenWiki quickstart](openwiki/quickstart.md), then continue to:

- [Architecture and request lifecycle](openwiki/architecture.md)
- [Policy integrations and contracts](openwiki/policy-integrations.md)
- [Operations and testing](openwiki/operations-testing.md)
- [Source map](openwiki/source-map.md)
- [Examples](example/readme.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [Changelog](CHANGELOG.md)

OpenWiki pages are generated by automation. Correct source code and maintained documentation first, then allow the OpenWiki workflow to refresh generated pages.

## License

See [`LICENSE`](LICENSE).
