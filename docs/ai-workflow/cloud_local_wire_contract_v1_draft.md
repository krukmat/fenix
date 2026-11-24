---
doc_type: proposal
id: FENIX-AI-W6-WIRE-V1
title: Cloud-local wire contract v1 implementation specification
status: draft_pending_independent_readiness
created: 2026-11-24
---

# W6.2–W6.3 — Wire contract v1 (draft)

This is the implementation-oriented refinement of [the W6 exchange draft](cloud_local_interaction_draft.md), not an active policy or implementation. It resolves field ownership, digest scope and transport capability ambiguity before code changes. The [W6 plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md) remains authoritative for task order and adoption. No peer verdict or RRI score is claimed.

## Trust boundary

The existing supervisor constructs TaskPacket after validating the task ledger and current policy. A trusted local adapter constructs ContextReceipt from actual input and wraps native results in ResultEnvelope. The model may propose patches or review findings; it never creates authoritative receipts, grants permissions, accepts tasks, or changes RRI. A remote provider that cannot access local files receives only explicitly authorized sanitized capsule content. GitHub references are not remote command requests. No new listener, service, or credential is required.

```text
Cloud supervisor --> TaskPacket --> local policy/scope validation
                                          |
                                   resolve context
                                          |
                      ContextReceipt <--- deliver to adapter
                                          |
                                  existing inference/QA
                                          |
                     ResultEnvelope <--- normalize terminal result
                                          |
                    existing RRI/HITL/independent peer acceptance
```

## Common fields (required, all messages)

| Field | Type / restriction |
|---|---|
| schema | Exact string: fenix.task-packet.v1, fenix.context-receipt.v1, or fenix.result-envelope.v1 |
| repository | Exact approved owner/repo identifier; never an arbitrary URL |
| source_commit | Lowercase full 40-character SHA-1 Git commit for the current repository |
| snapshot_digest | 64 lowercase hex SHA-256; binds the frozen input manifest including relevant working-tree changes |
| task_id | Stable ledger identifier, nonempty, max 128 ASCII characters |
| run_id | Unique supervisor-generated identifier for one task execution, nonempty, max 128 ASCII characters |
| attempt_id | Unique within run, nonempty, max 128 ASCII characters |
| input_digest | Lowercase 64 hex SHA-256; matches the validated TaskPacket |

IDs are correlation values only, not security credentials. A retry generates a different attempt_id while retaining run_id. Changed snapshot or task criteria require a fresh packet and fresh attempt; never silently reuse prior receipts.

## TaskPacket fields

| Field | Type |
|---|---|
| caller | {provider, model, channel}; bounded strings; channel enum local_cli, remote_handoff, local_inference |
| objective | Nonempty UTF-8 string |
| acceptance_criteria | Nonempty array of unique {id, text} records |
| allowed_paths | Array of exact repo-relative file paths, no globs |
| policy_refs | Nonempty array of ContextRef objects |
| rri_evidence_ref | ContextRef to current computed evidence, not a trusted number supplied by the model |
| context_refs | Array of ContextRef objects, including required task/plan/diff evidence for the actual stage |

ContextRef = {path, sha256, byte_length, source_kind, required, classification}. source_kind is git_blob or frozen_snapshot. required is boolean. classification is local_only or cloud_permitted as determined by the existing authority, never a model. path must resolve within the authorized root and must not contain absolute paths, dot segments, backslashes, NUL, or symlink escapes. Read exact bytes from a pinned blob or frozen manifest; recheck at time of use and reject mismatches.

## ContextReceipt fields

| Field | Type |
|---|---|
| delivery_mode | full_capsule, resolved_refs, or mixed |
| delivery_state | constructed, delivered, or receiver_confirmed |
| content_refs | Array of actual ContextRef objects used |
| delivered_sections | Array of section IDs and SHA-256 of exact delivered bytes |
| omitted_sections | Array of {id, reason}; mandatory omitted sections block candidate equivalence |
| receipt_digest | SHA-256 over canonical receipt without receipt_digest |

Constructed is not delivered. Delivered means the trusted adapter observed successful transfer to the provider boundary, not that the model read or understood each section. Receiver_confirmed requires an explicit supported receiver acknowledgement; ordinary model prose is insufficient. A terminal timeout before successful delivery may reference no receipt. Codex native post-code review records the actual base/snapshot inputs, and cannot claim explicit task or verification sections were delivered when they were not.

## ResultEnvelope fields

| Field | Type |
|---|---|
| receipt_digest | SHA-256 string or null when no successful delivery could be established |
| status | PROPOSED, NO_PATCH, BLOCKED, INVALID, TIMEOUT, REVIEW_REPORTED |
| evidence_refs | Array of ContextRef objects; no raw patch or prompt in exported telemetry |
| error_code | Null for normal terminal output; bounded machine-readable code for failure |
| usage | {input_tokens, output_tokens, cached_input_tokens, measurement, estimated_input_tokens} |
| duration_ms | Nonnegative integer measured with a monotonic clock |

Measurement enum provider_reported, estimated, unavailable. Token counters are nonnegative integers or null. Estimated tokens never populate provider-measured fields. A provider cache count must not be added to total input tokens without checking provider semantics. ACCEPTED is deliberately absent: existing supervisor policy derives acceptance from actual QA, current approval and independent peer evidence.

## Canonical encoding and digest scopes

- Reject duplicate JSON keys before parsing into a map, unknown fields, nonfinite numbers, unsupported versions and type coercions. V1 permits only safe integers where numeric values appear, boolean, null, string, array and object. Reject integers outside the interoperable exact range.
- Canonical bytes: RFC 8785 JCS-compatible ordering/escaping for this restricted domain, UTF-8 without BOM, no insignificant whitespace. Preserve array order and string content. Do not silently normalize Unicode. Test cross-language encoders before integration.
- TaskPacket.input_digest = SHA-256 of canonical TaskPacket object with input_digest omitted. ContextReceipt.receipt_digest = SHA-256 of canonical receipt with receipt_digest omitted (input_digest retained). The receipt binds the packet and snapshot. ResultEnvelope must match both identities exactly.
- ContextRef.sha256 = SHA-256 of exact source bytes, not a normalized text representation. snapshot_digest = SHA-256 of canonical, sorted authorized manifest of path, content hash, byte length and source kind. Dirty changes and deletions must be explicitly represented in the manifest used to construct the diff; reject a mutable snapshot if it cannot be frozen and checked.
- Digest equality proves content identity, not authentication or approval. Existing trusted supervisor, workspace controls, and reviewer identity remain the source of authority.
- Define input byte limits and provider prompt budgets as separately configured adapter limits; never treat transport-size savings as inference-token savings. Limit values remain pending benchmark and readiness review.

## Delivery decision

| Receiver capability | Allowed action |
|---|---|
| Can resolve immutable approved references with exact hashes | Resolve references and send delta; record actual delivery |
| Stateless inference or no shared context | Deliver complete permitted capsule |
| Cannot access required local-only evidence | BLOCKED for external delivery; retain existing authorized path |
| Context exceeds budget | Decompose or block; no silent omission of required evidence |
| Baseline stale, changed worktree or hash mismatch | BLOCKED; rebuild from current approved snapshot |

## Regression vectors (proposed, not yet executed against implementation)

| ID | Case | Expected |
|---|---|---|
| W6-WIRE-01 | Same packet keys permuted, same array order | Same digest |
| W6-WIRE-02 | Criteria array reordered or one character modified | Different digest |
| W6-WIRE-03 | Hash includes its own field | Reject mismatch |
| W6-WIRE-04 | Valid receipt paired with different attempt/snapshot | Reject |
| W6-WIRE-05 | Duplicate keys, NaN, unsafe integer, bool as number, unknown field/version | Reject |
| W6-WIRE-06 | Repo-relative traversal, symlink, stale worktree or content digest mismatch | Block |
| W6-WIRE-07 | Native Codex review omitted explicit criteria | Record actual coverage, not full equivalence |
| W6-WIRE-08 | Stateless Ollama receives only digest references | Full capsule or block |
| W6-WIRE-09 | Timeout before delivery | TIMEOUT, null receipt, measured duration, unknown usage |
| W6-WIRE-10 | Model output says ACCEPTED or supplies an approval | Invalid as authority; no closure |
| W6-WIRE-11 | Same terminal event replayed; conflicting payload under same ID | Deduplicate identical; reject conflict |
| W6-WIRE-12 | Required content local_only and remote receiver | Block external delivery |
| W6-WIRE-13 | Missing provider tokens or estimated-only tokens | Null measured fields |
| W6-WIRE-14 | Stale readiness artifact or missing mandatory peer gate | No supervisor acceptance |

## Scope and next action

Implement as opt-in pure validation/canonicalization helpers in scripts/workflow_exchange.py only after the W6.2 readiness gate. Unit-test W6-WIRE-01 through W6-WIRE-05 first; leave actual file resolution to W6.3 and provider adapters to W6.4. The existing native CLI and Ollama output contract stay unchanged. No production runtime changes.

As of 2026-11-24, the cloud execution environment does not expose authenticated Claude/Codex/Ollama executables or a repository checkout. Independent task-readiness remains NOT RUN; no artifact is invented. This draft can be reviewed without invoking a local model, but execution remains gated. Consolidate local work at the end wherever governance permits; do not call this design a certified implementation.
