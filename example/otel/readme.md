# OTEl Semantic Conventions injection

This demo shows how to inject OTEl Semantic Conventions into your application by using NACP and rego.

## Prerequisites

- Run nomad
Run nacp
```shell

nacp -config=otel.conf.hcl
```

```shell
NOMAD_ADDR=http://localhost:64500 nomad job run example.nomad.hcl

# inspect
nomad exec -job example sh -c 'env | grep OTEL_'
```
