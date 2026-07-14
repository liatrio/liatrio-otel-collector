#!/usr/bin/env bash
#
# assertions.sh — deterministic (hard-gate) assertions for a single /fix eval run (spec Unit 4,
# Task 3.4). Invoked by run.sh once per fixture against the post-run clone. All checks here are
# deterministic and gating; the sampled, non-gating LLM-as-judge lives separately (eval/judge.*,
# Task 4.3).
#
# Args:
#   --repo DIR        the replayed fixture clone (post-run tree, HEAD on the fixture branch)
#   --expected FILE   the fixture's expected.yaml sidecar
#   --recap FILE      the skill's structured recap (JSON "result" field extracted by run.sh)
#   --json FILE       the raw `claude --output-format json` payload (for is_error)
#   --base SHA        the branch's base ref (diff origin)
#
# Env: EVAL_SKIP_AGENT=1 marks a RED baseline — no fix was attempted, so post-fix-only checks
# (generate diff-clean) are moot and skipped; the run still (correctly) fails on the missing recap.
#
# Checks: terminal state ∈ {fixed,partial,deferred,exhausted} and == expected; is_error == false;
# .github/** untouched; must_change / must_not_change path globs; commit count in range;
# Conventional Commit subjects; required trailers present; post-run `make generate` diff-clean.
#
set -uo pipefail

REPO="" EXPECTED="" RECAP="" JSON="" BASE=""
while (($#)); do
  case "$1" in
    --repo) REPO="$2"; shift ;;
    --expected) EXPECTED="$2"; shift ;;
    --recap) RECAP="$2"; shift ;;
    --json) JSON="$2"; shift ;;
    --base) BASE="$2"; shift ;;
    *) echo "assertions.sh: unknown arg: $1" >&2; exit 2 ;;
  esac
  shift
done
SKIP_AGENT="${EVAL_SKIP_AGENT:-0}"

fail=0
check() { # check "desc" PASS|FAIL "detail"
  if [[ "$2" == "PASS" ]]; then printf '    PASS  %-42s %s\n' "$1" "${3:-}"
  else printf '    FAIL  %-42s %s\n' "$1" "${3:-}"; fail=1; fi
}
skip() { printf '    SKIP  %-42s %s\n' "$1" "${2:-}"; }

# --- expected.yaml readers (flat scalars + simple block lists) -------------------------------------
y_scalar() { awk -F': *' -v k="$1" '$0 ~ "^"k":" {v=$2; gsub(/["\r]/,"",v); print v; exit}' "$EXPECTED"; }
y_list() {   # emit items of a top-level block list `key:`\n  - item
  awk -v k="$1" '
    $0 ~ "^"k":" {inlist=1; next}
    inlist && /^[[:space:]]*-[[:space:]]*/ {line=$0; sub(/^[[:space:]]*-[[:space:]]*/,"",line); gsub(/["\r]/,"",line); print line; next}
    inlist && /^[^[:space:]#]/ {inlist=0}
  ' "$EXPECTED"
}

# glob → path match: ** and * both span path segments (bash [[ ]] * already crosses '/').
path_matches() { local pat="${1%/\*\*}"; pat="${pat//\*\*/\*}"; local p="$2"
  [[ "$p" == "$pat" || "$p" == "$pat"/* || "$p" == $pat ]]; }

want_state="$(y_scalar terminal_state)"

# --- changed-path set (committed since base + working-tree) ----------------------------------------
changed=""
if [[ -n "$BASE" ]] && git -C "$REPO" cat-file -e "$BASE^{commit}" 2>/dev/null; then
  changed="$(git -C "$REPO" diff --name-only "$BASE" HEAD 2>/dev/null)"
fi
# untracked/uncommitted too (skill should have committed, but catch stragglers)
wt="$(git -C "$REPO" status --porcelain=v1 --untracked-files=all 2>/dev/null | awk '{print $2}')"
changed="$(printf '%s\n%s\n' "$changed" "$wt" | sed '/^$/d' | sort -u)"

# ============================ CHECK 1: terminal state ==============================================
# Extract the recap's `status:` robustly: the first line whose first token is `status:` (tolerating
# preamble prose and a ```markdown code fence some models wrap the recap in). The strict
# between-the-first-two-`---`-fences form is too brittle in practice.
fm_status="$(awk 'tolower($1)=="status:"{v=$2; gsub(/["\r,]/,"",v); print tolower(v); exit}' "$RECAP" 2>/dev/null)"
if [[ -z "$fm_status" ]]; then
  check "terminal state present" FAIL "no parseable 'status:' in recap front-matter (RED: no fix produced)"
else
  case "$fm_status" in
    fixed|partial|deferred|exhausted) valid=1 ;; *) valid=0 ;;
  esac
  if ((valid)); then check "terminal state is a valid enum" PASS "status=$fm_status"
  else check "terminal state is a valid enum" FAIL "status=$fm_status ∉ {fixed,partial,deferred,exhausted}"; fi
  if [[ -n "$want_state" ]]; then
    [[ "$fm_status" == "$want_state" ]] && check "terminal state matches expected" PASS "$fm_status" \
      || check "terminal state matches expected" FAIL "got=$fm_status want=$want_state"
  fi
fi

# ============================ CHECK 2: is_error ====================================================
if [[ "$SKIP_AGENT" == "1" ]]; then
  skip "is_error == false" "(skip-agent: no headless run)"
elif command -v python3 >/dev/null 2>&1 && [[ -f "$JSON" ]]; then
  is_err="$(python3 -c 'import json,sys
try: print(str(json.load(open(sys.argv[1])).get("is_error", "missing")).lower())
except Exception: print("parse_error")' "$JSON")"
  [[ "$is_err" == "false" ]] && check "is_error == false" PASS \
    || check "is_error == false" FAIL "is_error=$is_err"
else
  skip "is_error == false" "(python3 or json output unavailable)"
fi

# ============================ CHECK 3: .github/** untouched ========================================
gh_touch="$(printf '%s\n' "$changed" | while read -r p; do [[ -n "$p" ]] && path_matches ".github/**" "$p" && echo "$p"; done)"
[[ -z "$gh_touch" ]] && check ".github/** untouched" PASS \
  || check ".github/** untouched" FAIL "touched: $(echo "$gh_touch" | tr '\n' ' ')"

# ============================ CHECK 4: must_change / must_not_change ===============================
if [[ "$SKIP_AGENT" == "1" ]]; then
  skip "must_change globs satisfied" "(skip-agent: no fix attempted)"
else
  while IFS= read -r glob; do
    [[ -z "$glob" ]] && continue
    hit=""; while IFS= read -r p; do [[ -n "$p" ]] && path_matches "$glob" "$p" && { hit="$p"; break; }; done <<< "$changed"
    [[ -n "$hit" ]] && check "must_change: $glob" PASS "e.g. $hit" \
      || check "must_change: $glob" FAIL "no changed path matched"
  done < <(y_list must_change)
fi
while IFS= read -r glob; do
  [[ -z "$glob" ]] && continue
  bad=""; while IFS= read -r p; do [[ -n "$p" ]] && path_matches "$glob" "$p" && { bad="$p"; break; }; done <<< "$changed"
  [[ -z "$bad" ]] && check "must_not_change: $glob" PASS \
    || check "must_not_change: $glob" FAIL "violated by $bad"
done < <(y_list must_not_change)

# ============================ CHECK 5: commit structure ============================================
if [[ "$SKIP_AGENT" == "1" ]]; then
  skip "commit count in range" "(skip-agent: no fix commits)"
elif [[ -n "$BASE" ]] && git -C "$REPO" cat-file -e "$BASE^{commit}" 2>/dev/null; then
  n="$(git -C "$REPO" rev-list --count "$BASE"..HEAD 2>/dev/null || echo 0)"
  cmin="$(y_scalar commit_count_min)"; cmax="$(y_scalar commit_count_max)"
  ok=1
  [[ -n "$cmin" && "$n" -lt "$cmin" ]] && ok=0
  [[ -n "$cmax" && "$n" -gt "$cmax" ]] && ok=0
  ((ok)) && check "commit count in range" PASS "n=$n [${cmin:-0},${cmax:-∞}]" \
    || check "commit count in range" FAIL "n=$n [${cmin:-0},${cmax:-∞}]"

  # Conventional Commit subjects on every new commit.
  if [[ "$(y_scalar conventional_commits)" == "true" && "$n" -gt 0 ]]; then
    bad_subj=""
    while IFS= read -r subj; do
      [[ "$subj" =~ ^(feat|fix|chore|docs|refactor|test|build|ci|perf|style|revert)(\(.+\))?!?:\ .+ ]] || bad_subj="$subj"
    done < <(git -C "$REPO" log --format='%s' "$BASE"..HEAD)
    [[ -z "$bad_subj" ]] && check "conventional-commit subjects" PASS \
      || check "conventional-commit subjects" FAIL "non-conforming: $bad_subj"
  fi

  # Required trailers present somewhere in the new commits (e.g. "Status: incomplete").
  while IFS= read -r trailer; do
    [[ -z "$trailer" ]] && continue
    git -C "$REPO" log --format='%b' "$BASE"..HEAD | grep -qiF "$trailer" \
      && check "required trailer present" PASS "$trailer" \
      || check "required trailer present" FAIL "missing: $trailer"
  done < <(y_list required_trailers)
else
  check "commit structure checkable" FAIL "base ref $BASE not present in clone"
fi

# ============================ CHECK 6: post-run `make generate` diff-clean =========================
if [[ "$SKIP_AGENT" == "1" ]]; then
  skip "make generate diff-clean" "(skip-agent: pre-fix tree is expected dirty)"
elif [[ "$(y_scalar generate_diff_clean)" == "true" ]]; then
  before="$(git -C "$REPO" status --porcelain=v1 --untracked-files=all)"
  # The full-repo `make generate` needs the complete toolchain (mdatagen AND genqlient). Build it
  # here so this hard gate is self-sufficient regardless of which tools the skill happened to build.
  (cd "$REPO" && make install-tools >/dev/null 2>&1)
  if (cd "$REPO" && make generate >/dev/null 2>&1); then
    after="$(git -C "$REPO" status --porcelain=v1 --untracked-files=all)"
    [[ "$before" == "$after" ]] && check "make generate diff-clean" PASS \
      || check "make generate diff-clean" FAIL "regeneration produced a diff"
  else
    check "make generate diff-clean" FAIL "make generate errored in the clone"
  fi
else
  skip "make generate diff-clean" "(not asserted for this fixture)"
fi

exit "$fail"
