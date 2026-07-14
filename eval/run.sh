#!/usr/bin/env bash
#
# run.sh — offline replay harness for the /fix skill (spec Unit 4, Task 3.3).
#
# For each named fixture (default: all fixtures under eval/fixtures/ with a repo.bundle), this:
#   1. clones the fixture's offline git bundle into a throwaway workdir (no network),
#   2. checks out the fixture's red-state branch and records its base ref,
#   3. drops the skill-under-test into the clone at .claude/skills/fix/ (git-excluded so it never
#      pollutes the tree the assertions inspect) and the offline failure-context sidecar,
#   4. invokes the skill headless with the spec's determinism flags, and
#   5. hands the resulting tree + recap + JSON output to eval/assertions.sh.
#
# The headless invocation is the ONLY step that spends money / needs network. Pass --skip-agent
# (or EVAL_SKIP_AGENT=1) to skip it and run the deterministic assertions against the untouched
# clone — this is how the test-first RED baseline is demonstrated without a paid Claude run.
#
# Config (all env-overridable; see the flags below for the common ones):
#   EVAL_MAX_TURNS        headless --max-turns runaway ceiling            (default 60)
#   EVAL_MAX_BUDGET_USD   headless --max-budget-usd runaway ceiling       (default 3)
#   EVAL_ALLOWED_TOOLS    headless --allowedTools                         (default "Bash,Read,Edit")
#   EVAL_SKILL_VERSION    eval/skills/<ver>/SKILL.md to test; empty uses  (default "": repo skill)
#                         the repo's canonical .claude/skills/fix/SKILL.md
#   EVAL_SKIP_AGENT=1     skip the paid headless run (RED baseline / dry-run)
#   EVAL_BARE=1           use `--bare` (production determinism). REQUIRES ANTHROPIC_API_KEY: --bare
#                         never reads the OAuth login / keychain. Default 0 = run on the ambient
#                         OAuth/plan login (no --bare) with --strict-mcp-config to strip ambient MCP;
#                         the versioned skill text is injected via --append-system-prompt either way.
#   EVAL_WORKDIR          base scratch dir for clones                     (default: mktemp under TMPDIR)
#   EVAL_KEEP=1           keep workdirs after the run (for debugging)
#
# Usage:
#   ./eval/run.sh                       # every fixture, live (spends money)
#   ./eval/run.sh backoff               # one fixture, live
#   ./eval/run.sh backoff --skip-agent  # RED baseline: assertions on the untouched clone, no spend
#   EVAL_SKILL_VERSION=fix-v1 ./eval/run.sh backoff   # A/B: pin a versioned skill snapshot
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FIXTURES_DIR="$SCRIPT_DIR/fixtures"

MAX_TURNS="${EVAL_MAX_TURNS:-60}"
MAX_BUDGET_USD="${EVAL_MAX_BUDGET_USD:-3}"
ALLOWED_TOOLS="${EVAL_ALLOWED_TOOLS:-Bash,Read,Edit}"
SKILL_VERSION="${EVAL_SKILL_VERSION:-}"
SKIP_AGENT="${EVAL_SKIP_AGENT:-0}"
BARE="${EVAL_BARE:-0}"
KEEP="${EVAL_KEEP:-0}"

# --- arg parsing: positional fixture names + a few flags --------------------------------------------
FIXTURES=()
while (($#)); do
  case "$1" in
    --skip-agent) SKIP_AGENT=1 ;;
    --skill-version) SKILL_VERSION="$2"; shift ;;
    --skill-version=*) SKILL_VERSION="${1#*=}" ;;
    --keep) KEEP=1 ;;
    -h|--help) sed -n '2,40p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    -*) echo "run.sh: unknown flag: $1" >&2; exit 2 ;;
    *) FIXTURES+=("$1") ;;
  esac
  shift
done

# Default to every fixture directory that ships a bundle.
if ((${#FIXTURES[@]} == 0)); then
  for d in "$FIXTURES_DIR"/*/; do
    [[ -f "$d/repo.bundle" ]] && FIXTURES+=("$(basename "$d")")
  done
fi
if ((${#FIXTURES[@]} == 0)); then
  echo "run.sh: no fixtures found under $FIXTURES_DIR (need <name>/repo.bundle)" >&2
  exit 1
fi

# --- resolve the skill-under-test -------------------------------------------------------------------
if [[ -n "$SKILL_VERSION" ]]; then
  SKILL_SRC="$SCRIPT_DIR/skills/$SKILL_VERSION/SKILL.md"
else
  SKILL_SRC="$REPO_ROOT/.claude/skills/fix/SKILL.md"
fi

# --- workdir ----------------------------------------------------------------------------------------
if [[ -n "${EVAL_WORKDIR:-}" ]]; then
  WORKROOT="$EVAL_WORKDIR"
  mkdir -p "$WORKROOT"
else
  WORKROOT="$(mktemp -d "${TMPDIR:-/tmp}/fix-eval.XXXXXX")"
fi
cleanup() { [[ "$KEEP" == "1" ]] || rm -rf "$WORKROOT"; }
trap cleanup EXIT

echo "eval harness — skill: ${SKILL_VERSION:-<repo .claude/skills/fix>}  skip-agent: $SKIP_AGENT"
echo "workdir: $WORKROOT"
echo "==================================================================================="

overall=0
for name in "${FIXTURES[@]}"; do
  fdir="$FIXTURES_DIR/$name"
  bundle="$fdir/repo.bundle"
  expected="$fdir/expected.yaml"
  if [[ ! -f "$bundle" || ! -f "$expected" ]]; then
    echo "FIXTURE $name: MISSING (need repo.bundle + expected.yaml)"; overall=1; continue
  fi

  echo ">>> fixture: $name"
  work="$WORKROOT/$name"
  repo="$work/repo"
  mkdir -p "$work"

  # 1. offline clone from the bundle
  git clone -q "$bundle" "$repo"

  # 2. check out the fixture branch (bundle HEAD points at it) and record the base
  branch="$(awk -F': *' '/^branch:/{gsub(/["\r]/,"",$2); print $2; exit}' "$expected")"
  base="$(awk -F': *' '/^base:/{gsub(/["\r]/,"",$2); print $2; exit}' "$expected")"
  if [[ -n "$branch" ]]; then
    git -C "$repo" checkout -q "$branch" 2>/dev/null || git -C "$repo" checkout -q -b "$branch" 2>/dev/null || true
  fi

  # 3a. install the skill-under-test, git-excluded so it never enters the inspected tree
  if [[ -f "$SKILL_SRC" ]]; then
    mkdir -p "$repo/.claude/skills/fix"
    cp "$SKILL_SRC" "$repo/.claude/skills/fix/SKILL.md"
    grep -qxF '.claude/' "$repo/.git/info/exclude" 2>/dev/null || echo '.claude/' >> "$repo/.git/info/exclude"
  fi
  # 3b. offline failure-context sidecar (group label + renovate-upgrades JSON), also git-excluded
  if [[ -f "$fdir/context.json" ]]; then
    cp "$fdir/context.json" "$repo/.eval-context.json"
    grep -qxF '.eval-context.json' "$repo/.git/info/exclude" 2>/dev/null || \
      echo '.eval-context.json' >> "$repo/.git/info/exclude"
    export FIX_CONTEXT_FILE="$repo/.eval-context.json"
  else
    unset FIX_CONTEXT_FILE || true
  fi

  out_json="$work/out.json"
  recap="$work/recap.md"
  : > "$recap"
  echo '{"skipped":true}' > "$out_json"

  # 4. headless skill invocation (the only paid / networked step)
  if [[ "$SKIP_AGENT" == "1" ]]; then
    echo "    [skip-agent] not invoking claude; asserting against the untouched clone (RED baseline)"
  else
    if [[ ! -f "$SKILL_SRC" ]]; then
      echo "    skill not found at $SKILL_SRC — cannot run live; use --skip-agent for the RED baseline" >&2
      overall=1; continue
    fi
    # Under --bare, project .claude/skills/ is NOT discovered ("/fix" -> "Unknown command"): --bare
    # strips plugin sync and skills only resolve via /name from synced/user config. Per `claude
    # --help`, the way to provide context under --bare is explicit injection. So we append the
    # versioned SKILL.md body (frontmatter stripped) as a system prompt and give a minimal user
    # prompt. This is deterministic under --bare AND is exactly what A/B-testing the skill *text*
    # means: fix-v1 vs fix-v2 differ only in this injected text.
    sp="$work/skill-system-prompt.md"
    awk 'BEGIN{n=0} /^---[[:space:]]*$/{n++; if(n<=2) next} n>=2{print}' "$SKILL_SRC" > "$sp"
    skill_body="$(cat "$sp")"
    userprompt="You are running as the \`/fix\` skill defined in your system prompt. The Renovate PR branch \`$branch\` is already checked out in the current working directory (repo root; paths in your instructions are relative to it). A failure-context sidecar is at \$FIX_CONTEXT_FILE. Fix the PR per your instructions (local effects only — never push or comment). Your FINAL message MUST be the structured recap exactly as specified: the YAML front-matter block followed by the fixed sections."
    if [[ "$BARE" == "1" ]]; then
      det_flags=(--bare)                # production determinism; requires ANTHROPIC_API_KEY
    else
      det_flags=(--strict-mcp-config)   # ambient OAuth/plan login; strip MCP for determinism
    fi
    echo "    invoking: claude ${det_flags[*]} (skill via --append-system-prompt) on \`$branch\` (max-turns=$MAX_TURNS, max-budget-usd=$MAX_BUDGET_USD)"
    (
      cd "$repo"
      claude "${det_flags[@]}" -p "$userprompt" \
        --append-system-prompt "$skill_body" \
        --output-format json \
        --allowedTools "$ALLOWED_TOOLS" \
        --permission-mode acceptEdits \
        --max-turns "$MAX_TURNS" \
        --max-budget-usd "$MAX_BUDGET_USD"
    ) > "$out_json" 2> "$work/claude.stderr" || echo "    (claude exited non-zero; assertions will judge the result)"
    # The skill's structured recap is the JSON "result" field; fall back to raw stdout.
    if command -v python3 >/dev/null 2>&1; then
      python3 -c 'import json,sys
try:
    d=json.load(open(sys.argv[1]))
    sys.stdout.write(d.get("result") or d.get("text") or "")
except Exception:
    pass' "$out_json" > "$recap" 2>/dev/null || true
    fi
    [[ -s "$recap" ]] || cp "$out_json" "$recap"
  fi

  # 5. deterministic assertions
  if EVAL_SKIP_AGENT="$SKIP_AGENT" bash "$SCRIPT_DIR/assertions.sh" \
        --repo "$repo" --expected "$expected" --recap "$recap" --json "$out_json" --base "$base"; then
    echo "    RESULT: PASS"
  else
    echo "    RESULT: FAIL"
    overall=1
  fi
  echo "-----------------------------------------------------------------------------------"
done

echo "==================================================================================="
if ((overall)); then
  echo "EVAL: FAIL"
  exit 1
fi
echo "EVAL: PASS"
