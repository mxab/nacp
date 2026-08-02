# Dependency Update and Nomad Dependency Removal

Two related changes:

1. Updating every direct dependency that had a newer release.
2. Removing the `github.com/hashicorp/nomad` main module, leaving only `github.com/hashicorp/nomad/api`.

## 1. Dependency update

All 24 direct dependencies with available updates were bumped. `github.com/docker/docker`,
`github.com/hashicorp/nomad`, `stretchr/testify`, `notaryproject/*` and `hashicorp/hcl/v2`
were already current.

Notable bumps and the adjustments they required:

| Dependency | Change | Impact |
| --- | --- | --- |
| `open-policy-agent/opa` | 1.16.0 → 1.19.0 | Wasm engine moved from `wasmtime-go` to pure-Go `wazero`, **dropping the cgo dependency**. No Rego policy changes needed. |
| `go.opentelemetry.io/otel` | 1.43.0 → 1.44.0 | — |
| `otel/log`, `sdk/log`, `otlploghttp` | 0.18.0 → 0.20.0 | — |
| `contrib/bridges/otelslog` | 0.17.0 → 0.19.0 | Error-valued attributes now land on the record's `Error` field and export as `exception.*` semconv attributes rather than a plain `error=` attribute. Assertions in `pkg/otel/otel_test.go` updated. |
| `testcontainers/testcontainers-go` | 0.41.0 → 0.43.0 | Migrated to the `moby/moby` modules: `HostConfigModifier` and `MappedPort` now use moby's `network.Port` instead of `docker/go-connections`' `nat.Port`. |
| `hashicorp/nomad/api` | → `v0.0.0-20260731161218` | `TaskGroup` gained `MaxRunDuration`, which appears in the serialized job fixture in `cmd/nacp/nacp_otel_test.go`. |
| `golang.org/x/crypto` | 0.50.0 → 0.54.0 | — |

## 2. Removing the Nomad main module

### Why

**Licensing.** NACP is MPL-2.0. `github.com/hashicorp/nomad/api` is also MPL-2.0 — HashiCorp
publishes it separately so clients can consume it. The Nomad *main* module, however, is
**BUSL-1.1** from v1.6.6 onwards. NACP was linking a BUSL-1.1 module to reach one 44-line
warning formatter.

**Weight.** A single *test-only* import reached into Nomad's server internals:

```
cmd/nacp.test → nomad/helper/tlsutil → nomad/nomad/structs/config → consul/api, vault/api, raft
```

That chain pulled in the AWS SDK (v1 and v2), Kubernetes `client-go`, Consul, Vault, Raft,
Serf, Memberlist, containerd, the Azure SDK, Google Cloud libraries and the cloud-discovery
providers — roughly half the module graph.

**Versioning.** Nomad's current release is v2.0.4, but its `go.mod` at that tag still declares
`module github.com/hashicorp/nomad` with no `/v2` suffix, so Go refuses it:

```
invalid version: module contains a go.mod file, so module path must match major version
```

`+incompatible` does not apply either, because the module *has* a `go.mod`. The main module was
therefore frozen at v1.11.3 with no supported upgrade path. Removing it makes the question moot.

### What was actually required

Three symbols, in two files. Only one was in production code:

| Symbol | Location | Sites |
| --- | --- | --- |
| `helper.MergeMultierrorWarnings` | `cmd/nacp/nacp.go:352` (**production**) + tests | 1 + 5 |
| `tlsutil.GenerateCA` / `GenerateCert` / `ParseSigner` / `Verify` | `cmd/nacp/nacp_test.go` (**test only**) | 4 |
| `file.WriteAtomicWithPerms` | `cmd/nacp/nacp_test.go` (**test only**) | 1 |

---

### 2a. The warning formatter — vendored from Nomad v1.6.3 (MPL-2.0)

`MergeMultierrorWarnings` builds the warning string NACP returns in Nomad's
`X-Nomad-Warnings` response header. Its exact output is user-visible, so behaviour had to stay
byte-identical.

**Approach: vendor the file verbatim from an MPL-2.0 era release, rather than reimplement it.**

Nomad's license history:

| Tag | LICENSE | `helper/warning.go` header | `Licensed Work:` parameter |
| --- | --- | --- | --- |
| v1.6.3 | MPL-2.0 | `SPDX-License-Identifier: MPL-2.0` | — |
| v1.6.5 | MPL-2.0 | `SPDX-License-Identifier: MPL-2.0` | — |
| v1.6.6 | BUSL-1.1 | `SPDX-License-Identifier: BUSL-1.1` | "Nomad Version **1.6.4** or later" |
| v1.7.0 | BUSL-1.1 | `SPDX-License-Identifier: BUSL-1.1` | "Nomad Version **1.7.0** or later" |

Note the inconsistency: v1.6.6's BUSL text names 1.6.4+ as the Licensed Work, which would reach
back over v1.6.4 and v1.6.5, while v1.7.0 restates it as 1.7.0+ — the earlier figure looks like
an error. Rather than rely on that reading, the file is vendored from **v1.6.3**, which predates
the boundary on either interpretation and is unambiguously MPL-2.0.

`helper/warning.go` is byte-identical at v1.6.3 and v1.6.5, and differs from v1.11.3 only in the
two header lines — the implementation is byte-for-byte identical throughout. Taking the MPL-2.0
copy therefore costs nothing functionally.

Why this is sound:

- **The MPL grant on v1.6.3 cannot be retroactively withdrawn.** MPL-2.0 §2.1 grants a
  world-wide, royalty-free licence to use, reproduce, modify and distribute. §5 is the only
  termination path and triggers on *the licensee's* non-compliance; the licence text contains no
  licensor revocation right. HashiCorp relicensed *subsequent* versions only.
- **Same-licence copying.** MPL-2.0 is file-level copyleft and NACP is already MPL-2.0, so an
  MPL-2.0 file moves into an MPL-2.0 project with no compatibility question and no change for
  downstream users.
- **Notices retained.** MPL-2.0 §3.4 forbids removing or altering licence notices, so the
  original header is kept verbatim.

This is also cleaner than a hand-written reimplementation, which — written after reading the
BUSL-1.1 source and deliberately matching its output format — would be a greyer position for a
function this small.

**Result:** new file `pkg/helper/warning.go` (51 lines).

```go
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Vendored verbatim from github.com/hashicorp/nomad at v1.6.3, helper/warning.go.
//
// Nomad switched to BUSL-1.1 at v1.6.6. v1.6.5 and earlier shipped under MPL-2.0,
// but v1.6.6's license names "Nomad Version 1.6.4 or later" as the Licensed Work ...

package helper
```

The package is deliberately named `helper`, mirroring upstream, so **call sites are unchanged** —
only the import path moved:

```diff
-"github.com/hashicorp/nomad/helper"
+"github.com/mxab/nacp/pkg/helper"
```

`helper.MergeMultierrorWarnings(allWarnings)` at `cmd/nacp/nacp.go:352` reads exactly as before.

---

### 2b. The TLS helpers — rewritten on the standard library

`nomad/helper/tlsutil` was used only by `generateTLSData` in `cmd/nacp/nacp_test.go`, which
builds a throwaway CA and server certificate for the mTLS proxy tests. This was the import
responsible for the bulk of the transitive dependency tree.

**Approach: replace with `crypto/x509` equivalents in a new test-only file,
`cmd/nacp/tlshelper_test.go` (173 lines).** The replacements keep the same names-and-shapes as
upstream so `generateTLSData` changes only at the call sites.

| Removed | Replacement |
| --- | --- |
| `tlsutil.CAOpts{Name, Days, PermittedDNSDomains}` | `caOpts{…}`, identical fields |
| `tlsutil.GenerateCA(opts)` | `generateCA(opts)` |
| `tlsutil.ParseSigner(pem)` | `parseSigner(pem)` |
| `tlsutil.CertOpts{Signer, CA, Name, Days, DNSNames, IPAddresses, ExtKeyUsage}` | `certOpts{…}`, identical fields |
| `tlsutil.GenerateCert(opts)` | `generateCert(opts)` |
| `tlsutil.Verify(ca, cert, name)` | `verifyCert(caPEM, certPEM, name)` |
| `file.WriteAtomicWithPerms(name, data, 0755, 0600)` | `os.WriteFile(name, data, 0600)` |

Implementation notes:

- **Keys:** ECDSA P-256, PEM-encoded as `EC PRIVATE KEY`; certificates as `CERTIFICATE`.
- **Serials:** 128-bit cryptographically random.
- **Validity:** `NotBefore` one minute in the past (clock skew), `NotAfter` at `Days`.
- **CA:** `IsCA`, `BasicConstraintsValid`, `KeyUsageCertSign | KeyUsageCRLSign | KeyUsageDigitalSignature`.
- **Name constraints preserved:** when `PermittedDNSDomains` is set, `PermittedDNSDomainsCritical`
  is also set, so the test still exercises constrained-CA verification rather than quietly
  weakening it.
- **Leaf:** carries the caller's `DNSNames`, `IPAddresses` and `ExtKeyUsage`, signed by the CA
  via the passed `crypto.Signer`.
- **`verifyCert`:** builds a root pool from the CA PEM and calls `cert.Verify` with the expected
  `DNSName` and both `ServerAuth` and `ClientAuth` key usages.

`WriteAtomicWithPerms` (temp file + `rename`) was replaced with a plain `os.WriteFile`: the test
writes into `t.TempDir()`, where atomicity buys nothing. The `0755` directory mode argument is
dropped as the directory already exists.

---

## Results

`go.mod` now references only `github.com/hashicorp/nomad/api`. No BUSL-1.1 code ships, links, or
appears in `go.sum`.

| | Before | After |
| --- | --- | --- |
| `go.sum` lines | 710 | **397** (−44%) |
| Indirect dependencies | 179 | **121** |
| Modules in graph | 510 | **258** (−49%) |

(Measured against `337fe48`, i.e. after the dependency update and before the Nomad removal.)

## Verification

`gofmt`, `go build ./...` and `go vet ./...` are clean, and the full test suite passes. Checks
targeting the two substitutions specifically:

| Check | Covers |
| --- | --- |
| `go test ./cmd/nacp/ -run TestProxy` (16/16 subtests) | Vendored warning formatter produces the expected strings |
| `go test ./cmd/nacp/ -run 'TestCreateTlsConfig\|TestBuildCustomTransport'` | The stdlib CA/cert helpers |
| `go test ./pkg/otel/ ./pkg/admissionctrl/notation/` | Docker-backed tests unaffected by the removal |
| `grep hashicorp/nomad go.mod` | Only `nomad/api` remains |

Note that `pkg/admissionctrl/notation` and `pkg/otel` require a running Docker daemon
(testcontainers). Under heavy Docker load these can time out for reasons unrelated to this
change.
