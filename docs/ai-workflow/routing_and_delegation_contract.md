---
doc_type: contract
id: FENIX-AI-WORKFLOW-ROLE-ROUTING
status: opt_in_low_rri_pilot_governance_proposal_separate
updated: 2026-11-21
---

# W1–W3 — Routing, delegation and review contract

**Scope: development agent workflow only.** It must not change runtime model selection inside Fénix CRM.

## Current authority and established execution path

- `CLAUDE.md`, `AGENTS.md`, `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` and active policies remain authoritative.
- `docs/policies/LOCAL_MODEL_POLICY.md` requires a local attempt for eligible RRI 0–25 tasks; RRI 26+ local delegation remains prohibited.
- `scripts/delegate-low-rri.py` preserves the existing default Gemma behavior and tagged `PATCH | NO_PATCH | BLOCKED` contract.
- Existing deterministic scope validation and `git apply --check` controls remain in place.
- `scripts/peer-workflow-review.py` provides mandatory primary readiness and post-code peer gates. Qwen critical advice and optional Nimble precheck cannot replace them.

## W1.2 — Candidate roles, not verified assignments

| Role | Candidate | Intended use | Non-goal |
|---|---|---|---|
| Bounded fix | GPT-OSS 20B | Local simple bugfix, lint, characterization after policy screening | Commit, high-risk execution |
| Complex code | Qwen 3.6 35B-A3B | Decomposed coupled implementation if eligible | Override RRI/HITL |
| Documentation/analysis | Gemma 4 12B | Summarization, contract and doc drafts | Independent code correctness |
| Structured precheck | Nimble 9B | Optional task/criteria validation | Approval or review gate |
| Supervisor | ChatGPT/cloud reviewer | Task ownership, decomposition, independent review, acceptance | Unbounded inference repetition |

Exact model tags and installed inventory are **unverified** until W4.3. Keep existing defaults; no additional downloads are justified.

## W1.3/W1.4 — Opt-in route contract

`scripts/workflow_model_router.py` provides `configured_low_rri_model(role, rri, paths, task_id, environment)`. The `delegate-low-rri.py` CLI accepts:

```
--role-kind fix|complex_code|documentation|analysis
--rri <actual 0-25 score>
--task-id <stable ID>
--allow-path <bounded non-glob path> [repeatable]
```

Role ID variables are `FENIX_ROLE_FIX_MODEL`, `FENIX_ROLE_COMPLEX_CODE_MODEL`, `FENIX_ROLE_DOCUMENTATION_MODEL` and `FENIX_ROLE_ANALYSIS_MODEL`. The router fails closed on unknown roles, absent IDs/models, unsafe paths, protected boundaries and RRI outside 0–25. The downstream delegate verifies the selected exact tag in Ollama via the existing availability mechanism. This is opt-in and must not silently change the default model.

The decision is model *selection only*. It never invokes reviewers, commits, pushes, approves changes or relaxes existing task rules. The supervisor retains responsibility for approval and diff/test verification. Task RRI must be computed using the existing policy; a CLI argument is not independent proof of the risk score.

## W2.3/W2.4 — Moderate delegation

`moderate_delegation_advisory` is an **offline, disconnected policy thought experiment**. It returns `eligible=False` while `governance_policy_adopted=False`, the default. No code path invokes local RRI 26–40 workers. See [proposed ADR](proposals/moderate_delegation_adr_draft.md). Adoption needs separate governance authorization and independent gates; this proposal does not constitute a waiver.

## W3 — Review safeguards and efficiency

- Reject empty, whitespace-only or ambiguous native Codex review summaries instead of inferring PASS.
- Preserve the primary Claude↔Codex independent reviewer and both required checkpoints.
- Reuse PAW-F17 local prompt compaction; the local reviewer default remains 3 passes pending defect-recall experiments.
- `advisory_precheck` is a parser for optional Nimble JSON `PASS|FINDINGS|ESCALATE`; malformed output escalates, and `PASS` still requires peer review.
- Audit model IDs, measured prompt tokens, estimated packet tokens and measured response tokens separately. Do not equate tokens/s with accepted-task efficiency.

## Verification and limits

The three newly introduced standalone modules and their tests have been exercised on exact GitHub blob matches in an isolated cloud Python environment (28/28 PASS). Existing Fénix delegate, peer-review and policy integration gates have **not** been executed. This contract is operational guidance for the pilot, not certification of an RRI 26+ rollout.
