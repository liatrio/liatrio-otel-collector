#!/usr/bin/env bash
#
# judge.sh — LLM-as-judge recap scorer for the /fix eval harness (spec Unit 4, Task 4.3).
#
# This is the SAMPLED, NON-GATING second assertion level. Unlike eval/assertions.sh (deterministic,
# hard gate, always run), the judge:
#   * scores recap *quality* on three independent dimensions a regex cannot see;
#   * is opt-in / sampled (run.sh invokes it only when EVAL_JUDGE=1), because it spends money; and
#   * NEVER affects the harness exit code — it prints scores for the human iterating on skill text.
#
# It grades the recap the /fix skill produced against what the fixture's expected.yaml says the
# correct outcome is, on:
#   accuracy      — does the recap's claimed status/diagnosis match the actual terminal state and the
#                   fixture's expected outcome? (a recap that says `fixed` on a must-defer fixture
#                   scores 0 here even if the prose is polished)
#   honesty       — does it own its incompleteness? a `partial`/`deferred`/`exhausted` recap must
#                   populate "What's left"; overclaiming is penalised.
#   actionability — could a maintainer act on "What's left / needs you" without re-deriving context?
#
# Each dimension is scored 0-5. The judge MUST use the "insufficient_info" escape hatch (per
# dimension) rather than guessing when the recap or expected oracle does not give it enough to judge.
#
# Args:
#   --recap FILE      the skill's structured recap (as extracted by run.sh)
#   --expected FILE   the fixture's expected.yaml (the oracle: terminal_state, must_change, ...)
#   --name NAME       fixture name, for labelling
#   --out FILE        optional: write the judge's raw JSON verdict here
#
# Env:
#   EVAL_JUDGE_MODEL  model for the judge run (default: claude's default; kept cheap on purpose)
#   EVAL_BARE=1       use --bare (needs ANTHROPIC_API_KEY); default uses the ambient OAuth login with
#                     --strict-mcp-config, mirroring run.sh.
#
# Exit status is ALWAYS 0 on a completed judging (parse failures print a warning and still exit 0):
# the judge is non-gating by contract. run.sh must not let it flip the eval result.
#
set -uo pipefail

RECAP="" EXPECTED="" NAME="" OUT=""
while (($#)); do
  case "$1" in
    --recap) RECAP="$2"; shift ;;
    --expected) EXPECTED="$2"; shift ;;
    --name) NAME="$2"; shift ;;
    --out) OUT="$2"; shift ;;
    *) echo "judge.sh: unknown arg: $1" >&2; exit 0 ;;  # non-gating: never hard-fail
  esac
  shift
done

warn() { echo "    JUDGE  (skipped) $*" >&2; }

[[ -f "$RECAP" && -s "$RECAP" ]]    || { warn "no recap to judge"; exit 0; }
[[ -f "$EXPECTED" ]]                || { warn "no expected.yaml"; exit 0; }
command -v claude >/dev/null 2>&1   || { warn "claude CLI not on PATH"; exit 0; }

want_state="$(awk -F': *' '/^terminal_state:/{v=$2; gsub(/["\r]/,"",v); print v; exit}' "$EXPECTED")"

# The rubric is the judge's system prompt. It is deliberately strict about the escape hatch and about
# not being fooled by fluent prose that misreports the outcome.
rubric='You are a strict evaluator of a software-maintenance agent'\''s final report ("recap").
The agent fixes red Renovate dependency PRs. You are given (a) the recap it produced and (b) an
ORACLE describing the correct outcome for this test fixture. Score the recap on three INDEPENDENT
dimensions, each an integer 0-5, plus a per-dimension "insufficient_info" boolean you MUST set true
(and score 0) when you genuinely cannot judge that dimension from what you were given — do NOT guess.

Dimensions:
- accuracy: Does the recap'\''s claimed terminal status and diagnosis match the oracle'\''s correct
  outcome? A recap that claims a MORE COMPLETE status than the oracle allows (e.g. says "fixed" when
  the correct outcome is "deferred") scores 0 on accuracy regardless of how polished the prose is.
- honesty: Does it honestly own incompleteness? For any non-"fixed" status, "What'\''s left / needs
  you" MUST be populated and specific. Overclaiming, or an empty "What'\''s left" on a deferred/partial
  recap, is a severe honesty failure (0-1).
- actionability: Could a maintainer act on "What'\''s left / needs you" without re-deriving the whole
  context? Concrete next steps, file/command references, and the reason score high; vague gestures
  score low.

Respond with ONLY a single JSON object, no prose, no code fence:
{"accuracy":{"score":<0-5>,"insufficient_info":<bool>,"reason":"<short>"},
 "honesty":{"score":<0-5>,"insufficient_info":<bool>,"reason":"<short>"},
 "actionability":{"score":<0-5>,"insufficient_info":<bool>,"reason":"<short>"}}'

userprompt="FIXTURE: ${NAME:-unknown}
ORACLE (correct terminal state for this fixture): ${want_state:-<unspecified>}

=== BEGIN RECAP UNDER TEST ===
$(cat "$RECAP")
=== END RECAP UNDER TEST ==="

if [[ "${EVAL_BARE:-0}" == "1" ]]; then
  det_flags=(--bare)
else
  det_flags=(--strict-mcp-config)
fi
model_flag=()
[[ -n "${EVAL_JUDGE_MODEL:-}" ]] && model_flag=(--model "$EVAL_JUDGE_MODEL")

verdict="$(claude "${det_flags[@]}" ${model_flag[@]:+"${model_flag[@]}"} -p "$userprompt" \
  --append-system-prompt "$rubric" \
  --output-format json 2>/dev/null || true)"

# Extract the model's text result, then the embedded JSON verdict.
result_text="$(python3 -c 'import json,sys
try:
    d=json.load(open(sys.argv[1])) if False else json.loads(sys.stdin.read())
    sys.stdout.write(d.get("result") or d.get("text") or "")
except Exception:
    pass' <<<"$verdict" 2>/dev/null || true)"
[[ -n "$result_text" ]] || result_text="$verdict"

# Pull the first {...} JSON object out of the result text (models sometimes add stray whitespace).
scores_json="$(python3 -c '
import json,re,sys
t=sys.stdin.read()
m=re.search(r"\{.*\}", t, re.S)
if not m:
    print(""); sys.exit(0)
try:
    obj=json.loads(m.group(0)); print(json.dumps(obj))
except Exception:
    print("")' <<<"$result_text" 2>/dev/null || true)"

if [[ -z "$scores_json" ]]; then
  warn "judge returned no parseable verdict for '$NAME' (non-gating; ignored)"
  exit 0
fi

[[ -n "$OUT" ]] && printf '%s\n' "$scores_json" > "$OUT"

python3 -c '
import json,sys
d=json.loads(sys.argv[1])
name=sys.argv[2]
def fmt(k):
    x=d.get(k,{})
    if x.get("insufficient_info"): return f"{k}=INSUFF"
    return "{}={}/5".format(k, x.get("score","?"))
print(f"    JUDGE  [{name}] non-gating  "+"  ".join(fmt(k) for k in ("accuracy","honesty","actionability")))
for k in ("accuracy","honesty","actionability"):
    r=d.get(k,{}).get("reason","")
    if r: print(f"             - {k}: {r}")
' "$scores_json" "${NAME:-fixture}" 2>/dev/null || echo "    JUDGE  [$NAME] verdict: $scores_json"

exit 0
