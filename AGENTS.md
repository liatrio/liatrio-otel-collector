# liatrio-otel-collector

Guidance for AI agents (and humans) working in this repository. This is the source of truth;
`CLAUDE.md` is a symlink to it.

`liatrio-otel-collector` is a custom OpenTelemetry Collector distribution. It bundles upstream
collector/contrib components with a small set of Liatrio-authored components, most of which scrape
software-delivery metrics from VCS platforms (GitHub, GitLab, Azure DevOps). The custom components
are intended for eventual upstream contribution to opentelemetry-collector-contrib, so they follow
upstream conventions closely.

## Domain Glossary

- **Distribution** — a collector binary built from a chosen set of components. This repo is one such
  distribution; its binary is `otelcol-custom`.
- **Component** — a receiver, processor, exporter, connector, or extension. The custom components
  here are four receivers, one processor, and one extension.
- **OCB** (OpenTelemetry Collector Builder) — `go.opentelemetry.io/collector/cmd/builder`; assembles
  the distribution binary from `config/manifest.yaml`. Invoked by `make build`.
- **`config/manifest.yaml`** — the OCB build manifest and the **only** place components are wired
  into the build (`gomod:` lines plus a local `replaces:` block). There is no hand-written registry.
- **Factory pattern** — the upstream OTel registration idiom; each component exposes `NewFactory()`.
- **Scraper** — a pull-based metrics source inside a receiver. Three receivers use a pluggable
  **multi-scraper** design (see [Receiver scraper pattern](#receiver-scraper-pattern)).
- **`metadata.yaml`** — the declarative source describing a component (type, status, attributes,
  metrics). Input to mdatagen.
- **mdatagen** — code generator that turns a component's `metadata.yaml` into its
  `internal/metadata/` package and `documentation.md`.
- **genqlient** — GraphQL client generator (Khan/genqlient) that turns `schema.graphql` +
  `genqlient.graphql` into generated Go for GraphQL-based scrapers.
- **semconv** — OpenTelemetry Semantic Conventions. The **spec** and the Go **package** version
  separately; see [`docs/semantic-conventions.md`](docs/semantic-conventions.md).
- **multimod** — OTel build tool that manages module-set versions from `versions.yaml` at release.
- **crosslink** — OTel build tool that syncs/prunes `replace` directives across module `go.mod`s.
- **otel-compgen** — in-repo CLI (`cmd/otel-compgen`) that scaffolds new components.

## Repository layout

A **Go multi-module monorepo**: each custom component is its own Go module with its own
`go.mod`/`go.sum`/`Makefile`. The root module (`github.com/liatrio/liatrio-otel-collector`) is mostly
build/config glue.

- `receiver/githubreceiver`, `receiver/gitlabreceiver`, `receiver/githubactionsreceiver`,
  `receiver/azuredevopsreceiver` — custom receivers
- `processor/gitlabprocessor` — custom processor
- `extension/githubappauthextension` — GitHub App auth extension
- `cmd/otel-compgen` — component scaffolding CLI (separate module `github.com/liatrio/otel-compgen`)
- `config/manifest.yaml` — OCB build manifest (the component wiring)
- `config/config.yaml` — runtime collector config used by `make run`
- `internal/tools` — pinned build-tool dependencies (`tools.go`); binaries build into `.tools/`
- `versions.yaml` — module-set versions managed by multimod for releases
- `docs/` — cross-cutting policy/reference docs (e.g. `semantic-conventions.md`)
- `build/` — OCB output (generated `main.go` + compiled `otelcol-custom`); do not hand-edit

## Common commands

All commands are Make targets. `install-tools` builds pinned tool binaries from `internal/tools`
into `.tools/`; most targets depend on it. Multi-module targets (`*-all`) fan out over every
directory containing a `go.mod` (excluding `build/` and `tmp/`) via the `for-all` helper.

```bash
make build          # build the otelcol-custom binary via OCB into ./build
make run            # build then run with config/config.yaml
make build-debug    # build with delve debug symbols (for the VS Code debug flow)
make dockerbuild    # cross-build linux/amd64 and produce a local Docker image

make checks         # full local CI gate: generate, fmt-all, tidy-all, lint-all, test-all, scan-all, multimod-verify, crosslink
make lint-all       # golangci-lint across all modules
make test-all       # go test across all modules (with coverage)
make generate       # run `go generate` (mdatagen + genqlient) across all modules
make tidy-all       # go mod tidy across all modules
make crosslink      # sync/prune replace directives across module go.mod files
```

Per-module targets (run from inside a component directory, defined in `Makefile.Common`):

```bash
make test           # go test -v ./... with coverage for this module
make lint           # golangci-lint run
make gen            # go generate ./... then fmt (regenerate this component's code)
make fmt            # goimports + go fmt + go mod tidy
```

### Running a single test

`cd` into the component's module first (tests only see their own module), then use plain `go test`:

```bash
cd receiver/githubreceiver
go test ./internal/scraper/githubscraper/ -run TestScrape -v
```

## Code generation (critical)

Two independent codegen pipelines run from `//go:generate` directives (usually in each component's
`doc.go`). Generated files are **never hand-edited** — they are marked `DO NOT EDIT` and flagged
`linguist-generated` in `.gitattributes`.

1. **mdatagen** — `.tools/mdatagen metadata.yaml`. Consumes a component's `metadata.yaml` and
   produces the entire `internal/metadata/` package (`generated_config.go`, `generated_metrics.go`,
   `generated_resource.go`, `generated_status.go`), the root `generated_*_test.go` files, and
   `documentation.md`. Hand-written factories consume the output (`metadata.Type`,
   `metadata.MetricsStability`, `metadata.NewDefaultMetricsBuilderConfig()`, etc.). To add or change
   a metric/attribute, edit `metadata.yaml` and run `make gen`.

2. **genqlient** — `.tools/genqlient`. GraphQL-based scrapers keep three hand-written inputs —
   `schema.graphql` (upstream API schema), `genqlient.graphql` (queries), `genqlient.yaml` (config +
   scalar bindings) — and generate `generated_graphql.go` / `generated.go`.

CI fails if regenerating produces a diff, so **run `make generate` and include the regenerated files
in the same change** after touching any `metadata.yaml`, `.graphql`, or genqlient config. Editing templates in `cmd/otel-compgen`
requires rebuilding that binary — templates are compiled in via `//go:embed`.

## Component architecture

Custom components use the standard upstream OTel factory pattern. Each exposes `NewFactory()` built
from `metadata.Type`, a `createDefaultConfig`, and signal-specific create funcs gated by generated
stability constants (e.g. `receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability)`).
Config types embed upstream config structs with `mapstructure:",squash"`, assert interface
conformance with blank-var checks (`_ component.Config = (*Config)(nil)`), and implement `Validate()`.

### Receiver scraper pattern

Three receivers (`githubreceiver`, `gitlabreceiver`, `azuredevopsreceiver`) use a pluggable
multi-scraper design:

- `receiver/<name>/internal/scraper.go` defines a `ScraperFactory` interface
  (`CreateDefaultConfig`, `CreateMetricsScraper`).
- Concrete scrapers live under `receiver/<name>/internal/scraper/<xscraper>/`, each with its own
  `factory.go` (a `TypeStr` const + `Factory` struct implementing `ScraperFactory`), `config.go`,
  scrape logic, and — for GraphQL scrapers — their own genqlient files.
- The receiver's root `factory.go` registers scrapers in a `scraperFactories map[string]ScraperFactory`
  keyed by `TypeStr`, and builds the receiver via `scraperhelper.NewMetricsController`.
- Config binds scrapers dynamically: `Config.Scrapers map[string]internal.Config` plus a custom
  `Unmarshal` that resolves each key to its factory's default config. `Validate()` requires at least
  one scraper.

`githubreceiver` is the richest: it supports both metrics (scrapers) and traces (a webhook), so its
factory registers `WithMetrics` and `WithTraces`. `githubactionsreceiver` is the exception — it has
**no scrapers**; it is a webhook/push traces-only receiver.

### Creating a new component

Use otel-compgen (see `cmd/otel-compgen/README.md`). It scaffolds a minimal, compiling skeleton
(factory, config, doc, metadata.yaml, Makefile importing `Makefile.Common`, etc.) from a full module
path, e.g. `otel-compgen receiver github.com/liatrio/liatrio-otel-collector/receiver/myreceiver ./receiver/myreceiver`.
Only the receiver (pull/scraper metrics) path is implemented. After scaffolding: run `make gen`,
implement the component logic, and add the module to `config/manifest.yaml`.

## Semantic conventions

**Rule of thumb: when in doubt, default to the semconv spec** — but to the spec version our packages
declare, not whatever is newest upstream. The **spec** (human-readable conventions, latest 1.43) and
the **package** (`otel/semconv/vX` Go constants, latest 1.41) are separate version axes; the package
lags the spec and is the hard ceiling (mdatagen can't stamp an import that doesn't exist). **Renovate
does not bump the semconv package** — that is a manual step. Where the spec has no convention, we
define local "extension" attributes; when the spec defines one, the spec name replaces the extension.
The repo is **not** on a single version (declared `sem_conv_version` ranges 1.27.0–1.37.0 and differs
within components), so verify before relying on it. Full policy, the version matrix, the extensions
register, and the package-upgrade/OCB/Renovate runbook live in
[`docs/semantic-conventions.md`](docs/semantic-conventions.md).

## Conventions & CI

- Commits must be **Conventional Commits** (enforced by a `commit-msg` pre-commit hook and a PR check).
- `pre-commit install` is expected; hooks run `make check-generate`, yamlfmt, json formatting, and markdownlint.
- Lint config is `.golangci.yaml` (Go 1.24, line length 185). Formatting is `goimports` + `gofmt`.
- CI (`.github/workflows/build.yml`) only runs real jobs on PRs targeting `main`. It re-runs
  `make generate`, `make tidy-all`, and `make crosslink` and fails on any resulting diff — so keep
  generated code, module tidiness, and replace directives in sync.
- Releases are automated on `main` via multimod + semantic versioning driven by `versions.yaml`
  (the `liatrio-otel` module set). Full pipeline in [`docs/releasing.md`](docs/releasing.md).

## Guardrails

- **Never hand-edit generated files** — `internal/metadata/generated_*.go`, `generated_graphql.go` /
  `generated.go`, `documentation.md`, and `internal/metadata/testdata/config.yaml`. Change the source
  (`metadata.yaml`, `.graphql`, `genqlient.yaml`) and regenerate with `make gen` / `make generate`.
- **Regenerate and re-tidy after touching inputs.** After editing any `metadata.yaml`, `.graphql`,
  genqlient config, or dependencies, run `make generate` and `make tidy-all` (and `make crosslink`
  if `replace`/module wiring changed) and include the regenerated files in the change. CI fails on
  any diff.
- **Adding or removing a component means editing `config/manifest.yaml`** and running `make crosslink`
  — there is no other registry.
- **Do not bump the semconv package as a side effect.** It is a deliberate manual migration; follow
  the runbook in [`docs/semantic-conventions.md`](docs/semantic-conventions.md).
- **Never commit unless explicitly asked**, and never `git commit --no-verify` — it bypasses the
  conventional-commit and generate checks.
- **No destructive git operations without confirmation**: `push --force`, `reset --hard`,
  `branch -D`, or history rewrites on `main`. Branch off `main`; do not work on `main` directly.
- **Verify before claiming done.** For the changed module, run `make build` (or `make lint` +
  `make test`), plus `make generate` if you touched generated inputs. Do not claim a change works
  without building it.

## Keeping docs current

When you add or rename a component, change a build/test command, alter the codegen or scraper
architecture, or change the semconv posture, update docs **as part of the change, not a follow-up**:

- **`AGENTS.md`** (this file; `CLAUDE.md` is a symlink to it) — if it changes how an agent navigates,
  builds, or reasons about the repo.
- **`docs/semantic-conventions.md`** — if the semconv version matrix, extensions register, or upgrade
  process changes.
- **The component's own `README.md` and `metadata.yaml`** — regenerate `documentation.md` afterward.
