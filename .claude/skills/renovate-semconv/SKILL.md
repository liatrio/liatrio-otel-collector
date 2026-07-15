---
name: renovate-semconv
description: >-
  Renovate maintenance playbook, read by the `/fix` skill, covering OpenTelemetry
  semantic-convention breakage. Select when a dependency bump surfaces semconv failure signatures in
  the failing logs — `undefined: semconv.`, a `sem_conv_version` mismatch, `semconv/v1`, or
  `has no field or method` on a semconv symbol — or when a package under
  `go.opentelemetry.io/otel/semconv/**` changes directly. A semconv migration is a deliberate manual,
  judgment-tier change: **never bump the semconv package** as a side effect; it terminates at best
  `partial`/`deferred`, never a confident `fixed`. Frequently overlaid on top of `renovate-otel-core`
  when a collector-core bump pulls a semconv change into generated code.
---

# Playbook: semantic conventions (semconv)

Semconv is **not** a Renovate group — Renovate deliberately does not bump the semconv package (it is
a manual migration). This playbook is therefore selected by **failure signature** during a `/fix`
run, not by a group label: a collector-core bump ([`renovate-otel-core`](../renovate-otel-core/SKILL.md))
can surface semconv breakage in generated code, and the compiler errors then match the failure
signatures in this skill's `description`. A direct `go.opentelemetry.io/otel/semconv/**` reference is
a secondary hook for the rare case Renovate touches the package.

**This is a judgment-tier class, not a mechanical one.** A semconv migration is best-effort at most;
do not report it as a completed fix. See spec Unit 3 for the three-tier classification.

## Authoritative docs

- [`docs/semantic-conventions.md`](../../../docs/semantic-conventions.md) — the full policy:
  spec-vs-package version axes, the version matrix, the extensions register, and the
  [package upgrade / OCB / Renovate runbook](../../../docs/semantic-conventions.md#package-upgrade--ocb--renovate-runbook).
- [`docs/semantic-conventions.md` → Rule of thumb](../../../docs/semantic-conventions.md#rule-of-thumb-when-in-doubt-default-to-the-spec)
  — default to the spec at the version our packages declare, not whatever is newest upstream.
- [`AGENTS.md` → Semantic conventions](../../../AGENTS.md#semantic-conventions) — the short-form
  posture and the "do not bump the semconv package as a side effect" guardrail.

## Failure modes → remediation

| Failure signature | Likely cause | Remediation |
| --- | --- | --- |
| `undefined: semconv.<Const>` after a core bump | an attribute/const was renamed or removed between semconv package versions | look up the new name in the declared spec version; update the reference to the spec name (an extension attribute is replaced by the spec name when the spec defines one) |
| `sem_conv_version` mismatch in `metadata.yaml` vs imported package | the declared spec version and the Go package version diverged | reconcile per the version matrix; **do not** silently bump the package to satisfy the code |
| generated code references a semconv version the package does not export | mdatagen tried to stamp an import that does not exist (package lags the spec) | the package version is the hard ceiling — stay at or below it; this is why the bump may only be *best-effort* |
| behavioral / attribute-meaning change | a spec revision changed semantics, not just names | **defer** — this is not mechanical; record it in the recap's "What's left" |

## Hard invariants / guardrails

- **Never bump the `otel/semconv/vX` package as a side effect of another bump.** It is a deliberate,
  manual, reviewed migration — the single most important invariant here.
- **The package version is the hard ceiling** — mdatagen cannot stamp an import that does not exist,
  so never target a spec version newer than the package exports.
- **Never hand-edit generated files** — change `metadata.yaml`'s `sem_conv_version` (or the
  hand-written source) and run `make gen`.
- When the spec defines a convention, the spec name **replaces** the local extension attribute;
  update the extensions register in [`docs/semantic-conventions.md`](../../../docs/semantic-conventions.md#extensions-register) if you change one.
- A semconv migration terminates at best **`partial`**/**`deferred`**, never a confident `fixed`
  unless it is a pure rename fully verified by `make generate` + `make test-all`.
