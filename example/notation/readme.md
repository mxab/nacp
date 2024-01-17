# Notation Image Verification

This Demo uses the quickstart example from the [Notation](https://notaryproject.dev/docs/quickstart-guides/quickstart-sign-image-artifact/) project.


https://github.com/mxab/nacp/assets/1607547/c06cf01d-43ba-440c-b63d-7a8a0ade9cfe


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
### Run Nomad Job
```sh
export IMAGE=$(docker inspect --format='{{index .RepoDigests 0}}' localhost:5001/net-monitor:v1)

export NOMAD_ADDR=http://localhost:6464

nomad job run -var "image=${IMAGE}" demo.nomad

# should fail
# now sign image

notation sign $IMAGE

nomad job run -var "image=${IMAGE}" demo.nomad
```
