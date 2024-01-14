# Notation Image Verification

This Demo uses the quickstart example from the [Notation](https://notaryproject.dev/docs/quickstart-guides/quickstart-sign-image-artifact/) project.

## Setup

### Run Nomad and proxy
```sh
sudo nomad agent -dev -bind=0.0.0.0
```

```
nacp -config notation.conf.hcl
```

### Run Docker Registry
```sh
nomad job run registry.nomad
```

### Build and push image
```sh
docker build -t localhost:5001/net-monitor:v1 https://github.com/wabbit-networks/net-monitor.git#main
docker push localhost:5001/net-monitor:v1
```