# OpenWiki CLI installer

Pins the [OpenWiki](https://www.npmjs.com/package/openwiki) CLI used by
`.github/workflows/openwiki-update.yml`. Nothing here is part of the NACP Go
build — it exists only so the scheduled documentation job installs a known,
reproducible dependency tree.

This is **not** the generated documentation. That lives in `openwiki/` at the
repository root.

## Why a manifest instead of `npm install --global openwiki@<version>`

Pinning `openwiki` alone pins nothing beneath it. Its 24 direct dependencies are
almost all `^` ranges (`@langchain/*`, `@aws-sdk/client-bedrock-runtime`,
`google-auth-library`, `posthog-node`, `react`, `ink`, …), so a global install
re-resolved a fresh ~198-package tree on every daily run. The committed
`pnpm-lock.yaml` pins that entire tree by integrity hash, and CI installs with
`--frozen-lockfile` so drift fails the job rather than silently re-resolving.

Separately, `npm` runs `preinstall`/`install`/`postinstall` for every package in
the tree and offers no per-package control — `--ignore-scripts` is all or
nothing. pnpm blocks dependency lifecycle scripts by default and takes an
explicit allowlist, which is why this directory uses pnpm.

## How builds are gated

The workflow installs with `--ignore-scripts`, so nothing in the tree executes
code at install time, and then rebuilds the single package that needs a build by
name:

```sh
pnpm install --frozen-lockfile --ignore-scripts
pnpm rebuild better-sqlite3
```

`pnpm-workspace.yaml` independently allows exactly that one package to build:

```yaml
allowBuilds:
  better-sqlite3: true
```

`better-sqlite3` ships a native addon built by its `install` script
(`prebuild-install || node-gyp rebuild --release`). OpenWiki reaches it through
`@langchain/langgraph-checkpoint-sqlite`. It is genuinely required — with the
build skipped, the CLI starts but any real run dies with `Could not locate the
bindings file`. Every other package in the tree is blocked, by both gates.

The rebuild step names the package deliberately. `pnpm rebuild --pending` would
rebuild everything the install skipped, which is worse than not passing
`--ignore-scripts` at all.

Before adding an entry here, confirm the package actually fails without its
build:

```sh
rm -rf node_modules
pnpm install --frozen-lockfile --ignore-scripts
pnpm rebuild better-sqlite3
cd ../.. && .github/openwiki-cli/node_modules/.bin/openwiki code --update --print
```

## Bumping the CLI

```sh
cd .github/openwiki-cli
pnpm add openwiki@<version>   # updates package.json and pnpm-lock.yaml
```

Commit both files. A new version may pull in a different set of packages wanting
to run code, and `--ignore-scripts` suppresses pnpm's usual "Ignored build
scripts" summary, so check for new build requirements deliberately:

```sh
pnpm install --frozen-lockfile   # without --ignore-scripts, just to read the list
```

If that surfaces a package that genuinely needs its build, add it to
`allowBuilds` **and** to the rebuild step in the workflow — the allowlist alone
no longer causes anything to build.

The pinned `packageManager` field keeps local and CI pnpm on the same version so
the lockfile format stays compatible; `pnpm/action-setup` reads it directly.
