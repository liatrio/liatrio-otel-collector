# Semantic Conventions Policy

This document is the single source of truth for how this distribution relates to OpenTelemetry
[Semantic Conventions][semconv] (semconv), which version(s) we currently target, and the process for
keeping that current.

## Rule of thumb: when in doubt, default to the spec

Every attribute, metric, resource, and namespace we emit should follow the upstream semconv
**spec** unless there is no convention for it yet. Reuse the spec's existing names for anything it
already defines, and match its naming exactly rather than adapting it for convenience.

**Follow the version of the spec that our packages declare, not whatever is newest upstream.** See
[Spec vs. package](#spec-vs-package): the spec moves faster than the Go package, and we can only
emit what the pinned package provides constants for. When referencing the spec while writing a
component, use the spec revision matching that component's declared `sem_conv_version`.

Where the spec has no convention for something, we define a **local extension**, following the
shape and namespace of the closest existing spec area (e.g. new VCS attributes live under `vcs.*`,
following the patterns of the existing `vcs.*` / `deploy.*` conventions). Local extensions are
recorded in the [Extensions register](#extensions-register). When the spec defines a convention that
an extension covers, the spec name replaces the extension.

## Spec vs. package

Two different things carry a semconv version number, and they do not move together:

| | What it is | Latest | Bumped by |
|---|---|---|---|
| **The spec** | Human-readable conventions ([opentelemetry.io][semconv] / the [semantic-conventions repo][sc-repo]) | **1.43** | N/A — published upstream |
| **The package** | `go.opentelemetry.io/otel/semconv/vX` Go constants we import | **1.41** (two versions behind the spec) | **Manually** — Renovate does **not** bump the version selection (see [runbook](#package-upgrade--ocb--renovate-runbook)) |

Consequences of the gap:

- The package is the **hard ceiling**. mdatagen stamps an `otel/semconv/vX` import derived from a
  component's `sem_conv_version`, so a component cannot declare a spec version with no corresponding
  package — there is no `otel/semconv/v1.43.0` to import.
- The newest package (1.41) is two versions behind the spec (1.43). Spec conventions introduced
  after 1.41 have no Go constants in this distribution.

## Current state (verify before trusting — see [How to verify](#how-to-verify))

> **We are not on a single version.** Declared `sem_conv_version` ranges from **1.27.0 to 1.37.0**
> across components; the newest package available upstream is **1.41**; the newest spec is **1.43**.
> Within a component, generated and hand-written code can target different package versions. The
> table is what is actually in the tree.

| Component | `sem_conv_version` (declared spec) | Generated code import (package) | Hand-written code import (package) | Notes |
|---|---|---|---|---|
| `receiver/githubreceiver` | 1.27.0 | `otel/semconv/v1.27.0` | `otel/semconv/v1.34.0` | Intra-component split: metrics gen on 1.27.0, trace code (`model.go`, `trace_event_handling.go`) on 1.34.0 |
| `receiver/gitlabreceiver` | 1.27.0 | `otel/semconv/v1.27.0` | — | |
| `processor/gitlabprocessor` | 1.27.0 | — | — | |
| `receiver/azuredevopsreceiver` | 1.37.0 | `otel/semconv/v1.37.0` | — | Uses `vcs.provider.name` |

The two version signals per component and how they relate:

1. **`sem_conv_version:` in `metadata.yaml`** — declares the **spec** revision the component follows.
   mdatagen stamps a matching `otel/semconv/vX` **package** import into
   `internal/metadata/generated_metrics.go`, which is why this value cannot exceed an available
   package version.
2. **`otel/semconv/vX` imports in hand-written Go** — the **package** constants the runtime code
   uses. Chosen by hand; can differ from the declared `sem_conv_version`.

## How to verify

Re-derive the table above at any time — do not trust a hardcoded number:

```bash
# Declared spec version per component
grep -rn 'sem_conv_version' --include='metadata.yaml' .

# Actual package imports in Go (generated + hand-written), grouped
grep -rn 'go.opentelemetry.io/\(collector\|otel\)/semconv/v' --include='*.go' . | sort -u
```

- **Latest spec:** the [semantic-conventions releases][sc-releases] / [changelog][sc-changelog].
- **Latest package:** the newest published `go.opentelemetry.io/otel/semconv/vX.Y.Z` (the Go module
  proxy / [pkg.go.dev][pkg]). This is the ceiling for `sem_conv_version`.

Record what you find rather than relying on memory — both move, and the package trails the spec.

## Extensions register

Attributes/metrics we emit that are not in the spec, or where we diverge from it.

| Extension | Where | Spec status |
|---|---|---|
| `vcs.vendor.name` | `receiver/githubreceiver` | Superseded — the spec renamed this to `vcs.provider.name`, which `azuredevopsreceiver` uses |
| `work_item.*` | `receiver/azuredevopsreceiver` | No spec convention for work items |
| `cve.severity` | `receiver/githubreceiver` | No verified spec equivalent under `cve.*` / vulnerability conventions |

## Package upgrade & OCB / Renovate runbook

**Renovate bumps OCB and the `go.opentelemetry.io/collector` / contrib modules in one large PR, but
it does *not* bump the semconv package selection.** The `otel/semconv/vX` version — both the
`sem_conv_version` in each `metadata.yaml` and the hand-written imports — is chosen by us and is
updated manually as part of a deliberate migration. An OCB bump does not move semconv.

To move a component to a newer semconv **package**:

1. **Pick the target package version** — no higher than the newest published `otel/semconv/vX`
   (currently 1.41).
2. **Diff the spec across the jump** — read the [changelog][sc-changelog] between the component's
   current `sem_conv_version` and the target, focusing on `vcs.*`, `cicd.*`, and `deploy.*`. Note
   renames, additions, removals, and stability changes.
3. **Reconcile differences** — update attribute/metric names in `metadata.yaml`, adjust scraper
   logic, and check the [Extensions register](#extensions-register) for extensions the spec now
   covers.
4. **Update both version signals together** — set the new `sem_conv_version` *and* bump the
   hand-written `otel/semconv/vX` imports so they agree.
5. **Regenerate and commit** — `make gen`, then commit the regenerated `internal/metadata/` and
   `documentation.md`.
6. **Note the package version** each component targets after the change in the PR description.

When reviewing an OCB / collector-core Renovate PR, confirm it did not change any semconv import —
re-derive the [version table](#how-to-verify) before and after and diff it.

## References

- OpenTelemetry Semantic Conventions (spec): <https://opentelemetry.io/docs/specs/semconv/>
- VCS metrics: <https://opentelemetry.io/docs/specs/semconv/cicd/cicd-metrics/#vcs-metrics>
- semconv repo / changelog: <https://github.com/open-telemetry/semantic-conventions>
- Go packages: <https://pkg.go.dev/go.opentelemetry.io/otel/semconv>

[semconv]: https://opentelemetry.io/docs/specs/semconv/
[sc-repo]: https://github.com/open-telemetry/semantic-conventions
[sc-releases]: https://github.com/open-telemetry/semantic-conventions/releases
[sc-changelog]: https://github.com/open-telemetry/semantic-conventions/blob/main/CHANGELOG.md
[pkg]: https://pkg.go.dev/go.opentelemetry.io/otel/semconv
