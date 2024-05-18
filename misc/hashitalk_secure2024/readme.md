# HashiTalks Secure 2024

This is the demo code for my HashiTalks Secure 2024 talk.

## Full example

Start Nomad in dev mode
```bash
sudo nomad agent -dev -bind=0.0.0.0
```

Start NACP
```bash
nacp -config nacp.conf.hcl
```

Run the registry job (port 5000 on macos requires to disable airdrop receiver otherwise it gets wired)
```bash
nomad run registry.nomad
```

```bash


docker build -t localhost:5000/my-app:v1 .

docker push localhost:5000/my-app:v1

nomad run demo.nomad

# full diget image reference
docker inspect --format='{{index .RepoDigests 0}}' localhost:5000/my-app:v1

# update demo with digest (image=...)
nomad run demo.nomad

nomad run -purge demo

# generate certs
notation cert generate-test --default "wabbit-networks.io"

# update nacp config (remove part2 comment block)

nomad run demo.nomad


grype <digest>

notation sign <digest>

nomad run demo.nomad
