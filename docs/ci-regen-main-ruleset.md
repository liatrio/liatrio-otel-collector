# CI: `regen` job and the `main` branch ruleset

The `regen` job (`.github/workflows/build.yml`) automatically commits mechanical fixes
(`make generate`/`tidy-all`/`crosslink`/`fmt-all`) back to a failing `renovate[bot]` PR branch. This
doc records why `regen` cannot reach `main`, and the one admin-side check needed to confirm it: no
`.github/workflows/build.yml` diff can show this, because it is enforced by the octo-sts trust
policy's subject match plus the repository's branch ruleset, not by workflow YAML.

## Defense layer 1: the trust policy only federates from a PR context

`.github/chainguard/regen.sts.yaml` restricts the `regen` identity's OIDC subject to
`repo:liatrio/liatrio-otel-collector:pull_request` with `claim_pattern.head_ref: "^renovate/.*"`. A
GitHub Actions OIDC token only carries the `pull_request` subject when the workflow run itself was
triggered by a `pull_request` event. The `regen` job runs under `on: pull_request` in
`build.yml` — it can never run under `on: push` to `main` — so no run of this job can ever present a
subject octo-sts would federate against. This means a token scoped to the `regen` identity cannot be
minted at all outside a PR run, independent of anything configured in GitHub's UI.

## Defense layer 2: the `main` branch ruleset must not bypass `regen`

As defense-in-depth, the repository ruleset on `main` (`Settings → Rules → Rulesets → main`) must
never list the `regen` octo-sts identity/app among its bypass actors, so that even a token minted
under an unexpectedly broad future policy change is still blocked from writing to `main` directly by
GitHub itself.

**This is an admin-only setting** — reading or editing `bypass_actors` on a ruleset requires
repository admin permissions. The identity used to prepare this document/PR only has `push` access
(`admin: false`, confirmed via `gh api repos/liatrio/liatrio-otel-collector --jq .permissions`), so it
cannot read or change the bypass list. **A repository admin must perform and record the following
manually.**

### Steps for a repository admin

1. Open `https://github.com/liatrio/liatrio-otel-collector/settings/rules/1433205` (ruleset `main`,
   id `1433205`).
2. Confirm the ruleset still targets the default branch (`~DEFAULT_BRANCH`) and enforcement is
   `active`.
3. Under **Bypass list**, confirm the `regen` octo-sts identity does **not** appear. At the time this
   doc was written, `GET /repos/liatrio/liatrio-otel-collector/rulesets/1433205` returned no
   `bypass_actors` key at all when queried with a non-admin, `push`-scoped token — an admin should
   re-verify this with an admin-scoped token/session, since the field is omitted (not necessarily
   empty) for callers without admin access.
4. Capture a sanitized export of the ruleset (e.g. `gh api repos/liatrio/liatrio-otel-collector/rulesets/1433205`
   run with an admin token, or a screenshot of the Bypass list section of the settings page) with any
   org-internal usernames/emails redacted, and paste it into the
   [`01-proofs`](./specs/01-spec-renovate-maintenance-agent/01-proofs/) artifact for Task 1.0 as the
   "never main" proof artifact.
5. If `regen` (or any octo-sts identity broad enough to match `regen`'s trust policy) is ever added to
   the bypass list for another reason, treat that as a regression of this control and re-run this
   check.

## Why this matters

`git diff` on `build.yml` cannot show branch-ruleset state — a reviewer reading the workflow diff has
no way to confirm `regen` can't write to `main`. This doc, plus the sanitized capture referenced in
step 4, is the durable, checked-in record of that boundary.
