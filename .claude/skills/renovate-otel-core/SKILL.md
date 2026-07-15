---
name: renovate-otel-core
description: >-
  Renovate maintenance playbook, read by the `/fix` skill, covering the OTel collector core +
  contrib + API/SDK lockstep group (Renovate group label `group:otel-core-contrib`). Select when the
  PR moves `go.opentelemetry.io/collector` / `collector/**`,
  `github.com/open-telemetry/opentelemetry-collector-contrib/**`, or `go.opentelemetry.io/otel` /
  `otel/**` — plus the `mdatagen`/`builder` tools — which must move in lockstep or OCB fails to
  assemble the distribution. Failure signatures are OCB / builder / mdatagen version-mismatch errors
  on `make build`. Move the whole group or none; a green module-local check is NOT sufficient (the
  full OCB build is the gate). If the core bump surfaces semconv errors, overlay `renovate-semconv`.
---

# Playbook: OTel collector core & contrib (lockstep group)

Selected by the **`group:otel-core-contrib`** label, which Renovate applies to its
`otel collector core & contrib deps` group. This is the repository's single most consequential
dependency group: it moves the collector core, the contrib components, and the OTel API/SDK modules
**in lockstep**, because OCB requires every component compiled into the distribution — plus the
`mdatagen`/`builder` tools that generate and assemble it — to share a compatible core version.

## Why these modules move together (the lockstep rationale)

Lifted and expanded from the `description` on this group in
[`renovate.json`](../../../renovate.json):

- **OCB compatibility.** The OpenTelemetry Collector Builder assembles the `otelcol-custom` binary
  from `config/manifest.yaml`. Every `go.opentelemetry.io/collector*` and
  `opentelemetry-collector-contrib/*` dependency — across the manifest, every component `go.mod`,
  and `internal/tools/go.mod`'s `mdatagen`/`builder` — must be on a mutually compatible core
  version. Bumping them one-at-a-time produces a build where components disagree on the core API and
  OCB fails to assemble.
- **The OTel API/SDK belongs here too (Open Question #5, resolved: include).**
  `go.opentelemetry.io/otel` and `go.opentelemetry.io/otel/**` (the API/SDK, distinct from
  `.../collector`) are **direct** dependencies of the custom receivers and are pinned to a version
  the collector core requires. Before this decision they matched neither the include nor the exclude
  list and fell through to generic gomod handling, so Renovate proposed them as independent
  one-at-a-time PRs — reintroducing exactly the version skew this group exists to prevent (a receiver
  on a newer `otel/metric` than the collector core it compiles into). They are now in
  `matchPackageNames` so a single PR moves core + contrib + API/SDK as one coherent set.
- **Rule ordering.** This rule is placed **last** in `packageRules` so it overrides the generic
  `tool deps` grouping for `mdatagen`/`builder` specifically (last-rule-wins).

## Authoritative docs

- [`AGENTS.md` → Domain Glossary](../../../AGENTS.md#domain-glossary) — OCB, `config/manifest.yaml`,
  multimod, crosslink.
- [`AGENTS.md` → Code generation](../../../AGENTS.md#code-generation-critical) — mdatagen/genqlient;
  regenerate and re-tidy after any dependency change.
- [`AGENTS.md` → Guardrails](../../../AGENTS.md#guardrails) — adding/removing a component means editing
  `config/manifest.yaml` + `make crosslink`.
- [`config/manifest.yaml`](../../../config/manifest.yaml) — the only place components are wired into
  the build.

## Failure modes → remediation

| Failure signature | Likely cause | Remediation |
| --- | --- | --- |
| OCB `make build` fails on incompatible core versions | one collector/contrib module bumped without the others | ensure the whole group moved together; run `make generate`, `make tidy-all`, `make crosslink`, then `make build` |
| `make crosslink` rewrites `replace` directives | inter-module `replace`s drifted after the bump | commit the crosslink result |
| `make generate` diff after a core bump | mdatagen output changed with the new core | commit regenerated `internal/metadata/` + `documentation.md`; never hand-edit them |
| a `go.opentelemetry.io/otel/*` **major** bump drops a require on tidy | major bump is an import-path migration | rewrite the import paths to the new major first, then `make tidy-all` (do not accept the drop — this is the backoff/v7 class of defect) |
| compiler errors mentioning `semconv` | the core bump pulled a semconv change | switch to [`renovate-semconv`](../renovate-semconv/SKILL.md); do **not** bump the semconv package to make it compile |

## Hard invariants / guardrails

- **Move the whole group or none of it.** Never merge a partial lockstep bump.
- **Never bump the semconv package as a side effect** of a core bump — defer to
  [`renovate-semconv`](../renovate-semconv/SKILL.md) / [`docs/semantic-conventions.md`](../../../docs/semantic-conventions.md).
- **Never hand-edit generated files**; regenerate with `make generate` and commit the result.
- **Adding/removing a component is a `config/manifest.yaml` + `make crosslink` change** — there is no
  other registry.
- The full gate for this class is the complete OCB build: `make generate` diff-check, `make
  lint-all`, `make test-all`, `make build`. A green module-local check is **not** sufficient here.
