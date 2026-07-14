# `fix-v1` — versioned skill snapshot slot

This directory holds the **frozen snapshot** of `.claude/skills/fix/SKILL.md` under test by the eval
harness (spec Unit 4; A/B baseline). `eval/run.sh --skill-version fix-v1` copies `SKILL.md` from
here into each fixture clone instead of the repo's live skill, so a regression can be attributed to a
specific wording change (v1 vs. v2).

**Status:** `SKILL.md` is populated — a frozen copy of the canonical `.claude/skills/fix/SKILL.md`
taken at the Task 3.0 green milestone (sub-task 3.13). `run.sh` with no `--skill-version` uses the
live repo skill; `run.sh --skill-version fix-v1` pins this snapshot. Task 4.0 adds `fix-v2/` for the
A/B comparison.
