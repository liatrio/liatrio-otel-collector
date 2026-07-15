# CI: `regen` job and the `main` branch ruleset

The `regen` job (`.github/workflows/build.yml`) automatically commits mechanical fixes
(`make generate`/`tidy-all`/`crosslink`/`fmt-all`) back to a failing `renovate[bot]` PR branch. This
doc records why `regen` cannot reach `main`, and captures the admin-side ruleset state. No
`.github/workflows/build.yml` diff can show this boundary, because it is enforced by the octo-sts
trust policy's subject match, the `regen` job's script discipline, and the repository's branch
ruleset — not by workflow YAML.

## Summary of the boundary

> **`regen` cannot push to `main` in practice — but the enforcement is the trust-policy mint gate
> plus the job's script discipline, NOT the `main` ruleset.** The ruleset does *not* block `regen`,
> because octo-sts (the shared minting App) must remain on `main`'s bypass list for releases. This is
> a known, accepted residual (decision 2026-07-15); see "Defense layer 3" and "Accepted residual"
> below.

## Defense layer 1 (primary): the trust policy only federates from a renovate PR context

`.github/chainguard/regen.sts.yaml` restricts the `regen` identity's OIDC subject to
`repo:liatrio/liatrio-otel-collector:pull_request` with `claim_pattern.head_ref: "^renovate/.*"` and
`claim_pattern.actor_id: "29139614"` (renovate[bot]'s signed, unspoofable numeric id). A GitHub
Actions OIDC token only carries the `pull_request` subject when the workflow run itself was triggered
by a `pull_request` event. The `regen` job runs under `on: pull_request` in `build.yml` — it can
never run under `on: push` to `main` — so no run of this job can ever present a subject octo-sts would
federate against for a `main` context. A `regen`-scoped token cannot be minted at all outside a
renovate[bot] PR run, independent of anything configured in GitHub's UI.

## Defense layer 2 (script discipline): the job only ever pushes to the PR head branch

Once minted from a legitimate renovate PR, the octo-sts installation token carries repo-wide
`contents: write` (App tokens cannot be branch-scoped at the token layer). What keeps it off `main` is
the `regen` job itself: it checks out and pushes only to `$HEAD_REF` (the `renovate/**` branch), and
refuses any diff touching `.github/**` before pushing. Because `pull_request` runs the workflow
*definition* from the base branch, a poisoned PR head cannot rewrite the job to redirect the push.

## Defense layer 3 (platform backstop): the `main` ruleset — NOT effective for `regen`

**As captured (2026-07-15), the `main` ruleset does NOT block `regen`.** The ideal defense-in-depth
would be a `main` ruleset that excludes the `regen` identity from bypass, so that even a token minted
under an unexpectedly broad future policy change is blocked by GitHub itself. That is **not
achievable** here:

- GitHub ruleset bypass is granted **per GitHub App, not per octo-sts trust policy**. The bypass list
  entry is the octo-sts App; GitHub cannot distinguish a token minted by the `regen` policy from one
  minted by the `semantic-release` policy — both are "a token from the octo-sts App installation."
- `semantic-release` (`.github/chainguard/semantic-release.sts.yaml`, subject
  `…:ref:refs/heads/main`) mints through the **same octo-sts App** and pushes release-prep commits to
  `main` (the `multimod-prepare-release` job in `build.yml`). So octo-sts **must** be on `main`'s
  bypass list for releases to work.
- Therefore octo-sts being on the bypass list transitively grants `regen`-minted tokens bypass of the
  `main` ruleset. Removing octo-sts from the bypass list would break releases; scoping one identity's
  bypass independently is not supported by GitHub.

The originally-planned "ruleset excludes `regen` from bypass" control is thus **not in place**, and
`regen`'s inability to reach `main` rests on Defense layers 1 and 2.

## Captured ruleset state (2026-07-15, admin session)

Ruleset `main` (id `1433205`), enforcement **Active**, targeting the default branch. **Bypass list**
(all "Always allow"), from the admin settings page — see the screenshot proof
[`01-proofs/01-task-01-main-ruleset-bypass.png`](./specs/01-spec-renovate-maintenance-agent/01-proofs/01-task-01-main-ruleset-bypass.png):

| Bypass actor | Type | Note |
| --- | --- | --- |
| Repository admin | Role | Standard admin bypass. |
| **Octo STS** | App (`octo-sts`) | **Shared minting App — transitively covers the `regen` identity.** Required for `semantic-release` release pushes to `main`. |
| Liatrio OTEL Collector Release | App (`liatrio`) | Release automation App. |
| tag-o11y-chairs | Team | Maintainer group. |

## Accepted residual (decision 2026-07-15)

Per maintainer decision, the shared-App bypass limitation is an **accepted residual**, documented
rather than remediated in this spec:

- The **practical** exposure is low: to push to `main` a `regen` token must be *minted* (requires a
  genuine renovate[bot] PR run — Defense layer 1) **and** *used* against `main` (no code path does
  this — Defense layer 2). Both would have to fail together.
- The clean platform-level fix — migrating release `main`-pushes off octo-sts onto the dedicated
  "Liatrio OTEL Collector Release" App so octo-sts can be removed from `main`'s bypass — is a
  release-infrastructure change **beyond this spec's scope**. It is recorded here as the follow-on
  remediation if the residual is later deemed unacceptable.
- **Regression watch:** if a *new* octo-sts identity is created whose trust policy is broad enough to
  be minted outside a renovate PR context, Defense layer 1 weakens and this residual must be
  re-evaluated. Any change to `main`'s bypass list should be reviewed against this doc.

## Why this matters

`git diff` on `build.yml` cannot show branch-ruleset state — a reviewer reading the workflow diff has
no way to confirm how `regen` is (and is not) kept off `main`. This doc plus the captured screenshot
is the durable, checked-in record of that boundary and its one accepted limitation.
