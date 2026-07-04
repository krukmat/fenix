---
doc_type: task
id: CRIT-001
title: "RRI criticality signal (advisory criticality_suggested output)"
status: done
phase: implementation
week: 2026-W27
tags: [rri, criticality, peer-review, workflow]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [CRIT-002, CRIT-003, CRIT-004]
files_affected:
  - scripts/rri.py
  - scripts/rri_test.py
  - docs/policies/RRI_POLICY.md
created: 2026-07-04
completed: 2026-07-04
---

# Task CRIT-001: RRI criticality signal

**Plan**: [Critical-Task Classification and Local/Cross-Agent Reviewer Mix](../plans/critical_task_reviewer_mix_plan.md#rollout)

## Scope

Add an advisory `criticality_suggested` boolean plus a `criticality_reason`
string to `scripts/rri.py`'s output. This is purely additive — no change to
the RRI score, formula, penalties, or bands. The signal is derived from
values already computed inside `evaluate()`:

- `P >= 4` (post-floor-raise score, i.e. `scores["P"]`), OR
- the existing `matched_auth` boolean already returned by `match_rubric()`
  (anchor-rubric P floor >= 4 — auth/audit/rights/secrets paths).

`matched_auth` is already a strict subset of `P >= 4` (see `match_rubric`,
`scripts/rri.py:337`: `matched_auth = floors["P"] >= 4`, and floors only ever
raise, never lower, the agent-supplied P). So the OR condition simplifies to
checking the final `scores["P"] >= 4` — but the plan calls out `matched_auth`
explicitly because it is a **prior finding of fact** independent of what
number the agent typed for `--P`, useful for the reason string.

## Out of scope

- Any change to `criticality:` task frontmatter, labeling protocol, or
  reviewer concurrence (CRIT-002, CRIT-003).
- Any change to `scripts/peer-workflow-review.py` (CRIT-004).
- Any change to the RRI score, band, or penalty computation.

## High-Level Pseudocode

```
# In evaluate(), after `applied = detect_penalties(...)` and before building
# the return dict (scripts/rri.py:590-607):

criticality_suggested = scores["P"] >= 4
if matched_auth:
    criticality_reason = "anchor-rubric P floor >= 4 (auth/audit/rights/secrets)"
elif criticality_suggested:
    criticality_reason = f"agent-supplied P={scores['P']} >= 4"
else:
    criticality_reason = "P < 4; no critical-task signal"

# add both fields to the returned dict:
return {
    ...,
    "criticality_suggested": criticality_suggested,
    "criticality_reason": criticality_reason,
}

# render_markdown(r): append one line near the end, after the
# Decomposition/advisories lines:
#   **Criticality suggested:** yes|no — <criticality_reason>

# render_json(r): add the two keys at top level of the emitted object.
```

Tests to add in `scripts/rri_test.py` (new `CriticalitySignal` test class):
1. `P=4` via agent judgment (no rubric match) -> `criticality_suggested=True`,
   reason mentions "agent-supplied P".
2. Path with anchor-rubric P floor >= 4 (e.g. `internal/domain/auth/jwt.go`)
   with agent `p=0` -> floor raises P to 4 -> `criticality_suggested=True`,
   reason mentions "anchor-rubric".
3. `P=3` and no rubric match -> `criticality_suggested=False`, reason states
   "P < 4".
4. JSON output (`render_json`) includes both new keys.

Policy doc update: `docs/policies/RRI_POLICY.md` gets a short new subsection
(after "Reporting format" or as a note under the **P** variable row) stating
that the script also emits an advisory `criticality_suggested` /
`criticality_reason` pair when `P >= 4`, and that this is an input to the
(future, CRIT-002) task `criticality:` label — not itself the label.

## Acceptance criteria

- `python3 scripts/rri.py --json ...` output includes `criticality_suggested`
  (bool) and `criticality_reason` (string) at the top level.
- Markdown rendering includes a `**Criticality suggested:**` line.
- Existing tests in `scripts/rri_test.py` continue to pass unmodified (no
  score/band/penalty regression).
- New tests cover the 4 cases above; `python3 scripts/rri_test.py` passes.
- `docs/policies/RRI_POLICY.md` documents the new field.

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 0 | raw CC ~5 (additive fields, one small conditional) | High |
| F files | 2 | 3 touched files | High |
| D domain | 1 | `scripts/**` anchor floor | High |
| T coverage | 1 | reasonable tests exist for rri.py | High |
| A ambiguity | 0 | task fully specified | High |
| K coupling | 0 | pure additive output field | High |
| P impact | 1 | minor internal impact (advisory-only signal) | High |
| X context | 2 | 2-5 files | Medium |

Final RRI: 15 → Low band (0-25) → execute directly, no full approval packet.
