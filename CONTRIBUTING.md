# Contributing

## Development setup

Install the Go version declared in `go.mod`. Optional development tools and tasks are defined in `mise.toml`.

Start with the [OpenWiki quickstart](openwiki/quickstart.md) and [source map](openwiki/source-map.md) before changing admission behavior.

## Checks

Run the default checks before opening a pull request:

```bash
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
```

Docker-backed Notation tests are opt-in locally:

```bash
go test -tags=integration ./pkg/admissionctrl/notation
```

Add focused regression tests for behavior changes. Route or response-rewrite changes belong in `cmd/nacp/nacp_test.go`; ordering and controller-contract changes belong in `pkg/admissionctrl/controller_test.go`; adapter tests should remain next to their implementation.

## Pull requests

- Keep changes focused and explain user-visible or operational impact.
- Preserve mutation order and ensure validators see the final mutated job.
- Document configuration and policy contract changes, including breaking changes in `CHANGELOG.md`.
- Update runnable examples when they are part of the changed contract.
- Never commit credentials, ACL tokens, private keys, trust-store secrets, or generated local artifacts.

OpenWiki pages under `openwiki/` are generated. Prefer updating source code and maintained documentation, then allow the OpenWiki workflow to regenerate them.
