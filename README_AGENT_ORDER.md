# Agent Orientation Order

> **Status:** Active. This document defines the order in which an agent should
> orient itself before acting in the `fenix` repository. `CLAUDE.md` is the
> highest authority and overrides everything else. Do not let this file contradict
> `CLAUDE.md` or `AGENTS.md`.

## Read order (highest authority first)

1. **`CLAUDE.md`** — behavioral and workflow rules. These OVERRIDE default behavior
   and every other document in this repository.
2. **`AGENTS.md`** — task-presentation contract and push/hooks discipline.
3. **`docs/policies/HITL_AUTONOMY_POLICY.md`** — when approval is required and
   what autonomy is permitted.
4. **`docs/policies/RRI_POLICY.md`** — how to score task complexity and risk with
   the RRI formula; run `python3 scripts/rri.py` to compute it.
5. **`docs/decisions/`** — architecture decisions (ADRs) that constrain
   implementation; check relevant ADRs before touching their governed paths.
6. **`docs/plans/` and `docs/tasks/`** — the active plan and task ledger for the
   current workstream.
7. Product, BDD, design, config, and CI docs that constrain the current task
   (`docs/architecture.md`, `features/`, `DESIGN.md`, `.github/workflows/ci.yml`).

## Operating order for a task

1. Read this file and `CLAUDE.md` first.
2. Identify the affected files and which governing documents constrain the task
   (ADRs, plan, task file, BDD features, DESIGN.md for UI work).
3. Ensure a task file exists in `docs/tasks/` with the required frontmatter before
   writing any code (see `CLAUDE.md` — Task file discipline).
4. Compute RRI with `python3 scripts/rri.py --touches <path>... --cc <n> ...`.
5. If RRI > 25: present the task card and wait for explicit human approval.
   If RRI 0–25: execute directly (see `docs/policies/HITL_AUTONOMY_POLICY.md`).
6. Implement one task at a time.
7. Run the relevant QA gates before commit or push (see `CLAUDE.md` — Push discipline).
8. Update the task file status, report the outcome, and stop until the user
   confirms the next task.

## Related

- `CLAUDE.md`, `AGENTS.md`
- `docs/policies/HITL_AUTONOMY_POLICY.md`
- `docs/policies/RRI_POLICY.md`
- `docs/decisions/` (ADR index: `docs/decisions/README.md` — created in PAW-B2)
