---
doc_type: policy
title: "Local Model Policy"
governs: "local Ollama/Gemma delegation: eligible tasks, model selection, budgets, audit contract"
status: active
---

# Local Model Policy

> **Status:** Active. `CLAUDE.md` is the highest authority; this policy is subordinate.
> It defines when and how a local Ollama model may be used for task delegation and
> code review within the fenix agent workflow.

## Purpose

Enable cost-efficient, private, offline-capable task execution for low-complexity
work by delegating to a local Gemma model running via Ollama. The local model is
sandboxed: it receives a structured packet and returns structured text. The
orchestrating agent retains responsibility for validation, diff application, QA,
and task closure.

## Eligible delegation bands

Local-model delegation is only permitted for tasks in the **Low** RRI band (0–25).

| RRI band | Label | Local delegation |
|---|---|---|
| 0–25 | Low | Permitted via `delegate-low-rri.py` |
| 26+ | Moderate and above | Not permitted; use Claude (see `RRI_POLICY.md`) |

Tasks in band 26+ must follow the standard HITL approval gate with Claude as the
implementation model. Local delegation is never a substitute for the HITL gate.

## Model selection

| Role | Default model | Fallback model | Env var override |
|---|---|---|---|
| Developer (delegation) | `gemma4:26b-a4b-it-qat` | `gemma4:12b-it-qat` | `FENIX_LOW_RRI_MODEL` |
| Reviewer (code review) | `gemma4:26b-a4b-it-qat` | `gemma4:12b-it-qat` | `FENIX_REVIEW_MODEL` |
| Push reviewer | `gemma4:26b-a4b-it-qat` | `gemma4:12b-it-qat` | `FENIX_REVIEW_MODEL` (or `FENIX_PUSH_REVIEW_MODEL` for a push-specific override) |
| Peer-review fallback reviewer | `gemma4:26b-a4b-it-qat` | `gemma4:12b-it-qat` | `FENIX_REVIEW_MODEL` |

Use the fallback model when GPU VRAM is insufficient for the 26B model. The
fallback reduces quality but preserves the delegation workflow. Document the
fallback in the audit log entry.

## Generation budgets

| Parameter | Default | Env var |
|---|---|---|
| Context window (`num_ctx`) | 16 384 tokens | `FENIX_LOW_RRI_NUM_CTX` |
| Max new tokens (`num_predict`) | 4 096 tokens | `FENIX_LOW_RRI_NUM_PREDICT` |
| Temperature | 0.1 | — |
| Think mode | Off (delegation), On (push review) | — |
| Idle timeout | 60 s | `FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS` |
| Wall-clock cap | 900 s | `FENIX_LOW_RRI_MAX_WALL_SECONDS` |

Push review uses a larger context window (32 768 tokens) because CI logs are
included in the packet. It honors `FENIX_REVIEW_*` defaults and may use
`FENIX_PUSH_REVIEW_*` overrides for push-review-specific tuning.

## Peer-review fallback role

The local model may act as a backup peer reviewer only when the primary Phase F
reviewer has already been attempted and the result is blocked by timeout or
reviewer unavailability. It is never the default reviewer and may not replace
the primary reviewer for convenience, cost, or speed.

The fallback peer reviewer must:

- review the same packet shape as the primary reviewer;
- write an artifact that identifies both the failed primary reviewer attempt and
  the fallback reviewer verdict;
- fail closed when the fallback output is invalid, missing, or blocked.

This fallback role remains subordinate to `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`
and `docs/policies/HITL_AUTONOMY_POLICY.md`. A fallback PASS does not waive HITL
approval or any explicit human approval gate.

## Pre-delegation budget gate

Before delegating, the orchestrating agent (or pre-push hook) must run
`scripts/check-review-budget.py`. This gate fails closed when the diff exceeds
the reviewable line budget derived from `num_ctx` and `num_predict`.

**Escape hatch (D14-OVERRIDE)**: if a change is legitimately irreducible, add
`D14-OVERRIDE: <reason>` (non-empty reason required) to the commit body. The
gate will log the override and exit 0. The override is recorded in the audit log.

## Ollama connectivity

- Default host: `http://localhost:11434` (override: `FENIX_OLLAMA_HOST`).
- The model must be installed and listed by `GET /api/tags` before delegation.
- If Ollama is unreachable or the model is absent, the script writes a blocked
  artifact and exits 0 (non-blocking). A non-Gemma agent must handle the task.
- CI jobs that invoke local-model scripts are opt-in and `continue-on-error: true`.
  They will skip gracefully when `FENIX_OLLAMA_HOST` is not set in the runner.

## Path allowlist

The local model may only propose changes to files under these path prefixes:

```
internal/
mobile/
bff/
scripts/
docs/
features/
cmd/
pkg/
```

Any proposed path outside this allowlist results in `status: blocked`. The
orchestrating agent must escalate to the standard HITL flow.

## Audit contract

Every local-model invocation must produce an entry in `logs/gemma-audit/YYYY-MM.jsonl`.
Required fields:

| Field | Description |
|---|---|
| `ts` | ISO 8601 UTC timestamp |
| `role` | `developer` or `reviewer` or `push-reviewer` |
| `outcome` | Script-specific status (`PATCH`, `NO_PATCH`, `BLOCKED`, `PASS`, `FINDINGS`) |
| `done_reason` | Ollama `done_reason` (`stop`, `length`, etc.) |
| `elapsed_s` | Wall-clock seconds |
| `escalated` | Boolean — true if blocked artifact was written |
| `system_prompt` | System prompt text (secrets auto-redacted) |
| `user_prompt` | User/packet text (secrets auto-redacted) |
| `task_id` | Task ID if known, else null |
| `rri` | RRI score if known, else null |
| `band` | RRI band label if known, else null |

The audit log directory (`logs/gemma-audit/`) is excluded from Git via `.gitignore`.

**Secrets redaction**: `gemma_local.append_audit_log()` automatically redacts
values matching `api_key`, `token`, `password`, `secret`, `credential` patterns
before writing. Never log raw secrets.

## Invariants

- The local model proposes; the orchestrating agent validates, applies, and verifies.
- The local model never commits, pushes, or calls external services.
- A blocked result (Ollama down, model absent, path violation) is never a failure
  that stops the agent — it is a signal to escalate to the standard HITL path.
- `disposition_divergence` must be recorded in the audit log when an adjudicator
  (D14) is spawned (see `adjudicator-packet.py`).
- A local fallback peer review is allowed only after a blocked primary peer
  review attempt and must leave a traceable artifact chain.

## Related

- `CLAUDE.md` (highest authority)
- `docs/policies/RRI_POLICY.md` — RRI bands and model tiers
- `docs/policies/HITL_AUTONOMY_POLICY.md` — when Claude-based HITL is required
- `scripts/gemma_local.py` — transport layer
- `scripts/delegate-low-rri.py` — developer delegation
- `scripts/gemma-code-review.py` — reviewer
- `scripts/check-review-budget.py` — pre-delegation budget gate
- `scripts/adjudicator-packet.py` — D14 adjudication trigger
- `docs/decisions/ADR-030-gemma-local-adjudication.md` — D14 decision record
