---
doc_type: proposal
id: FENIX-AI-WORKFLOW-RRI-MODERATE-DELEGATION
status: proposed_not_adopted
updated: 2026-11-21
---

# Proposed ADR — Role-aware and cloud-supervised local delegation

## Context

Fénix's active `LOCAL_MODEL_POLICY.md` restricts local delegation to RRI 0–25 and requires attempting Gemma. Existing Claude/Codex peer reviews remain independent from the development worker. The proposed broader model matrix is NOT verified on the user's Mac.

## Decision proposed, NOT YET ACTIVE

1. Keep RRI scoring, criticality floors, human approval and independent review unchanged.
2. For RRI 0–25, permit an explicit installed role worker instead of a fixed default when all existing low-band path and packet controls pass.
3. For RRI 26–40, consider a **future pilot** only after a separately approved governance change, task-specific approval, standard criticality, complete acceptance criteria, deterministic tests and bounded non-sensitive paths. Policy remains **DENY** today.
4. RRI 41+, auth, audit, policy, permissions and migrations remain outside local delegation.
5. Models cannot commit, push, waive approval or certify peer reviews. Nimble structured precheck stays optional and advisory, never a substitute for mandatory independent reviews.
6. An invalid path, missing model ID, untested role, missing task ID, incomplete acceptance/tests, unsupported review result or missing HITL approval always blocks or escalates; never silently select another model.

## Adoption prerequisites

- Review changes as one governance-critical task, including `LOCAL_MODEL_POLICY.md`, `RRI_POLICY.md`, `HITL_AUTONOMY_POLICY.md` and `AGENT_WORKFLOW_GUIDE.md`.
- Create an official ADR with an assigned number only **after** independent peer readiness and explicit policy approval.
- Implement a disabled-by-default moderate mode with tests for all denial paths; do not enable by an undocumented environment toggle.
- Obtain W4 and W5 local benchmark evidence before replacing any default.
- Preserve ADR-035/ADR-037 gate and waiver semantics without inventing a PASS or a waiver.

## Current decision

**NOT ADOPTED.** The deployed pilot in `delegate-low-rri.py` is confined to RRI 0–25. This document is a discussion artifact, not authorization to modify governance invariants.
