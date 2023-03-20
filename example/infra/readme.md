# Start Vault in dev mode
```bash
$ vault server -dev -dev-root-token-id=root -dev-listen-address=0.0.0.0:8200
```


# Start Nomad in dev mode
```bash
$ nomad agent -dev -dev-connect -dev-root-token-id=root -dev-listen-address=
```
