---
group: _default
packages: []
failure_signatures: []
---

# Playbook: default (fallback)

The fallback playbook used when a Renovate PR matches **no** group label and **no** other
playbook's `packages:` globs. It carries only repository-wide conventions — it deliberately holds no
dependency-specific knowledge. When you find yourself repeatedly hand-holding a class of bump that
lands here, that is the signal to add a dedicated playbook (see
[`README.md`](./README.md#adding-a-playbook)).

## Authoritative docs

Read these before changing anything; they are the source of truth this playbook refuses to
duplicate:

- [`AGENTS.md`](../../AGENTS.md) — repository conventions, the codegen pipelines, the multi-module
  layout, and the [Guardrails](../../AGENTS.md#guardrails) section.
- [`AGENTS.md` → Common commands](../../AGENTS.md#common-commands) — the `make` targets
  (`generate`, `tidy-all`, `crosslink`, `lint-all`, `test-all`, `build`) a fix must keep green.
- [`docs/semantic-conventions.md`](../semantic-conventions.md) — only relevant if the bump surfaces
  a semconv migration; if so, switch to [`semconv.md`](./semconv.md).

## Failure modes → remediation

| Failure signature | Likely cause | Remediation |
| --- | --- | --- |
| `make generate` produces a diff | stale generated code after a dependency or `metadata.yaml`-adjacent change | run `make generate` and commit the regenerated files (never hand-edit `generated_*.go` / `documentation.md`) |
| `go mod tidy` / `make tidy-all` drops a just-added require | a **major** Go module bump is an import-path migration; the new path is unused until imports are rewritten | do **not** accept the drop — rewrite import paths to the new major version first, then tidy (see [`otel-core.md`](./otel-core.md) for the lockstep variant) |
| `make crosslink` changes `replace` directives | module wiring drifted | commit the crosslink result; if a component was added/removed, `config/manifest.yaml` also needs editing |
| lint failure new to this PR | formatting or a newly-flagged construct | run the module-local `make fmt` / `make lint`; fix only what the bump introduced |

## Hard invariants / guardrails

These are non-negotiable regardless of which dependency triggered the PR:

- **Never hand-edit generated files** — change the source (`metadata.yaml`, `.graphql`, genqlient
  config) and regenerate. See [`AGENTS.md` → Guardrails](../../AGENTS.md#guardrails).
- **Never `git commit --no-verify`** — the conventional-commit and `check-generate` hooks must run.
- **Never touch `.github/**`** during a fix.
- **Never work on `main`** — fixes land on the PR's `renovate/**` branch only.
- **Do not bump the semconv package as a side effect** — it is a deliberate manual migration
  ([`docs/semantic-conventions.md`](../semantic-conventions.md)).
- Keep commits concern-separated: regeneration in one commit, mechanical adaptation in another.
