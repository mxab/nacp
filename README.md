# NACP - Nomad Admission Control Proxy

This proxy acts as a middleman between the Nomad API and the Nomad client.
![nacp](https://user-images.githubusercontent.com/1607547/224442234-685950f7-43ff-4570-91d1-fe004827caef.png)

## How
It intercepts the Nomad API calls that include job data (plan, register, validate) and performs mutation and validation on the job data. The job data is at that point is already transformed from HCL to JSON.
If any errors occur the proxy will return the error to the Nomad API caller.
Warnings are attached to the Nomad response when they come back from the actual Nomad API.

Currently validation comes into two flavors:
- Embedded OPA rules
- Webhooks

## Mutation

During the mutation phase the job data is modified by the configured mutators.
### OPA
The opa mutator uses the [OPA](https://www.openpolicyagent.org/) policy engine to perform the mutation.
The OPA rule is expects to return a [JSONPatch](https://jsonpatch.com/) object. The JSONPatch object is then applied to the job data.
It can also return errors and warnings.
An example rego could look like this:

```rego
package hello_world_meta

default patch = []

patch := [
    {
        "op": "add",
        "path": "/Meta",
        "value": {
            "hello": "world"
        }
    }
]

errors := [
    "some error"
]

warnings := [
    "some warning"
]
```

For the embedded you also have to define the query that is used to extract the patch from the OPA response:

```hcl
mutator "opa_jsonpatch" "hello_world_opa_mutator" {

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

	not input.Meta.costcenter
	msg := "Every job must have a costcenter metadata label"
}

errors contains msg if {
	value := input.Meta.costcenter

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

### Other Configuration

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

# Note
This work was inspired by the internal [Nomad Admission Controller](https://github.com/hashicorp/nomad/blob/v1.5.0/nomad/job_endpoint_hooks.go#L74)
