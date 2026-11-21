---
doc_type: plan
id: FENIX-AI-WORKFLOW-REVIEW-OPTIMIZATION
status: safety_patch_committed_measurements_pending
updated: 2026-11-21
---

# W3 — Review correctness, prompt efficiency and optional precheck

## Existing gates that MUST remain in place

- `scripts/peer-workflow-review.py task-readiness` before the task card is acted on.
- `scripts/peer-workflow-review.py post-code-review` after changes are verified.
- Independent primary selection (Claude↔Codex); fallback only under the currently allowed unavailability/timeout circumstances.
- Critical Qwen advisory remains separate from the independent primary reviewer.

## W3.4 correctness first

The original `parse_codex_review_output("")` returned PASS. The main branch now raises `InvalidVerdict` on empty/whitespace-only output and has two regression cases. **Source code is published, but independent peer review and repository QA are outstanding**. No claim of certified PASS.

## W3.3 optimize by evidence

PAW-F17 already compacts local fallback/advisory packets and protects task context, criteria, verification and diff. Do not duplicate compaction or automatically repeat it through a new Nimble model. Existing local code reviewer uses `FENIX_REVIEW_PASSES=3`; compare one and three passes on frozen defectful/non-defectful packets before deciding if a default change is safe.

Compare token totals and latency at the **accepted task** level, plus missed critical findings, reruns, truncation flags, fallback invocations and local memory pressure. Cache only immutable context metadata keyed by task/source SHA; never cache review verdicts or human approval.

## W3.2 Nimble: optional structured precheck

Schema: `{"status":"PASS|FINDINGS|ESCALATE","reason":"non-empty","evidence_refs":["file:line"]}`. Invalid output means `ESCALATE`. This check is advisory only and adds latency; enable it only if A/B evidence shows lower total cost or fewer faulty tasks. It cannot directly approve any action, satisfy deterministic tests or supplant either primary peer review.

## Blocking gaps

1. Run Python regression tests in an authorized repo environment.
2. Run real independent task-readiness and post-code gates, retain artifacts.
3. Run the local benchmark block before changing multi-pass defaults or installing new model dependencies.
