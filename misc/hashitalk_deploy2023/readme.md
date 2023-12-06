# Demo for HashiTalks Deploy 2023

## Setup

### Cluster

Starts the nomad dev cluster

```
sudo misc/hashitalk_deploy2023/demos/setup/infra
/start_nomad.sh
```

### Support Infra

```
cd misc/hashitalk_deploy2023/demos/setup/infra
terraform init
terraform apply -auto-approve
```

```
cd misc/hashitalk_deploy2023/demos/setup/vault
terraform init
terraform apply -auto-approve
```

### Images

```
cd misc/hashitalk_deploy2023/assets/apps/my-app
docker build -t my-app:v1 .
```

```
cd misc/hashitalk_deploy2023/assets/apps/grafana-agent-sidecar
docker build -t grafana-agent-sidecar:v1 .
```

### Finally run NACP
```
nacp -config=nacp.conf.hcl
```

## Demo

Point the nomad cli to the proxy
```
export NOMAD_ADDR='http://localhost:6464'

nomad run my-app.nomad
```
