---
doc_type: adr
title: "ADR-035: Peer review and code review gates are unconditionally blocking, no waiver or BLOCKED-terminal exception"
status: Accepted
supersedes: ""
superseded_by: "ADR-037 (partial amendment — decision items #2 and #3 only; items #1, #4, #5 remain in force)"
---

# ADR-035: Peer review and code review gates are unconditionally blocking, no waiver or BLOCKED-terminal exception

- **Status:** Accepted
- **Date:** 2026-07-05
- **Deciders:** Repository owner (Matias Kruk) via explicit direction; recorded by Claude Code
- **Scope:** Phase F of the Portable Agent Workflow port (`docs/plans/portable_agent_workflow_port_plan.md`), specifically the peer readiness review (workflow step 6) and peer code review (workflow step 10) gates
- **Closes:** "The peer review and code review approval contract permits closing or presenting a task with `status=BLOCKED` as an acceptable terminal state, and permits an explicit user waiver to override a non-pass verdict" — identified as a governance gap during a peer-review-gate audit on 2026-07-05.

## Context

`CLAUDE.md`, `AGENTS.md`, and `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` define two peer-review checkpoints implemented by `scripts/peer-workflow-review.py`:

1. **Peer readiness review** — runs before a task card is presented, checking the task file and governing docs.
2. **Peer code review** — runs before a development task is closed, checking the diff.

Prior wording in all three files allowed two escape hatches around a non-`pass` verdict:

- `status=BLOCKED` (reviewer CLI unavailable or unauthenticated) was documented as an "acceptable" terminal state, letting a task be presented or closed without an actual independent review having occurred.
- A `fail` verdict could be overridden by "an explicit user waiver," letting a task proceed despite a reviewer having found a problem.

Both escape hatches were introduced deliberately in `task_paw_f1_peer_review_policy_contract.md` (the task that originally defined the contract) and were consistent with `docs/plans/portable_agent_workflow_port_plan.md` §4's explicit scope decision that Phase F enforcement is "scripted, report-contract blocking, not a PreToolUse denial" — i.e., the gate has always been a documentation contract that a compliant agent follows voluntarily, not a mechanism the tooling can force.

The repository owner requested that the gate become blocking **in all cases, without exception**. Two operational questions were resolved by explicit direction before this ADR was written:

1. **What happens when the reviewer CLI is unavailable/unauthenticated?** Decision: hard stop. No task may be presented or closed while the gate reports anything other than `PASS`. The caller must stop, report the blocker, and either revise the work or restore reviewer availability (fix authentication, `PATH`, or the `FENIX_CODEX_BIN` / `FENIX_CLAUDE_BIN` override) — not seek a waiver, and not treat `BLOCKED` as a valid resting state.
2. **Should this ADR also cover mechanical enforcement (a PreToolUse hook or CI job that makes the gate impossible to skip)?** Decision: no. Mechanical enforcement is out of scope here and is deferred to a separate future task. This decision only closes the wording loophole in the reporting contract; it does not change the fact that today nothing but agent compliance actually invokes the script.

This is a cross-cutting, hard-to-reverse governance decision — it changes the approval contract every agent (Claude Code, Codex, local, or remote) must follow at two mandatory checkpoints in every task's lifecycle — so it is recorded as an ADR rather than left as prose-only drift across three files.

## Decision

### 1. Only a `PASS` verdict unblocks presentation or closure

`Peer readiness review approval` and `Peer code review approval` fields must report `status=PASS` before a task card is presented or a task is closed. No other verdict value is an acceptable terminal state to act on.

### 2. `BLOCKED` is a hard stop, not a resting state

If the designated peer reviewer CLI is unavailable, unauthenticated, or times out, the gate script correctly writes a `blocked` artifact — this behavior in `scripts/peer-workflow-review.py` is unchanged. What changes is what the calling agent may do next: it must stop and escalate to the user, not present the task card or close the task while citing `status=BLOCKED`. The only ways forward are (a) revise the task file or diff and re-run the gate, or (b) restore reviewer availability and re-run the gate — in both cases the gate must return `PASS` before work proceeds.

### 3. No user waiver overrides a non-pass verdict

The prior "explicitly waived by the user" language is removed. A `fail` verdict cannot be waived past; the underlying issue the reviewer flagged must be resolved and the gate re-run.

### 4. Additive mechanisms are unaffected

- The **local fallback reviewer** (used only after the primary reviewer times out or is unavailable) remains part of the blocking contract — its own failure still produces a hard stop, not a permissible close.
- The **critical-task advisory reviewer** (`local-qwen`, additive for `--criticality critical` runs) remains explicitly non-blocking by design — it never governed the exit code before this decision and continues not to. This is not an exception to the "no exceptions" rule because the advisory reviewer was never the gate; the primary cross-agent reviewer's `PASS`/non-`PASS` verdict is the sole gate, unchanged by this ADR.

### 5. Mechanical enforcement remains explicitly out of scope

This ADR does not change `docs/plans/portable_agent_workflow_port_plan.md` §4's "no hook-enforced peer review in Phase F" scope decision. The gate remains a documentation/reporting contract enforced by agent compliance, not a `PreToolUse` hook or CI check. This is a known, accepted residual risk (see Consequences) and is tracked separately rather than solved by this decision.

## Consequences

**Positive:**
- Removes a documented, exploitable loophole where a task could be presented or closed without any actual independent review, simply by asserting the reviewer was unavailable.
- Removes an override path that let a `fail` verdict be waived without resolving the underlying finding.
- Makes the three governing documents (`CLAUDE.md`, `AGENTS.md`, `AGENT_WORKFLOW_GUIDE.md`) internally consistent — the prior text in `AGENT_WORKFLOW_GUIDE.md`'s failure-modes section contradicted itself ("the caller must stop rather than self-review" immediately followed by "...or obtain an explicit user waiver before proceeding").

**Negative / accepted risk:**
- If the peer reviewer CLI (Codex or Claude Code) is genuinely unavailable or unauthenticated in an environment, **no task can be presented or closed** until it is restored. This can halt all work in that environment. This is the intended, explicitly confirmed behavior — the alternative (treating `BLOCKED` as acceptable) is exactly the loophole being closed.
- Because enforcement is still "scripted, report-contract blocking" rather than hook/CI-enforced (see Decision §5), an agent that simply omits running the gate script is not mechanically prevented from doing so today. This ADR closes the *wording* loophole (an agent that reports its status honestly can no longer report `BLOCKED` or a waiver as sufficient); it does not yet close the *mechanical* gap (nothing stops a non-compliant agent from not running the script at all, or misreporting its result). Closing that gap requires a follow-up task to add real hook or CI enforcement, tracked separately and not implemented by this ADR.

## Related

- `docs/decisions/ADR-037-human-waiver-override-contract.md` (2026-08-03 — amends decision items #2 and #3 of this ADR: a human waiver may now override a non-pass verdict, provided the gate still ran and the waiver is an explicit, in-turn, verbatim-recorded `WAIVER: <reason>`. Items #1, #4, #5 below are unchanged.)
- `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` (workflow steps 6, 10; `## Peer review` section)
- `CLAUDE.md`, `AGENTS.md` (Reporting sections)
- `docs/policies/HITL_AUTONOMY_POLICY.md` (peer review and HITL are independent gates; this ADR does not change that independence)
- `docs/plans/portable_agent_workflow_port_plan.md` §4 (Phase F scope-excluded: no hook-enforced peer review — unchanged by this ADR)
- `docs/tasks/task_paw_f1_peer_review_policy_contract.md` (original contract that introduced the now-removed waiver/BLOCKED-acceptable language)
- `docs/tasks/task_paw_f15_peer_review_gate_unconditional_block.md` (implementation task for this ADR)
