# Agent Orientation Order

> **Status:** Active. This document defines the order in which an agent should
> orient itself before acting in the `fenix` repository. `CLAUDE.md` is the
> highest authority and overrides everything else. Do not let this file contradict
> `CLAUDE.md` or `AGENTS.md`.
>
> This file is an orientation index. The portable workflow sequence lives in
> `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` so agent-specific templates can
> reference the same agnostic procedure instead of duplicating it.

## Read order (highest authority first)

1. **`CLAUDE.md`** — behavioral and workflow rules. These OVERRIDE default behavior
   and every other document in this repository.
2. **`AGENTS.md`** — task-presentation contract and push/hooks discipline.
3. **`docs/playbooks/AGENT_WORKFLOW_GUIDE.md`** — portable workflow order for
   orientation, task readiness, RRI gating, execution, verification, and closure.
4. **`docs/policies/HITL_AUTONOMY_POLICY.md`** — when approval is required and
   what autonomy is permitted.
5. **`docs/policies/RRI_POLICY.md`** — how to score task complexity and risk with
   the RRI formula; run `python3 scripts/rri.py` to compute it.
6. **`docs/decisions/`** — architecture decisions (ADRs) that constrain
   implementation; check relevant ADRs before touching their governed paths.
7. **`docs/plans/` and `docs/tasks/`** — the active plan and task ledger for the
   current workstream.
8. Product, BDD, design, config, and CI docs that constrain the current task
   (`docs/architecture.md`, `features/`, `DESIGN.md`, `.github/workflows/ci.yml`).

## Operating order for a task

Follow `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`. In short:

1. Load local authority and governing context.
2. Ensure the plan and individual task ledger exist.
3. Compute RRI with `python3 scripts/rri.py`.
4. Resolve the HITL gate from `docs/policies/HITL_AUTONOMY_POLICY.md`.
5. Implement one approved task.
6. Run relevant QA gates.
7. Sync status artifacts, report closure, and wait before starting the next task.

## Related

- `CLAUDE.md`, `AGENTS.md`
- `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`
- `docs/policies/HITL_AUTONOMY_POLICY.md`
- `docs/policies/RRI_POLICY.md`
- `docs/decisions/` (ADR index: `docs/decisions/README.md` — created in PAW-B2)
