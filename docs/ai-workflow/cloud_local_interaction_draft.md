---
doc_type: proposal
id: FENIX-AI-W6-EXCHANGE-DRAFT
title: Cloud-local interaction draft
status: proposed_not_active
created: 2026-11-24
---

# W6 exchange design — draft, not runtime policy

## W6.1 source audit

Pinned source commit: 0c0ba347dd03f48b874d285c521a368c644adb96.

| Finding | Source | Proposed mitigation |
|---|---|---|
| F01 Full prompt for primary Claude/readiness Codex; local-only compaction | scripts/peer-workflow-review.py:334–367,556–563,727–739 | Measure actual provider inputs; reuse PAW-F17 only where evidence coverage holds |
| F02 Native Codex post-code --base receives no explicit task/plan/verification packet | scripts/peer-workflow-review.py:646–659,727–738 | Explicit coverage receipt; never infer criteria coverage from a stored packet that was not transmitted |
| F03 Aggregates are not matched task traces | scripts/workflow_audit_aggregate.py:54–149 | Correlation keys and accepted-task aggregation |
| F04 Peer artifact and delegate audit lack a common run/attempt identity | scripts/peer-workflow-review.py:1000–1044; scripts/delegate-low-rri.py:971–997 | Stable IDs and immutable input receipt |
| F05 Exception paths return before normal audit entry | scripts/delegate-low-rri.py:1002–1013 | Terminal envelope for failures/timeouts as well as valid responses |

The preflight sentinel (scripts/agent-preflight.py) establishes orientation, not task-specific context equivalence. Existing RRI, scope and peer gates remain authoritative. No benchmark outcome or provider token savings is asserted by this audit.

## W6.2 contracts

Proposed transport-neutral JSON v1. Validate at the trusted adapter boundary, not by asking a language model to validate itself. Reject duplicate keys, unknown schema versions, malformed types and mismatched identity. Canonical serialization: UTF-8 JSON, deterministic key ordering and separators, reject NaN/Infinity; SHA-256 over exact canonical bytes. Define byte budgets separately from provider token budgets. Schema extension requires versioning. IDs are correlation identifiers, not authentication or authorization.

- TaskPacket: schema, repository, source_commit, task_id, run_id, attempt_id, caller, objective, acceptance_criteria, allowed_paths, policy_refs, rri_evidence_ref, context_refs, input_digest. Paths are bounded repo-relative references; no shell commands or approval claims are executable.
- ContextReceipt: schema, repository, source_commit, task_id, run_id, attempt_id, input_digest, content_refs with byte digests, delivered_sections, omitted_sections with reasons, delivery_mode, receipt_digest. Distinguish constructed, delivered, and receiver-confirmed evidence; a receipt does not prove that a model understood it.
- ResultEnvelope: schema, repository, source_commit, task_id, run_id, attempt_id, input_digest, receipt_digest (nullable only before delivery), status, evidence_refs, error_code, usage, duration_ms. Status vocabulary: PROPOSED, NO_PATCH, BLOCKED, INVALID, TIMEOUT, REVIEW_REPORTED. ACCEPTED is a supervisor-led derived state only when current required independent evidence and QA are present. Never trust a model-generated approval field.

## W6.3 context resolution

Use immutable references (commit/blob, repo-relative path, content SHA-256). Dirty working-tree content requires exact snapshot/diff digest; source commit alone is insufficient. Resolve only within authorized workspace, reject escaping paths, symlinks, invalid references and hash mismatch. Cache immutable content only; never cache a verdict or human approval. Do not reuse mutable task/plan content by filename alone. If the receiver cannot resolve references, send an authorized, self-contained capsule or BLOCKED. A stateless Ollama invocation must receive all essential context; a hash alone conveys no content. Provider prefix cache/resume is optional and measured separately. Full diff and mandatory criteria must never be silently truncated. Secret redaction happens before external transport; no-cloud content stays local.

## W6.4 adapters

- Claude Code: existing print/JSON adapter; record exact delivered packet and usage if provided.
- Codex CLI readiness: existing read-only exec; preserve reviewer independence. Post-code native review: record actual base/worktree and native review scope; test explicit context-aware alternative in shadow, not as a silent replacement.
- ChatGPT/Codex remote: approved repository-pinned task artifacts and sanitized handoff only. Local CLI/Ollama access is not assumed. GitHub is not an RPC channel; do not expose secrets, arbitrary command execution or local endpoints.
- Ollama: keep tagged PATCH/NO_PATCH/BLOCKED protocol and validators. Wrap its output at the adapter; preserve scope checking and apply ownership. Do not ask weak models to emit the entire exchange envelope.
- Keep primary peer selection and eligible fallback exactly as active policy. A failed transport never becomes PASS.

## W6.5 suitability advisory

Separate risk (RRI) from empirical model fit. Inputs: per-lane accepted-task success, comparable token and elapsed cost including retries, memory pressure, context budget and provider/model version. Report sample size, missing values and measured/estimated distinction. Candidate score is advisory; insufficient data produces UNKNOWN. Never allow RRI 26+ local delegation, bypass mandatory low-band attempt, or alter default model selection. Any proposed policy change is a separate governed decision. Avoid arbitrary fixed weights before calibration.

## W6.6 telemetry

Append structured redacted terminal events keyed by task_id/run_id/attempt_id plus stage/provider/model. Reject duplicate or conflicting event identities; keep immutable evidence refs and separate candidate vs baseline. Capture elapsed time, input/output tokens (provider-measured vs estimated vs null), bytes, cache counters if available, status, fallback reason and verification coverage. Record failure before returning from the adapter when possible. Missing usage is unknown, not zero. Aggregate all attempts per accepted task; report unsuccessful-task cost and acceptance rate separately. Do not export raw prompts, patch bodies, credentials or private paths. Protect logs using existing local retention boundaries.

## W6.7 deterministic regression matrix

- Same logical packet with reordered keys: identical canonical digest.
- Unsupported schema, duplicate JSON keys, nonfinite numeric values, oversized payload: reject.
- Mismatched task/run/attempt or changed source/worktree/digest: reject.
- Traversal, symlink, unauthorized path or secret-bearing external capsule: block.
- Missing context or receiver cannot resolve reference: authorized full capsule or block.
- Valid legacy response: unchanged native outcome; malformed/empty/truncated response: never PASS.
- Timeout/unavailable reviewer: actual blocked attempt preserved; fallback only per policy.
- Replay identical event: deduplicate; conflicting same ID: report invalid.
- Missing provider usage: null; estimated values never added to measured totals.
- RRI 26+ and protected paths: cannot become eligible through DSS.
- Native Codex review with missing criteria receipt: report delivery coverage gap; do not certify candidate equivalence.
- Mandatory review or QA absent: candidate cannot be accepted.

## W6.8/W6.9 experimental design

Freeze identical representative W4 task cases, baseline and candidate snapshots, model/provider versions, criteria and known seeded defects; pair runs and alternate order where possible. Local runs sequential; observe cold and warm residency independently, memory pressure, timeouts, total elapsed, all retries and measured tokens. Measure acceptance and critical defect recall first. Separate transport reductions from inference reductions, and estimated counts from provider-reported usage. Apply plan acceptance targets only where matched evidence exists. Any critical missed finding or policy regression is NO_GO. Missing data or independent reviews is HOLD. No change to active defaults is authorized by this draft.

## Status

Design preparation only; implementation, independent peer readiness, RRI and repository QA have not run. No new service, model, CI, or policy activation.
