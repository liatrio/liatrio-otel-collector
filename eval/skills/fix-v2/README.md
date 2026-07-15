# `fix-v2` — versioned skill snapshot slot (A/B challenger)

This directory holds the **frozen snapshot** of `.claude/skills/fix/SKILL.md` at the Task 4.0 green
milestone — the three-tier version. `eval/run.sh --skill-version fix-v2` (or `--ab fix-v1,fix-v2`)
copies `SKILL.md` from here into each fixture clone, so a pass/fail delta between v1 and v2 is
attributable to the wording change.

**What changed v1 → v2** (spec Unit 3/4, Task 4.0):

- **Three-tier change classification** — Tier A mechanical/determinable API adaptation (fix
  completely), Tier B semconv migration (best-effort first pass, never bump the semconv package;
  `Status: incomplete` trailer on any partial commit; else defer), Tier C behavioural redesign
  (defer). v1 fixed Tier A and blanket-deferred semconv.
- **Report-only osv-scanner CVE-delta gate** (Step 5b) — the final gate now scans base-vs-PR, diffs
  findings by vuln ID, and populates the recap's `cve_introduced` field for any new vulnerability.
  It is **report-only / non-gating**: osv-scanner v2.4.0 does not populate Go call-graph reachability
  (`experimentalAnalysis.called`), so a new CVE is surfaced prominently but does not by itself change
  the terminal state. v1 had no CVE-delta gate (`cve_introduced` was always `null`).

**A/B signal:** on the `cve` fixture, v1 emits `cve_introduced: null` and fails the
`cve_introduced_expected` assertion; v2 names the CVE and passes — a pass/fail delta attributable
solely to adding the CVE-delta gate. On the `semconv-defer` fixture both versions correctly `defer`
(the stable negative-case guard).
