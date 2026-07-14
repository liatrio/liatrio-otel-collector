#!/usr/bin/env bash
#
# select_playbook_test.sh — deterministic selection-contract check (spec Unit 2, Task 2.6).
#
# Implements the playbook selection contract from docs/playbooks/README.md against the *actual*
# front-matter of docs/playbooks/*.md (nothing hard-coded about which file wins), then asserts the
# documented worked examples resolve as specified. This is the Unit-2 test artifact: it makes the
# group-first / glob-fallback / most-specific-wins mapping auditable and regression-proof.
#
# Standalone and dependency-free (bash + awk only) so it needs neither the eval harness (Task 3.0)
# nor network access. Run directly:  bash eval/select_playbook_test.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PLAYBOOK_DIR="$REPO_ROOT/docs/playbooks"

# --- front-matter parsing -------------------------------------------------------------------------

# Emit the YAML front-matter block (between the first two '---' fences) of a file.
fm_block() {
  awk 'BEGIN{n=0} /^---[[:space:]]*$/{n++; next} n==1{print} n>=2{exit}' "$1"
}

# The scalar value of `group:` (trimmed, unquoted). Empty string when the value is `null`.
fm_group() {
  fm_block "$1" | awk -F: '/^group:/{sub(/^group:[[:space:]]*/,""); gsub(/["\r]/,""); if($0=="null")$0=""; print; exit}'
}

# The `packages:` list items (one glob per line, unquoted). Empty when `packages: []`.
fm_packages() {
  fm_block "$1" | awk '
    /^packages:/{inlist=1; next}
    inlist && /^[[:space:]]*-[[:space:]]*/{ line=$0; sub(/^[[:space:]]*-[[:space:]]*/,"",line); gsub(/["\r]/,"",line); print line; next }
    inlist && /^[^[:space:]]/{inlist=0}
  '
}

# --- glob matching + specificity -----------------------------------------------------------------

# glob_matches PATTERN PKG — treat ** and * both as "match anything" (bash [[ ]] has no globstar,
# and unquoted * already spans '/'), which is sufficient because specificity is scored separately.
glob_matches() {
  local pat="${1//\*\*/*}" pkg="$2"
  # shellcheck disable=SC2053
  [[ "$pkg" == $pat ]]
}

# specificity PATTERN — length of the literal prefix before the first wildcard; exact names (no
# wildcard) get a large bonus so exact > scoped path glob > broad wildcard.
specificity() {
  local pat="$1" prefix="${1%%[*]*}"
  if [[ "$pat" != *'*'* ]]; then
    echo $(( ${#pat} + 1000 ))
  else
    echo "${#prefix}"
  fi
}

# --- the selection contract ----------------------------------------------------------------------

# select_playbook GROUP_LABEL PKG...  — echoes the resolved playbook basename(s), space-separated.
# GROUP_LABEL is the raw label (e.g. "group:otel-core-contrib") or "" for an ungrouped PR.
select_playbook() {
  local label="$1"; shift
  local pkgs=("$@")
  local f grp

  # Rule 1: group label first (authoritative).
  if [[ -n "$label" ]]; then
    local want="${label#group:}"
    for f in "$PLAYBOOK_DIR"/*.md; do
      [[ "$(basename "$f")" == "README.md" ]] && continue
      grp="$(fm_group "$f")"
      if [[ -n "$grp" && "$grp" == "$want" ]]; then
        basename "$f"
        return 0
      fi
    done
    # Rule 4: label read but no dedicated playbook → fallback.
    echo "_default.md"
    return 0
  fi

  # Rule 2 + 3: glob fallback for ungrouped PRs, most-specific-wins per package.
  local matched=() pkg best best_score score g
  for pkg in "${pkgs[@]}"; do
    best=""; best_score=-1
    for f in "$PLAYBOOK_DIR"/*.md; do
      [[ "$(basename "$f")" == "README.md" ]] && continue
      while IFS= read -r g; do
        [[ -z "$g" ]] && continue
        if glob_matches "$g" "$pkg"; then
          score="$(specificity "$g")"
          if (( score > best_score )); then
            best_score="$score"; best="$(basename "$f")"
          fi
        fi
      done < <(fm_packages "$f")
    done
    [[ -n "$best" ]] && matched+=("$best")
  done

  if (( ${#matched[@]} == 0 )); then
    # Rule 4: no glob matched → fallback.
    echo "_default.md"
    return 0
  fi

  # Surface the *set* (unique), never a silent collapse.
  printf '%s\n' "${matched[@]}" | sort -u | tr '\n' ' ' | sed 's/[[:space:]]*$//'
}

# --- assertions ----------------------------------------------------------------------------------

fail=0
check() {
  local desc="$1" want="$2" got="$3"
  if [[ "$got" == "$want" ]]; then
    printf 'PASS  %-58s -> %s\n' "$desc" "$got"
  else
    printf 'FAIL  %-58s -> got [%s], want [%s]\n' "$desc" "$got" "$want"
    fail=1
  fi
}

echo "selection-contract check (docs/playbooks/)"
echo "-------------------------------------------"

# Worked example 1: grouped collector/contrib PR → otel-core.md (rule 1, direct group match).
check "group:otel-core-contrib label" \
  "otel-core.md" \
  "$(select_playbook 'group:otel-core-contrib')"

# Worked example 2: file-path-grouped tool-deps PR → _default.md (rule 1 read, no playbook, rule 4).
check "group:tool-deps label (no dedicated playbook)" \
  "_default.md" \
  "$(select_playbook 'group:tool-deps')"

# Worked example 3: ungrouped single-package bump, no glob match → _default.md (rule 2, rule 4).
check "ungrouped github.com/some/lib bump" \
  "_default.md" \
  "$(select_playbook '' 'github.com/some/lib')"

# Every other Renovate group label without a dedicated playbook also falls back (rule 1 -> 4).
check "group:dockerfile label (no dedicated playbook)" \
  "_default.md" \
  "$(select_playbook 'group:dockerfile')"
check "group:github-actions label (no dedicated playbook)" \
  "_default.md" \
  "$(select_playbook 'group:github-actions')"
check "group:otel-compgen label (no dedicated playbook)" \
  "_default.md" \
  "$(select_playbook 'group:otel-compgen')"

# Most-specific-wins (rule 3): a semconv module ungrouped matches both otel-core's
# go.opentelemetry.io/otel/** and semconv's go.opentelemetry.io/otel/semconv/** — the longer
# literal prefix (semconv) wins.
check "ungrouped otel/semconv module (most-specific-wins)" \
  "semconv.md" \
  "$(select_playbook '' 'go.opentelemetry.io/otel/semconv/v1.27.0')"

echo "-------------------------------------------"
if (( fail )); then
  echo "RESULT: FAIL"
  exit 1
fi
echo "RESULT: PASS"
