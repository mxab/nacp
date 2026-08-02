# NACP - Nomad Admission Control Proxy

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=mxab_nacp&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=mxab_nacp)

A proxy infront of the Nomad API that allows to perform mutation and validation on the job data.



![nacp](https://user-images.githubusercontent.com/1607547/224442234-685950f7-43ff-4570-91d1-fe004827caef.png)

## How
It intercepts the Nomad API calls that include job data (plan, register, validate) and performs mutation and validation on the job data. The job data is at that point is already transformed from HCL to JSON.
If any errors occur the proxy will return the error to the Nomad API caller.
Warnings are attached to the Nomad response when they come back from the actual Nomad API.

Currently validation comes into two flavors:
- Embedded OPA rules
- Webhooks

## Input

The input for mutation and validation is the job data and the request metadata. See the [`types.Payload` struct](./admissionctrl/types/opa_payload.go) and its json representation for more details.

## Mutation

During the mutation phase the job data is modified by the configured mutators.
### OPA
The opa mutator uses the [OPA](https://www.openpolicyagent.org/) policy engine to perform the mutation.
The OPA rule is expects to return a [JSONPatch](https://jsonpatch.com/) object. The JSONPatch object is then applied to the job data.
It can also return errors and warnings.
An example rego could look like this:

```rego
package hello_world_meta
import future.keywords

patch contains ops if [

   input.job.Name == "greeting_job"
   ops:= {
        "op": "add",
        "path": "/Meta",
        "value": {
            "hello": "world"
        }
    }
]

errors contains msg if {

    input.job.Name == "silent_job"
    msg := "cannot greet"
}

warnings contains msg if {

  input.job.Name == "had_no_coffee_yet_job"
  msg := "you should have coffee first"
}
```

For the embedded you also have to define the query that is used to extract the patch from the OPA response:

```hcl
mutator "opa_json_patch" "hello_world_opa_mutator" {

    opa_rule {
        query = <<EOH
        patch = data.hello_world_meta.patch
        errors = data.hello_world_meta.errors
        warnings = data.hello_world_meta.warnings
        EOH
        filename = "hello_world_meta.rego"
    }
}
```

### Webhook

The webhook mutator sends the job data to a configured endpoint and expects a JSONPatch object in return.
It can also return errors and warnings.
The JSONPatch object is then applied to the job data.
An example response could look like this:

```json
{
  "patch": [
    {
      "op": "add",
      "path": "/Meta",
      "value": {
        "hello": "world"
      }
    }
  ],
  "errors": [
    "some error"
  ],
  "warnings": [
    "some warning"
  ]
}
```

The webhook mutator can be configured with the following options:

```hcl
mutator "json_patch_webhook" "hello_world_webhook_mutator" {

  webhook {
    endpoint = "http://example.org/send/job/here"
    method = "POST"
  }

}
```

Hint: You can also setup the OPA server as a webhook mutator. You can use the [system main package](https://www.openpolicyagent.org/docs/latest/rest-api/#execute-a-simple-query) to run the OPA server as a webhook mutator.

## Validation

During the validation phase the job data is validated by the configured validators. If any errors occur the proxy will return the error to the Nomad API caller.
Warnings are attached to the Nomad response when they come back from the actual Nomad API.

### OPA

The opa validator uses the [OPA](https://www.openpolicyagent.org/) policy engine to perform the validation.
The OPA rule is expects to return a list of errors and warnings.
An example rego could look like this:

```rego
package costcenter_meta

import future.keywords.contains
import future.keywords.if

errors contains msg if {

	not input.job.Meta.costcenter
	msg := "Every job must have a costcenter metadata label"
}

errors contains msg if {
	value := input.job.Meta.costcenter

	not startswith(value, "cccode-")
	msg := sprintf("Costcenter code must start with `cccode-`; found `%v`", [value])
}
```

Then configure the validator in the config file:

```hcl
validator "opa" "costcenter_opa_validator" {

    opa_rule {
        query = <<EOH
        errors = data.costcenter_meta.errors
        warnings = data.costcenter_meta.warnings
        EOH
        filename = "costcenter_meta.rego"
    }
}
```

### Webhook

The webhook validator sends the job data to a configured endpoint and expects a list of errors and warnings in return.

The response should include potential `errors` and `warnings`:

```json
{
  "errors": [
    "some error"
  ],
  "warnings": [
    "some warning"
  ]
}
```

The webhook validator can be configured with the following options:


```hcl
validator "webhook" "some_webhook_validator" {

  webhook {
    endpoint = "http://example.org/send/job/here"
    method = "POST"
  }

}
```

## OPA Bundles

The `opa` and `opa_json_patch` controllers above read a single Rego file from disk, so changing a policy means redeploying NACP. OPA *bundles* instead let NACP pull policy from a bundle server at runtime, on OPA's own refresh schedule, so rules can be shipped independently of the proxy.

Declare one `opa_bundle` block per bundle source. `config_path` points at a regular [OPA configuration file](https://www.openpolicyagent.org/docs/latest/configuration/) — NACP passes it to the OPA SDK verbatim, so services, bundles, signing, decision logs and status settings all work as documented upstream.

```hcl
opa_bundle "platform" {
  config_path = "/local/opa-platform.yml"

  # Optional. How long startup waits for the first bundle activation. Default 30s.
  ready_timeout = "30s"
  # Optional. Bounds a single policy evaluation. Default 5s; "0s" to disable.
  decision_timeout = "5s"
  # Optional. Refuse to start unless every bundle verifies signatures. Default false.
  require_signing = true
}

validator "opa_bundle" "costcenter" {
  bundle_rule {
    source = "platform"     # the opa_bundle id; optional when only one is configured
    path   = "/costcenter"  # the decision to evaluate
  }
}

mutator "opa_bundle_json_patch" "add_meta" {
  bundle_rule {
    source = "platform"
    path   = "/add_meta"
  }
}
```

Several `opa_bundle` blocks may be declared, each with its own services and signing keys — for example a platform-wide bundle alongside per-team ones. Each `bundle_rule` then names the `source` it evaluates against.

### Decision contract

A bundle decision path must evaluate to an **object**. Validators read `errors` and `warnings`; JSON-Patch mutators additionally read `patch`:

```rego
package costcenter

errors contains msg if {
	not input.job.Meta.costcenter
	msg := "Every job must have a costcenter metadata label"
}

warnings contains msg if {
	input.job.Priority > 75
	msg := "High priority jobs are reviewed manually"
}
```

Anything else fails the admission rather than being treated as "no findings" — a path pointing at a scalar (`data.costcenter.allow`), a missing decision, or an `errors` value that is not a list of strings all produce an admission error. This is deliberate: policy that is fetched over the network must fail closed when it does not say what NACP expects.

Rego builtin errors are also fatal (`strict-builtin-errors`), so a failing `http.send` or `json.unmarshal` surfaces instead of silently making the rule undefined.

### Signing

Bundle policy runs with full authority over every job passing through the proxy, so whoever can answer the bundle URL can rewrite admission control. Configure [bundle signing](https://www.openpolicyagent.org/docs/latest/management-bundles/#signing) in the OPA configuration and set `require_signing = true` to make NACP refuse to start without it:

```yaml
keys:
  global_key:
    algorithm: RS256
    key: ${BUNDLE_PUBLIC_KEY}

bundles:
  platform:
    service: bundle_server
    resource: /bundle.tar.gz
    signing:
      keyid: global_key
```

### Operating bundles

- `GET /-/health` reports each bundle's active revision and last successful activation, returning 503 until every configured bundle has activated. Nomad's API lives under `/v1/`, so this endpoint does not shadow a proxied route.
- Failed refreshes are logged at warn level. NACP keeps enforcing the last activated bundle, so this log line is the signal that policy has gone stale.
- Every decision is logged and traced with its OPA decision ID and the active revision of each bundle, so an admission outcome can be traced back to an exact policy version.
- The `nacp.opa.decision.duration` metric records evaluation time by bundle source, decision path and outcome.
- `SIGHUP` re-reads each `config_path` and reconfigures the running instances. A configuration that fails to load leaves the previous one active.

## More Examples

Checkout the [examples](./example) folder for more examples.

## Usage
### Run Proxy

```bash
$ nacp -config config.hcl
```

It will launch per default on port 6464.

### Send Job to Nomad via Proxy

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run job.hcl
```

## Other Configuration

### NACP Server

The NACP server can be configured with the following options:

```hcl
server {
  # The address the server will listen on
  bind = "0.0.0.0"
  port = 6464

  tls { # If this is present nomad will use TLS
    # The path to the certificate file
    cert_file = "cert.pem"
    # The path to the private key file
    key_file = "key.pem"

    # The path to the CA certificate file
    ca_file = "ca.pem"
  }
}
```

### Nomad Upstream

The Nomad upstream can be configured with the following options:

```hcl
nomad {
  # The address of the Nomad API
  address = "http://localhost:4646"

  tls { # If this is present nomad will use TLS
    # The path to the certificate file
    cert_file = "cert.pem"
    # The path to the private key file
    key_file = "key.pem"

    # The path to the CA certificate file
    ca_file = "ca.pem"
  }
}
```

### Notation

Image signature validation can be done in two ways. Either by the `notation` validator or via the opa by using the `notation_verify_image` function which returns either `true` if the image is valid or `false` if the image is not valid.
See [example/notation](./example/notation) for an example.

Both validators expect a notation block. E.g.:

```hcl
...
validator "opa" "notation_opa_validator" {

  opa_rule {
      ...
  }
  notation {
    repo_plain_http   = false
    trust_store_dir   = "/some/path/to/truststore"
    trust_policy_file = "/some/path/to/trustpolicy.json"
    credential_store_file = "/some/path/to/credentialstore.json"
  }
}
```

The `credential_store_file` refers to the [oras' credential file] (https://docs.docker.com/engine/reference/commandline/cli/#docker-cli-configuration-file-configjson-properties)

e.g.:
```json
{
  "auths": {
    "https://my-registy.example.org": {
      "auth": "<base64 encoded username:password>"
    }
  }
}
```

### OpenTelemetry

NACP has built-in support for OpenTelemetry (OTLP) to provide observability and monitoring capabilities for admission requests and responses.

The OpenTelemetry exporter can be configured using the following settings in the configuration file:

```hcl
telemetry {
  logging {
    otel {
      enabled = true # Enable OpenTelemetry logging
    }
  }
  metrics {
    enabled = true # Enable metrics collection
  }
  tracing {
    enabled = true # Enable tracing
  }
}
```

The OTLP exporter is configured using the common OpenTelemetry environment variables. You can set these variables to specify the endpoint, and other settings for the OTLP exporter. (e.g., `OTEL_SERVICE_NAME=nacp`, `OTEL_RESOURCE_ATTRIBUTES=...`, etc.)

### slog logging

To use `log/slog` for logging, you can configure the telemetry logging settings in your NACP configuration file. This allows you to add json and text slog handlers.

```hcl
telemetry {
  logging {

    level = "info" # Set the logging level (e.g., debug, info, warn, error)
    slog {
      json = true # Adds the json slog handler (defaults to false)
      text = true # Adds the text slog handler (defaults to false)

      text_out = "stderr" # default "stdout"
      json_out = "stdout" # same
    }
  }
}
```

# Note
This work was inspired by the internal [Nomad Admission Controller](https://github.com/hashicorp/nomad/blob/v1.5.0/nomad/job_endpoint_hooks.go#L74)
