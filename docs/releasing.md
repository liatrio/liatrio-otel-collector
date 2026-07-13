# Release Process

Releases are **fully automated** off `main` — no manual version bumps or tagging. This document
describes the pipeline, the role of [multimod](#multimod), and how to verify a release locally.

## multimod

[multimod][multimod] (`go.opentelemetry.io/build-tools/multimod`, pinned in `internal/tools`) adds
versioning to a repo containing multiple Go modules. It versions a **module set** — a named group of
modules released together — driven by `versions.yaml`. It is the standard release mechanism across
the upstream OpenTelemetry collector repos.

`versions.yaml` defines the set:

```yaml
module-sets:
  liatrio-otel:
    version: v0.102.1        # the set's current version
    modules: [ ... ]         # modules versioned + tagged together
excluded-modules: [ ... ]    # modules deliberately NOT versioned
```

Subcommands used here:

| Subcommand | Where | What it does |
|---|---|---|
| `verify` | `make multimod-verify` (in `make checks`) | Validates `versions.yaml`: every module in exactly one set, semver sanity |
| `prerelease` | `make multimod-prerelease` + CI | Rewrites the `go.mod` versions across the set and creates the `Prepare liatrio-otel for version vX` commit |
| `tag` | CI (`create-release.yml`) | Creates a git tag per module in the set at the merged commit |

`make multimod-prerelease` runs `multimod prerelease -b=false -v ./versions.yaml -m liatrio-otel`
(`-b=false` = operate on the current branch instead of creating a new one).

## The pipeline

Three workflows chain together, gated by the commit author (`octo-sts[bot]`) and message. A single
merge to `main` walks all the way to published artifacts with no human action after the merge.

```
merge PR to main
      │
      ▼
build.yml : multimod-prepare-release   (actor ≠ octo-sts[bot])
  • compute next version from Conventional Commits
  • yq-write it into versions.yaml
  • commit "chore: run multimod to update versions ahead of release(version vX)"
  • make multimod-prerelease  →  commit "Prepare liatrio-otel for version vX"
  • push to main (as octo-sts[bot])
      │
      ▼
create-release.yml : create-release    (actor = octo-sts[bot], message starts "Prepare liatrio-otel for version")
  • build changelog from Conventional Commits
  • GPG-sign, multimod tag --module-set-name liatrio-otel --commit-hash <sha>
  • git push --tags
  • gh release create <tag> --notes <changelog>
      │
      ▼
release.yml : release                  (trigger: GitHub Release "created")
  • GoReleaser Pro (.goreleaser.yaml): cross-platform binaries + archives,
    multi-arch Docker images (linux 386/amd64/arm64) pushed to ghcr.io
```

The `octo-sts[bot]` author gate is what prevents infinite loops: `build.yml` jobs skip when the
actor **is** the bot, and `create-release.yml` only runs when it **is** the bot with a
`Prepare liatrio-otel for version` commit.

### Version bumping

The next version is derived from Conventional Commit types since the last release (via
`thenativeweb/get-next-version` in `build.yml`, and `mathieudutour/github-tag-action` for the
changelog in `create-release.yml`): `feat:` → minor, `fix:` / `chore:` → patch. This is why commit
message discipline matters — commits drive the version.

## Verifying locally

You cannot cut a release locally (it requires the bot identity and signing keys), but you can
validate the versioning config and dry-run the artifact build:

```bash
make multimod-verify   # validate versions.yaml (also runs as part of `make checks`)
make release           # local goreleaser snapshot (no publish) — see the Makefile note on OSS vs Pro
```

Note: `make release` uses the OSS GoReleaser installed via `internal/tools`, which does not support
the `partial:` option used in CI (`goreleaser-pro`); it is a smoke test, not a faithful CI build.

## Module-set membership caveat

`versions.yaml` versions and tags **five** modules: the root module, `githubreceiver`,
`gitlabreceiver`, `githubactionsreceiver`, and `githubappauthextension`. `internal/tools` and
`otel-compgen` are explicitly excluded.

`receiver/azuredevopsreceiver` and `processor/gitlabprocessor` are their own Go modules and ship in
the build (`config/manifest.yaml`), but are **neither in the module set nor in `excluded-modules`**,
so multimod does not version or tag them with the rest of the release. Confirm this is intentional
before relying on their tags; if it is not, add them to the `liatrio-otel` module set.

[multimod]: https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/multimod
