# Demo for HashiTalks Deploy 2023

## Setup

### Cluster

Starts the nomad dev cluster (assuming mac and en0 network)

```bash
sudo misc/hashitalk_deploy2023/demos/setup/infra
/start_nomad.sh
```

### Support Infra

```bash
cd misc/hashitalk_deploy2023/demos/setup/infra
terraform init
terraform apply -auto-approve
```

```bash
cd misc/hashitalk_deploy2023/demos/setup/vault
terraform init
terraform apply -auto-approve
```

### Images

```bash
cd misc/hashitalk_deploy2023/assets/apps/my-app
docker build -t my-app:v1 .
```

```bash
cd misc/hashitalk_deploy2023/assets/apps/grafana-agent-sidecar
docker build -t grafana-agent-sidecar:v1 .
```

### Finally run NACP

```bash
nacp -config=nacp.conf.hcl
```

## Demo

Point the nomad cli to the proxy
```bash
cd demos
export NOMAD_ADDR='http://localhost:6464'

nomad run my-app.nomad
```


## Rego Files

1. validate meta `owner` field: `demos/demo1/owner.rego`
2. add `PG...` env vars: `demos/demo2/postgres.rego`
3. add log shipper side car: `demos/demo3/logging.rego`

