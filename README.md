# Nomad Admission Control Proxy (NACP)

This proxy acts as a middleman between the Nomad API and the Nomad client.
![nacp](https://user-images.githubusercontent.com/1607547/224442234-685950f7-43ff-4570-91d1-fe004827caef.png)

It intercepts the Nomad API calls that include job data (plan, register, validate) and performs mutation and validation on the job data.

If any errors occur the proxy will return the error to the Nomad API caller.

Warnings are attached to the Nomad response when they come back from the actual Nomad API.

Currently it supports following mutator and validators:
- Opa Engine for the validation and mutation.
- Webhook for the validation and mutation.


This work was inspired by the internal [Nomad Admission Controller](https://github.com/hashicorp/nomad/blob/v1.5.0/nomad/job_endpoint_hooks.go#L74)

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

### Configuration

```hcl
validator "opa" "some_opa_validator" {

    opa_rule {
        query = "errors = data.costcenter_meta.errors"
        filename = "testdata/opa/validators/costcenter_meta.rego"
    }
}

mutator "opa_jsonpatch" "some_opa_mutator" {

    opa_rule {
        query = "patch = data.hello_world_meta.patch"
        filename = "testdata/opa/mutators/hello_world_meta.rego"
    }
}

```
