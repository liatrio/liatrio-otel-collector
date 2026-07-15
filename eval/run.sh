#!/usr/bin/env bash
#
# run.sh — offline replay harness for the /fix skill (spec Unit 4, Tasks 3.3 + 4.4).
#
# For each named fixture (default: all fixtures under eval/fixtures/ with a repo.bundle), this:
#   1. clones the fixture's offline git bundle into a throwaway workdir (no network),
#   2. checks out the fixture's red-state branch and records its base ref,
#   3. drops the skill-under-test into the clone at .claude/skills/fix/ (git-excluded so it never
#      pollutes the tree the assertions inspect) and the offline failure-context sidecar,
#   4. invokes the skill headless with the spec's determinism flags, and
#   5. hands the resulting tree + recap + JSON output to eval/assertions.sh (deterministic hard gate)
#      and, when EVAL_JUDGE=1, to eval/judge.sh (sampled, non-gating LLM-as-judge).
#
# The headless invocation is the ONLY step that spends money / needs network. Pass --skip-agent
# (or EVAL_SKIP_AGENT=1) to skip it and run the deterministic assertions against the untouched
# clone — this is how the test-first RED baseline is demonstrated without a paid Claude run.
#
# A/B MODE (Task 4.4): --ab v1,v2 (or EVAL_AB=v1,v2) runs EACH fixture against TWO skill snapshots
# from eval/skills/<ver>/ and prints a per-fixture pass/fail delta, so a regression can be attributed
# to the wording change between the two versions. A/B implies two paid runs per fixture.
#
# Config (all env-overridable; see the flags below for the common ones):
#   EVAL_MAX_TURNS        headless --max-turns runaway ceiling            (default 60)
#   EVAL_MAX_BUDGET_USD   headless --max-budget-usd runaway ceiling       (default 3)
#   EVAL_ALLOWED_TOOLS    headless --allowedTools                         (default "Bash,Read,Edit")
#   EVAL_SKILL_VERSION    eval/skills/<ver>/SKILL.md to test; empty uses  (default "": repo skill)
#                         the repo's canonical .claude/skills/fix/SKILL.md
#   EVAL_AB               "verA,verB": A/B two skill snapshots per fixture (default "": single run)
#   EVAL_JUDGE=1          also run the sampled, non-gating LLM-as-judge (eval/judge.sh); spends money
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
#   ./eval/run.sh cve --ab fix-v1,fix-v2             # A/B: v1 vs v2 on the cve fixture (2 paid runs)
#   EVAL_JUDGE=1 ./eval/run.sh semconv-defer          # + sampled non-gating recap quality scores
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FIXTURES_DIR="$SCRIPT_DIR/fixtures"

MAX_TURNS="${EVAL_MAX_TURNS:-60}"
MAX_BUDGET_USD="${EVAL_MAX_BUDGET_USD:-3}"
ALLOWED_TOOLS="${EVAL_ALLOWED_TOOLS:-Bash,Read,Edit}"
SKILL_VERSION="${EVAL_SKILL_VERSION:-}"
AB="${EVAL_AB:-}"
JUDGE="${EVAL_JUDGE:-0}"
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
    --ab) AB="$2"; shift ;;
    --ab=*) AB="${1#*=}" ;;
    --judge) JUDGE=1 ;;
    --keep) KEEP=1 ;;
    -h|--help) sed -n '2,50p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
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

# --- resolve a skill version name to its SKILL.md source -------------------------------------------
skill_src_for() { # skill_src_for <version-or-empty> -> path (echoes)
  local ver="$1"
  if [[ -n "$ver" ]]; then echo "$SCRIPT_DIR/skills/$ver/SKILL.md"
  else echo "$REPO_ROOT/.claude/skills/fix/SKILL.md"; fi
}

# --- workdir ----------------------------------------------------------------------------------------
if [[ -n "${EVAL_WORKDIR:-}" ]]; then
  WORKROOT="$EVAL_WORKDIR"
  mkdir -p "$WORKROOT"
else
  WORKROOT="$(mktemp -d "${TMPDIR:-/tmp}/fix-eval.XXXXXX")"
fi
cleanup() { [[ "$KEEP" == "1" ]] || rm -rf "$WORKROOT"; }
trap cleanup EXIT

# ===================================================================================================
# run_one — replay one fixture against one skill source; returns 0=PASS 1=FAIL.
#   $1 fixture name   $2 skill source path   $3 skill label (for logs/workdir uniqueness)
# ===================================================================================================
run_one() {
  local name="$1" skill_src="$2" label="$3"
  local fdir="$FIXTURES_DIR/$name"
  local bundle="$fdir/repo.bundle" expected="$fdir/expected.yaml"
  if [[ ! -f "$bundle" || ! -f "$expected" ]]; then
    echo "    FIXTURE $name: MISSING (need repo.bundle + expected.yaml)"; return 1
  fi

  local work="$WORKROOT/$name-$label" repo
  repo="$work/repo"
  mkdir -p "$work"

  # 1. offline clone from the bundle
  git clone -q "$bundle" "$repo"

  # 2. check out the fixture branch (bundle HEAD points at it) and record the base
  local branch base
  branch="$(awk -F': *' '/^branch:/{gsub(/["\r]/,"",$2); print $2; exit}' "$expected")"
  base="$(awk -F': *' '/^base:/{gsub(/["\r]/,"",$2); print $2; exit}' "$expected")"
  if [[ -n "$branch" ]]; then
    git -C "$repo" checkout -q "$branch" 2>/dev/null || git -C "$repo" checkout -q -b "$branch" 2>/dev/null || true
  fi

  # 3a. install the skill-under-test, git-excluded so it never enters the inspected tree
  if [[ -f "$skill_src" ]]; then
    mkdir -p "$repo/.claude/skills/fix"
    cp "$skill_src" "$repo/.claude/skills/fix/SKILL.md"
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

  local out_json="$work/out.json" recap="$work/recap.md"
  : > "$recap"
  echo '{"skipped":true}' > "$out_json"

  # 4. headless skill invocation (the only paid / networked step)
  if [[ "$SKIP_AGENT" == "1" ]]; then
    echo "    [skip-agent] not invoking claude; asserting against the untouched clone (RED baseline)"
  else
    if [[ ! -f "$skill_src" ]]; then
      echo "    skill not found at $skill_src — cannot run live; use --skip-agent for the RED baseline" >&2
      return 1
    fi
    # Under --bare, project .claude/skills/ is NOT discovered ("/fix" -> "Unknown command"): --bare
    # strips plugin sync and skills only resolve via /name from synced/user config. Per `claude
    # --help`, the way to provide context under --bare is explicit injection. So we append the
    # versioned SKILL.md body (frontmatter stripped) as a system prompt and give a minimal user
    # prompt. This is deterministic under --bare AND is exactly what A/B-testing the skill *text*
    # means: fix-v1 vs fix-v2 differ only in this injected text.
    local sp="$work/skill-system-prompt.md" skill_body userprompt
    awk 'BEGIN{n=0} /^---[[:space:]]*$/{n++; if(n<=2) next} n>=2{print}' "$skill_src" > "$sp"
    skill_body="$(cat "$sp")"
    userprompt="You are running as the \`/fix\` skill defined in your system prompt. The Renovate PR branch \`$branch\` is already checked out in the current working directory (repo root; paths in your instructions are relative to it). A failure-context sidecar is at \$FIX_CONTEXT_FILE. Fix the PR per your instructions (local effects only — never push or comment). Your FINAL message MUST be the structured recap exactly as specified: the YAML front-matter block followed by the fixed sections."
    local det_flags
    if [[ "$BARE" == "1" ]]; then
      det_flags=(--bare)                # production determinism; requires ANTHROPIC_API_KEY
    else
      det_flags=(--strict-mcp-config)   # ambient OAuth/plan login; strip MCP for determinism
    fi
    echo "    invoking: claude ${det_flags[*]} (skill='$label' via --append-system-prompt) on \`$branch\` (max-turns=$MAX_TURNS, max-budget-usd=$MAX_BUDGET_USD)"
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

  # 5a. deterministic assertions (hard gate)
  local rc=0
  if EVAL_SKIP_AGENT="$SKIP_AGENT" bash "$SCRIPT_DIR/assertions.sh" \
        --repo "$repo" --expected "$expected" --recap "$recap" --json "$out_json" --base "$base"; then
    rc=0
  else
    rc=1
  fi

  # 5b. sampled, non-gating LLM-as-judge (never affects rc)
  if [[ "$JUDGE" == "1" && "$SKIP_AGENT" != "1" ]]; then
    bash "$SCRIPT_DIR/judge.sh" --recap "$recap" --expected "$expected" \
      --name "$name/$label" --out "$work/judge.json" || true
  fi

  return "$rc"
}

# ===================================================================================================
# main
# ===================================================================================================
if [[ -n "$AB" ]]; then
  echo "eval harness (A/B) — versions: $AB  skip-agent: $SKIP_AGENT  judge: $JUDGE"
else
  echo "eval harness — skill: ${SKILL_VERSION:-<repo .claude/skills/fix>}  skip-agent: $SKIP_AGENT  judge: $JUDGE"
fi
echo "workdir: $WORKROOT"
echo "==================================================================================="

overall=0
if [[ -n "$AB" ]]; then
  IFS=',' read -r VER_A VER_B <<< "$AB"
  [[ -n "$VER_A" && -n "$VER_B" ]] || { echo "run.sh: --ab needs two comma-separated versions, got '$AB'" >&2; exit 2; }
  SRC_A="$(skill_src_for "$VER_A")"; SRC_B="$(skill_src_for "$VER_B")"
  for name in "${FIXTURES[@]}"; do
    echo ">>> fixture: $name   (A/B: $VER_A vs $VER_B)"
    ra=0; rb=0
    echo "  --- $VER_A ---"; run_one "$name" "$SRC_A" "$VER_A" || ra=1
    echo "  --- $VER_B ---"; run_one "$name" "$SRC_B" "$VER_B" || rb=1
    a_txt=$([[ $ra -eq 0 ]] && echo PASS || echo FAIL)
    b_txt=$([[ $rb -eq 0 ]] && echo PASS || echo FAIL)
    if [[ $ra -ne $rb ]]; then
      echo "    A/B DELTA: $VER_A=$a_txt  $VER_B=$b_txt   <-- versions DIVERGE (regression signal)"
    else
      echo "    A/B SAME:  $VER_A=$a_txt  $VER_B=$b_txt"
    fi
    echo "-----------------------------------------------------------------------------------"
  done
  # A/B mode is a comparison report, not a gate: it always exits 0 so a divergence is a signal, not a
  # CI failure (make eval never runs A/B anyway — this is a human-driven iteration tool).
  echo "==================================================================================="
  echo "EVAL (A/B): report complete"
  exit 0
fi

SKILL_SRC="$(skill_src_for "$SKILL_VERSION")"
for name in "${FIXTURES[@]}"; do
  echo ">>> fixture: $name"
  if run_one "$name" "$SKILL_SRC" "${SKILL_VERSION:-repo}"; then
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
