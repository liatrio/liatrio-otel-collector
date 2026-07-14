# Renovate maintenance playbooks

Versioned, matchable knowledge that keeps the Tier-2 `/fix` agent inside repository conventions
instead of freelancing. Each playbook grounds a class of Renovate dependency PR; this README defines
how a PR is mapped to its playbook.

The selection contract's guiding principle: **key off Renovate's own grouping decision, do not
re-derive it.** Renovate already decided how to group these bumps (in
[`renovate.json`](../../renovate.json)); selection reads that decision back off the PR rather than
re-implementing the grouping logic and risking drift.

## The seed set

| Playbook | Selected by | Covers |
| --- | --- | --- |
| [`otel-core.md`](./otel-core.md) | `group:otel-core-contrib` label | the lockstep collector core + contrib + OTel API/SDK group |
| [`semconv.md`](./semconv.md) | failure signature (and a secondary `packages:` glob) | semconv migrations surfaced by a core bump |
| [`_default.md`](./_default.md) | fallback (no group, no glob match) | repository-wide conventions only |

Note that most Renovate group labels (`group:dockerfile`, `group:github-actions`, `group:tool-deps`,
`group:otel-compgen`) have **no dedicated playbook yet** — they resolve to `_default.md` by the
fallback rule below. That is intentional: a dedicated playbook is added only once a class of bump
proves it needs bespoke knowledge (see [Adding a playbook](#adding-a-playbook)).

## Selection contract

Resolve a PR to a playbook (or a set of playbooks) with these rules, in order:

1. **Group label first (authoritative).** Read the PR's Renovate group label
   (`gh pr view <pr> --json labels`). Strip the `group:` prefix and match it 1:1 against a playbook's
   `group:` front-matter field. This is authoritative because it directly consumes Renovate's own
   grouping decision — crucially, it reconstructs the two **file-path-based** groups (`tool deps` via
   `matchFileNames: internal/tools/**`, `otel-compgen cmd deps` via `cmd/otel-compgen/**`) that a
   package-**name** glob cannot reconstruct from the dependency list alone.
2. **Glob fallback for ungrouped single-package PRs.** When a PR carries no `group:*` label,
   enumerate the bumped packages from the structured `renovate-upgrades` JSON blob (emitted into the
   PR body as `<!-- renovate-upgrades:[...] -->` via the top-level `prBodyNotes` in
   [`renovate.json`](../../renovate.json)) and match each package name against playbook `packages:`
   globs. Parse the JSON from that HTML comment rather than scraping the human-readable dependency
   table.
3. **Most-specific-wins on multi-match.** If more than one playbook's globs match a package, choose
   the most specific: **exact name > scoped path glob > broad wildcard**. This mirrors
   `renovate.json`'s own last-rule-wins convention.
4. **Fall back to `_default.md`** when neither a group label nor any glob matches. When a **grouped**
   PR genuinely spans multiple playbooks, surface the *set* of matched playbooks — do not silently
   collapse to one.

Failure signatures (`failure_signatures:` front-matter) are a secondary, `/fix`-time hook: a bump
selected by group/glob can still surface a *different* class of breakage (a core bump that produces
semconv compiler errors), and matching the failing-log text pulls in the relevant playbook
([`semconv.md`](./semconv.md)) on top of the primary selection.

## Worked examples

| PR | Group label | Resolution | Rule |
| --- | --- | --- | --- |
| grouped collector/contrib bump | `group:otel-core-contrib` | [`otel-core.md`](./otel-core.md) | 1 (direct group match) |
| `tool deps` bump (grouped by file path `internal/tools/**`) | `group:tool-deps` | [`_default.md`](./_default.md) | 1 then 4 (label read, no dedicated playbook → fallback) |
| a single ungrouped `github.com/some/lib` bump | none | [`_default.md`](./_default.md) | 2 then 4 (no glob matches → fallback) |
| a core bump whose build fails with `undefined: semconv.Foo` | `group:otel-core-contrib` | [`otel-core.md`](./otel-core.md) **+** [`semconv.md`](./semconv.md) | 1 + failure-signature overlay |

The `tool deps` row is the important one: it is grouped by **file path**, so only the label — not a
package-name glob — can identify it. That is why rule 1 is authoritative and comes first.

## Playbook front-matter

```yaml
---
group: <renovate-group-label-suffix | null>   # e.g. otel-core-contrib; null if not a Renovate group
packages:                                      # globs for ungrouped single-package matching
  - "go.opentelemetry.io/otel/**"
failure_signatures:                            # substrings matched against failing logs at /fix time
  - "undefined: semconv."
---
```

- `group:` — the label suffix after `group:`. `_default` marks the fallback; `null` means the
  playbook is not tied to a Renovate group and is reached only by glob or failure signature.
- `packages:` — package-name globs; `*` matches within a path segment, `**` across segments.
- `failure_signatures:` — substrings; if any appears in the failing logs, overlay this playbook.

## Adding a playbook

Add one when a class of bump repeatedly needs bespoke, non-obvious handling that `_default.md`'s
repository-wide conventions don't cover:

1. Create `docs/playbooks/<name>.md` with the front-matter above and the three required body
   sections: **authoritative-doc pointers**, **failure-mode → remediation**, and **hard
   invariants/guardrails**. Link into authoritative docs; do not duplicate them.
2. If it maps to a Renovate group, ensure that group has an `addLabels: ["group:<suffix>"]` entry in
   [`renovate.json`](../../renovate.json) and set the playbook's `group:` to `<suffix>`.
3. Add a worked example to this README's table and a case to the selection-contract check
   ([`../../eval/select_playbook_test.sh`](../../eval/select_playbook_test.sh)) so the mapping stays
   deterministic and auditable.
