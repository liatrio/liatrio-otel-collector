# Release Process

Releases are **maintainer-triggered**: a dispatched workflow opens a release PR, and merging it tags
and publishes. This document describes the pipeline, the role of [multimod](#multimod), and how to verify a release locally.

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
| `prerelease` | `make multimod-prerelease` + CI (`cut-release.yml`) | Rewrites the `go.mod` versions across the set and creates the `Prepare liatrio-otel for version vX` commit |
| `tag` | CI (`create-release.yml`) | Creates a git tag per module in the set at the merged commit |

`make multimod-prerelease` runs `multimod prerelease -b=false -v ./versions.yaml -m liatrio-otel`
(`-b=false` = operate on the current branch instead of creating a new one).

## The pipeline

A maintainer decides when to release. Everything merged since the last root `v*` tag ships together
as one release. Merging a PR to `main` never releases by itself.

```text
Actions → cut-release → Run workflow (main, bump: auto | patch | minor)
      │
      ▼
cut-release.yml                        (workflow_dispatch, main only)
  • compute the version (see "Version bumping")
  • branch release/vX: yq-write versions.yaml, commit "chore: prepare release vX"
  • make multimod-prerelease  →  commit "Prepare liatrio-otel for version vX"
  • push the branch, open PR "chore(release): prepare liatrio-otel vX" with a notes preview
      │
      ▼
release PR: normal PR CI, code-owner review, squash merge
      │
      ▼
create-release.yml                     (push to main touching versions.yaml, or workflow_dispatch)
  • read vX from versions.yaml; exit 0 if tag vX already exists
  • gitsign-sign and multimod tag --module-set-name liatrio-otel --commit-hash <sha>
  • git push --tags
  • gh release create vX --generate-notes (since the previous v* tag)
      │
      ▼
release.yml : release                  (trigger: GitHub Release "created")
  • GoReleaser Pro (.goreleaser.yaml): cross-platform binaries + archives,
    multi-arch Docker images (linux 386/amd64/arm64) pushed to ghcr.io
```

Both release workflows mint the `semantic-release` octo-sts identity, whose trust policy only allows
those two workflow files on `main`. The octo-sts token (not `GITHUB_TOKEN`) is used so the release PR
gets normal CI and the created Release fires `release.yml`. Both share the `release` concurrency
group.

`cut-release` refuses to run if the tag, the `release/vX` branch, or an open PR for it already
exists, or if `versions.yaml` holds a version that was prepared but never tagged (publish that first,
see [Recovery](#recovery)).

### Version bumping

Commit types only pick the size of the bump. They no longer decide whether a release happens.

- `auto` (default): `thenativeweb/get-next-version` reads Conventional Commits since the last tag
  (`feat:` → minor, `fix:` → patch). If nothing releasable is found (e.g. only `chore:` commits), it
  falls back to patch, since dispatching means you want a release.
- `patch` / `minor`: bump from the latest root `v*` tag regardless of commit types.

### Tag signing

multimod always creates signed tags (`git tag -a -s`). `create-release` signs them keylessly with
[gitsign][gitsign] using the job's GitHub Actions OIDC identity, so there is no long-lived signing key
to rotate or expire. GitHub shows these tags as "Unverified" (it does not trust Sigstore's CA), but
they can be checked locally:

```bash
git fetch --tags
gitsign verify-tag v0.104.4 \
  --certificate-identity-regexp '^https://github.com/liatrio/liatrio-otel-collector/\.github/workflows/create-release\.yml@refs/heads/main$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

### Recovery

`create-release` is idempotent. If a release PR merged but tagging or publishing failed, re-run it
from Actions (`workflow_dispatch` on `main`): it tags `main` HEAD at the `versions.yaml` version if
that tag is missing, and no-ops otherwise. If the tags were pushed but the Release was not created,
run `gh release create vX --generate-notes --notes-start-tag <previous> --verify-tag` with a token
other than `GITHUB_TOKEN`, so `release.yml` fires.

## Verifying locally

You cannot cut a release locally (it requires the bot identity and the workflow's OIDC identity for signing), but you can
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
[gitsign]: https://github.com/sigstore/gitsign
