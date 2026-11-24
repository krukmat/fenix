---
doc_type: task
id: FENIX-AI-W6-2
title: "Version exchange contracts"
status: needs_changes_human_criticality_pending
phase: W6
week: 0
tags: [ai-workflow, cloud-local]
fr_refs: []
uc_refs: []
blocked_by: [FENIX-AI-W6-1]
blocks: [FENIX-AI-W6-3, FENIX-AI-W6-4, FENIX-AI-W6-5, FENIX-AI-W6-6]
files_affected: ["scripts/workflow_exchange.py", "scripts/workflow_exchange_test.py"]
criticality: standard
criticality_basis: "Original standard label disputed by Codex. Proposed escalation to critical: forged/mismatched receipts and invalid acceptance could undermine RRI/HITL and independent review evidence downstream. Human resolution pending; no change to declared label before resolution."
created: 2026-11-24
completed: null
owner: "ChatGPT"
confidence_execution_pct: 95
---

# Task W6.2 — Version exchange contracts

**Plan:** [W6 execution plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md)
**Task:** W6.2
**Task file:** `docs/tasks/task_fenix_ai_workflow_w6_2.md`
**Plan file:** `docs/plans/ai_workflow_w6_cloud_local_execution_plan.md`
**Summary:** Define strictly validated TaskPacket, ContextReceipt and ResultEnvelope; distinguish proposals from acceptance.
**Code affected:** scripts/workflow_exchange.py, scripts/workflow_exchange_test.py
**Criticality:** standard (original label disputed; HOLD). Proposed human resolution: critical, despite RRI 28 and no automatic criticality suggestion. A validator defect can permit identity confusion, forged evidence or invalid acceptance by downstream consumers; the module itself cannot authorize actions. Do not execute until the human resolves the label and the reviewer concurs.
**Effort/reasoning:** Medium — bounded implementation with explicit negative tests and peer gates
**Recommended model:** claude-sonnet-5-5. Non-default recommendation: currently active successor to the repository's pinned Sonnet tier; official model catalog checked 2026-11-24 (https://platform.claude.com/docs/en/models/overview). This is a task recommendation, not a change to reviewer configuration or routing defaults.
**Estimated tokens:** ~3000 (planning estimate, not provider usage).
**RRI:** 28 (Moderate); developer-scored local session, 2026-11-24; evidence: `logs/rri-w6/w6_2.json` (owner-reported; gitignored, not inspected in cloud). RRI criticality suggestion: false. Recompute if scope or scoring inputs change.
**Peer readiness review approval:** reviewer=codex; artifact=`logs/peer-workflow-review/task-readiness_claude-code_by_codex_20261124T115837Z.json`; status=needs_changes (owner-reported prior result, not inspected here). W6.1 closure and human criticality resolution remain blocking; rerun with final task and governing contract.
**Peer code review approval:** NOT RUN; no artifact.

## Acceptance criteria

Authoritative proposed implementation detail: [wire contract v1](../ai-workflow/cloud_local_wire_contract_v1_draft.md), especially common fields, per-message fields, digest scopes and regression vectors. The reviewer must receive this governing specification explicitly; task and plan alone are not sufficient if linked content is unavailable.

- Support exactly fenix.task-packet.v1, fenix.context-receipt.v1 and fenix.result-envelope.v1. Require every documented common and message-specific field, including explicitly nullable fields; reject unknown fields, duplicate keys, wrong types, unsupported schemas and invalid cross-field combinations.
- Require exact repository/source_commit/snapshot_digest/task_id/run_id/attempt_id/input_digest equality across packet, receipt and result. Verify canonical SHA-256 digests, excluding each object's own digest field; no trust in model-supplied digests or acceptance.
- Require task/plan/policy/RRI and acceptance criteria references in a TaskPacket; validate metadata, not file contents or authorization. Actual file resolution and mandatory-section delivery checks remain W6.3; adapter transport remains W6.4.
- PROPOSED, NO_PATCH and REVIEW_REPORTED require matching delivered or receiver-confirmed receipts and appropriate evidence references; invalid or missing evidence is not a normal outcome. BLOCKED/INVALID/TIMEOUT permit null receipt only before verified delivery, with explicit failure code. Never emit or accept ACCEPTED, PASS as task closure, or a human approval claim. No filesystem writes, provider calls, commit/push, or gate changes from this module.
- Invalid objects produce a typed validation error; callers may construct a separately trusted INVALID/BLOCKED terminal event only when identity was independently established. Never construct a trusted result from malformed identity.
- Verify standalone fixtures for each schema, all mandatory field omissions, unknown versions, duplicate keys, noncanonical/invalid numbers, cross-attempt and snapshot mismatch, digest tampering, missing receipt or evidence, forbidden ACCEPTED and error-code consistency. Preserve native behavior and default routing; no integration until subsequent approved tasks.
Preserve current RRI/HITL and independent peer review requirements. Do not declare DONE until the applicable checks are complete.

## System context

```text
Approved task + RRI -> existing supervisor -> opt-in exchange adapter
                                      |                |
                                      |                v
                                      |        existing CLI / Ollama
                                      |                |
                                      v                v
                           existing QA and peer gates <- receipt/result
                                      |
                                      v
                              evidence and telemetry
```

No new service or process lifecycle; no direct database writes or new privileges. Errors return a blocked/invalid result; existing supervisor owns acceptance.

## High-level pseudocode

```text
load approved task and pinned source
validate applicable authority, context and scope
validate and canonicalize versioned envelope; bind identities
return evidence without changing active approval or routing defaults
```

## Draft evidence

See [W6 exchange draft](../ai-workflow/cloud_local_interaction_draft.md), [wire contract v1](../ai-workflow/cloud_local_wire_contract_v1_draft.md) and [integration draft](../ai-workflow/cloud_local_integration_benchmark_draft.md). These are proposed specifications, not implementation evidence. No code or tests have been executed; the earlier local readiness returned needs_changes.
