---
doc_type: adr
title: "ADR-030: Gemma local reviewer multi-pass contract and context-isolated adjudication (D14)"
status: Accepted
supersedes: ""
superseded_by: ""
---

# ADR-030: Gemma local reviewer multi-pass contract and context-isolated adjudication (D14)

- **Status:** Accepted
- **Date:** 2026-06-28
- **Deciders:** Fenix platform / DevEx
- **Scope:** Phase E of the Portable Agent Workflow port (`docs/plans/portable_agent_workflow_port_plan.md`)
- **Source:** Adapted from DubBridge ADR-034 (2026-06-24); all decisions re-ratified for the fenix context.
- **Closes:** "The local-Gemma review process has no auditable record, the reviewer runs a single noisy pass, and there is no structural isolation for finding disposition" — identified during PAW Phase E planning.

## Context

Fenix's Phase E workflow introduces two local-Gemma roles through Ollama:

- **Gemma Developer** (`scripts/delegate-low-rri.py`) — patch delegation for eligible Low-RRI (0–25) slices.
- **Gemma Reviewer** (`scripts/gemma-code-review.py`) — read-only advisory code review for Low/Moderate (0–40) development tasks.

Both write per-invocation `result.json` artifacts but produce **no aggregated record** across invocations. A single-pass reviewer is noisy: a finding present in one sampling may be absent in the next, and a single run gives no signal about which findings are stable versus likely false positives.

Separately, when the same primary agent that wrote the code also decides which findings to accept or dismiss, disposition is biased by anchoring even when the agent attempts simulated detachment. That bias is invisible without a structural isolation step and an audit field that measures it.

These are cross-cutting, hard-to-reverse decisions — they define a new telemetry surface, change the advisory contract every agent consumes for Low/Moderate work, and introduce a new deterministic gate (`adjudicator-packet.py`). They are recorded as an ADR because the decisions bind future agent behavior once enforced.

## Decision

### 1. Append-only audit log, emitted through the shared helper

Both roles emit one structured JSONL record per invocation to `logs/gemma-audit/YYYY-MM.jsonl`, written by `append_audit_log()` in `scripts/gemma_local.py`. The log is **local telemetry only**: git-ignored, never committed, never required by CI.

### 2. What is recorded — and what is never recorded

Automatic fields the wrapper can compute are always present: `role`, `outcome`, `done_reason`, `mode`, file/diff sizes, scope violations, apply result, `elapsed_s`.

Orchestrator-only fields (`task_id`, `rri`, `band`, `attempt`, `disposition`) are optional and default to `null`.

**Prompts are recorded.** The `system_prompt` and `user_prompt` sent to Gemma on each invocation are written verbatim. This is a first-party local log (git-ignored, never committed) and full prompt visibility is required for prompt tuning and failure reconstruction.

**What is never written:** raw target-file bodies beyond the diff already in the prompt; free-text fields are secret-redacted per `docs/policies/HITL_AUTONOMY_POLICY.md` before any write.

### 3. Reviewer runs N passes (default 3), sequential

The Reviewer runs N independent passes over the same packet (`--passes`, `FENIX_REVIEW_PASSES`), sequentially — the target hardware is single-GPU. `--passes 1` reproduces the prior single-pass contract exactly and is the rollback path.

### 4. Review is mandatory; isolated subagent is the fallback when Gemma fails

A pass succeeds on `PASS`/`FINDINGS` and fails on `BLOCKED`/timeout/malformed/`done_reason == "length"`. With **≥2 of N** successful passes the wrapper emits an aggregate (exactly 2/N ⇒ `degraded: true`).

With **<2** successful passes (quorum failure) or when Gemma is entirely unavailable, the agent **must** spawn a context-isolated subagent as the mandatory fallback reviewer. The fallback subagent receives the same isolation packet and its output is advisory. The primary agent remains orchestrator of record.

Review is **mandatory for all development tasks**. Neither quorum failure nor Gemma unavailability may skip the review entirely. No additional human approval gate beyond what the RRI band already requires is opened by this fallback path.

### 5. Reconciliation is deterministic and wrapper-owned

The wrapper, not the model, classifies findings across passes:

| Class | Definition |
|---|---|
| `consensus` | ≥2 passes, exact `(path, line, severity)` match |
| `pass_specific` | appears in exactly 1 pass |
| `severity_inconsistent` | same `(path, line)`, differing severity across passes |
| `location_inconsistent` | same path, line within ±3, from different passes |
| `likely_false_positive` | `pass_specific` ∧ out-of-scope |

Constants (±3, ≥2 = consensus) are fixed and unit-tested in `scripts/gemma_code_review_test.py`.

### 6. Backward-compatible artifacts and exit codes

Per-pass artifacts: `result.passK.json`. Aggregate: base `--out` path. Exit `0` on quorum met; non-zero only on quorum failure or operational failure.

### 7. Context-isolated adjudication trigger (D14)

When any of the following is true, the disposition is adjudicated by a **context-isolated reviewer** (label: **D14**):

| Trigger | Condition |
|---|---|
| Gemma unavailable or quorum failure | `gemma_blocked=True` |
| Consensus finding severity | any `consensus` finding has `blocking` or `major` severity |
| Slice band | band ∈ `{Med-high, Complex}` |
| Inter-pass disagreement | `severity_inconsistent_count > 0` or `location_inconsistent_count > 0` |

The isolated reviewer receives **only**: the final diff, the acceptance criteria, and the reconciled findings — never the development transcript, chain-of-thought, or dead-ends. This is enforced by `scripts/adjudicator-packet.py` via an allowlist assertion (`ALLOWED_PACKET_SECTIONS = {"diff", "criteria", "reconciled_findings"}`).

The adjudicator is **advisory**: the primary agent remains orchestrator of record and owns the final close per `docs/policies/HITL_AUTONOMY_POLICY.md`. Its only new obligation is to reconcile its disposition against the adjudicator's and record `disposition_divergence` in the audit log:

| Value | Meaning |
|---|---|
| `"none"` | adjudicator agreed with the primary's disposition |
| `"partial"` | a subset of findings diverged |
| `"full"` | adjudicator disposition fully differs from the primary's |
| `null` | adjudicator was not spawned, or field not yet populated |

For trivial cases (Low/Moderate band, Gemma available, no `blocking`/`major` consensus findings, no inter-pass disagreement) D14 does not fire.

**Escape hatch**: if a change is legitimately irreducible and exceeds the reviewability budget, the delivering agent records `D14-OVERRIDE: <reason>` (non-empty reason required) in the commit body. The override is captured in the audit log; an empty reason does not satisfy the escape.

## Consequences

### Positive

- Process tuning becomes data-driven: truncation, escalation, finding quality, and inter-pass disagreement become measurable signals.
- The reviewer surfaces stability of findings instead of a single noisy opinion.
- Context-isolated adjudication removes implementer anchoring bias from the disposition, and `disposition_divergence` makes that bias measurable instead of invisible.
- The HITL guarantee and read-only/advisory authority are unchanged.

### Negative / cost

- Reviewer latency roughly triples (~36–90 s for 3 passes on current hardware).
- A new local artifact surface to manage (rotation, git-ignore, redaction).
- More wrapper logic (N-pass loop + reconciliation + adjudicator trigger) to test and maintain.
- The isolated adjudicator, lacking development rationale, may re-raise intentional decisions as findings; the trigger gate bounds when this cost is paid.

### Neutral

- The audit log is advisory telemetry, not a gate; an empty or missing log never fails a task.
- `--passes 1` keeps the prior behavior available unchanged.

## Alternatives considered

- **Model-owned reconciliation**: rejected — not inspectable or testable. Reconciliation must be deterministic Python.
- **Keeping simulated self-review for disposition**: rejected when D14 fires — role-play detachment does not remove anchoring bias. Retained below the trigger threshold, where isolation cost is not justified.
- **Making the isolated adjudicator authoritative**: rejected — conflicts with the HITL orchestrator-of-record model. The adjudicator stays advisory.
- **No ADR, decisions in plan only**: rejected — the audit schema, reconciliation contract, and D14 label are durable and cross-cutting enough to warrant an indexed record.

## Scripts implementing this ADR

| Script | Role |
|---|---|
| `scripts/gemma_local.py` | Shared transport + `append_audit_log()` |
| `scripts/gemma-code-review.py` | N-pass reviewer + reconciliation |
| `scripts/check-review-budget.py` | Pre-delegation budget gate + D14-OVERRIDE escape |
| `scripts/adjudicator-packet.py` | D14 trigger gate + isolation packet builder |
| `scripts/gemma-push-review.py` | Push-time orchestration (PAW-E5) |
