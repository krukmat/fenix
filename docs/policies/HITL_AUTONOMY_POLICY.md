---
doc_type: policy
title: "Human-in-the-Loop (HITL) Autonomy Policy"
governs: "when explicit human approval is required and what autonomy is permitted"
status: active
---

# Human-in-the-Loop (HITL) Autonomy Policy

> **Status:** Active. `CLAUDE.md` is authoritative on conflict. This policy
> consolidates the approval and autonomy rules already stated in `CLAUDE.md` and
> `AGENTS.md` into one referenceable document.

## Principle

The agent plans and proposes; a human approves before implementation. Irreversible
or outward-facing actions require explicit human sign-off. When uncertain whether
approval is required, default to asking.

## Always requires explicit approval

- Starting any implementation task with **RRI > 25**, even if a plan was approved
  in a prior session. Approval does not carry across sessions or across tasks.
- Deleting or overwriting files or data.
- Committing, pushing, or any outward-facing action (PRs, external API calls,
  messages to external services).
- Schema migrations (`infra/migrations/`).
- Changes to governance-critical invariants (audit, policy, auth boundaries).
- Security or permission boundary changes unless the specific task is already
  explicitly approved.

The only exception to the approval gate is when the user explicitly says
"proceed without asking" for a clearly bounded scope, or when the computed RRI is
0–25 and the task stays within the low-band handling rules below.

## Low-band handling (RRI 0–25)

When the computed RRI falls in the **0–25 Low band**, the agent executes directly
without presenting a full approval packet. No local-model delegation is configured
in this project (Phase E deferred).

The agent must still:

1. Compute RRI with `python3 scripts/rri.py` before starting.
2. Verify against all acceptance criteria before marking the task complete.
3. Run the relevant QA gates (see `AGENTS.md` and `CLAUDE.md` push discipline).
4. Include the RRI score and verification result in the task closure report.

## Approval checkpoint wording

When approval is required (RRI > 25), end the task card presentation with:

```
Execution has not started. Approve this task to proceed.
```

## Permitted without prior approval

- Read-only analysis, repository search, and codebase navigation.
- Reading affected files and governing documents.
- Drafting plans, task files, ADRs, and proposals (no code execution or file writes
  beyond the draft).
- Computing RRI.
- Running non-destructive validation commands.
- Non-destructive documentation or configuration fixes only when the user explicitly
  asked for that bounded cleanup.

## Safety rules

- Do not commit with broken tests. Run all relevant QA gates before commit or push.
- Ask before deleting files or data.
- Surface contradictions instead of guessing through them.
- Redact secrets and credentials in logs, reports, prompts, and artifacts.
- Report failed, skipped, or unavailable verification steps honestly — do not
  omit them from the closure report.
- If a required local QA gate cannot be executed, stop and report that explicitly
  before pushing.

## Related

- `CLAUDE.md` (highest authority)
- `AGENTS.md`
- `README_AGENT_ORDER.md`
- `docs/policies/RRI_POLICY.md`
