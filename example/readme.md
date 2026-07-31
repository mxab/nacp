# Examples

See [setup](#infrastructure-setup) to configure the required infrastructure.

## Validation

### Cost Center

This example demonstrates a validator that checks whether a job contains cost-center metadata and whether the code starts with `cccode-`.

[Cost-center validator](example1/validators/costcenter_meta.rego).


```bash
(cd example1 && ../../nacp -config example1.conf.hcl)
```

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run example1/example1.nomad
```


https://user-images.githubusercontent.com/1607547/227664253-d65cd5d4-12e4-4b99-9143-4a346911b7dd.mov


### Image Validation via Notation

See the README in the [`notation/`](notation/) folder.

## Mutator

### Simple Hello World

This example demonstrates a simple mutator that adds a `hello` key to the job meta data with the value `world`.

[simple hello world mutator](example2/mutators/hello_world_meta.rego)

```bash
(cd example2 && ../../nacp -config example2.conf.hcl)
```

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run example2/example2.nomad
```



https://user-images.githubusercontent.com/1607547/227664271-cc1b82a2-d5ec-4afe-94df-b6702700cf98.mov



### Postgres Env Template Injection

In this example the mutator checks whether a job task contains a `postgres` metadata field. If so, it injects a template block and Vault policy that render the PostgreSQL connection details.

If the `postgres` metadata equals `native` it creates a template that renders the environment variables `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD` and `PGDATABASE`.

If the `postgres` metadata equals `springboot` it creates a template that renders the environment variables `SPRING_DATASOURCE_URL`, `SPRING_DATASOURCE_USERNAME` and `SPRING_DATASOURCE_PASSWORD`.

[PostgreSQL environment-template mutator](example3/mutators/pg.rego)

```bash
(cd example3 && ../../nacp -config example3.conf.hcl)
```

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run example3/example3.nomad
```

https://user-images.githubusercontent.com/1607547/227664282-e2ee22fc-946f-4b0f-9bfc-d275300dcec5.mov


### OAuth2 Proxy Injection

This example deploys a job containing a simple [web application](example4/webapp.js).

If a task group's `secure` metadata names a service such as `webapp`, the mutator injects [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/) and rewrites the service so incoming requests pass through the proxy.

```bash
(cd example4 && ../../nacp -config example4.conf.hcl)
```

```bash
terraform init && terraform apply -auto-approve
```

```bash
NOMAD_ADDR=http://localhost:6464 nomad job run example4/example4.nomad
```

[OAuth2-proxy mutator](example4/mutators/secure.rego)



https://user-images.githubusercontent.com/1607547/227664325-768c8b5f-a991-4467-89e5-26a211c3511c.mov



## Infrastructure Setup

Run Vault

```bash
cd infra/vault
vault server -dev -dev-root-token-id=root -dev-listen-address=0.0.0.0:8200
```

```bash
cd infra/nomad
sudo nomad agent -dev -bind=0.0.0.0 -network-interface=en0 -config=conf
```

Deploy some infrastructure (keycloak, postgres, treafik)

```bash
cd infra/nomad/jobs
terraform init && terraform apply -auto-approve
```

Configure Postgres Database Engine Vault

```bash
cd infra/vault
terraform init && terraform apply -auto-approve
```

These example assume that every nomad job that comes with a service is accessible via `<service_name>.nomad.local`
Use consul catalog to update `/etc/hosts`

```bash
cd infra/etchosts
./run.sh
```
(If you trust my script that requires sudo and writes to `/etc/hosts` ;) )

## Notes



The examples are not meant to be used in production. They are just meant to demonstrate the capabilities of nacp and opa.
