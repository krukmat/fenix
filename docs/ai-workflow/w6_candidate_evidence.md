---
doc_type: audit
id: FENIX-AI-W6-CANDIDATE-EVIDENCE
title: W6 standalone candidate evidence
status: candidate_isolated_tests_pass_not_certified
created: 2026-11-24
---

# W6.2–W6.4 candidate evidence

Baseline repository commit: 6bcb9502c104d4f1a484dffa1deeb91f4508207b. Cloud-only isolated Python workspace, not a full Fénix checkout.

Command: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest -v workflow_exchange_test workflow_context_test workflow_exchange_adapter_test`

Result: 33/33 unittest methods PASS (13 wire, 11 context, 9 adapter; synthetic fixtures and a temporary Git repository). No provider invocation or local LLM.

| File | Verified Git blob SHA |
|---|---|
| scripts/workflow_exchange.py | f6040853a040db06cf4b70ec759ceacde3432a03 |
| scripts/workflow_exchange_test.py | f8f4a63ec5069a38c8dfb8714c8090fe1a5840b9 |
| scripts/workflow_context.py | 72f2e4b94bf9b702a535f8cfadccc22201acd6ca |
| scripts/workflow_context_test.py | 66aecd18980f12b3a2c7af98921eea70c8a1d6ae |
| scripts/workflow_exchange_adapter.py | 81f5bef9d3a6fef0bac51ecaea226a6ec23b98e6 |
| scripts/workflow_exchange_adapter_test.py | 68b2fb80512e19b58ca691d9236f10479cdd9e13 |

## Candidate functionality

- W6.2: strict schema/version validation, exact common identity, canonical digest, invalid-result rejection and separated proposal vs acceptance.
- W6.3: pinned regular Git blob reads (reject symlinks), trusted frozen bytes, hash and manifest verification, budget, explicit external-export guard, receiver-known digest delta. Prepared receipts remain constructed. The trusted caller must establish real allowed paths, absent target claims and external classification; this module does not authenticate those assertions.
- W6.4: normalize native delegate and peer JSON artifacts into envelopes; explicit trusted observation needed before a delivery receipt; failed or unknown usage stays separate. No verdict is an acceptance. Native parsers and actual peer gates remain authoritative.

## Limitations and remaining work

No live provider transport or existing-script instrumentation is connected; the native artifact adapter is callable candidate code only. No independent peer gate, full repository QA, Mac benchmark, end-to-end accepted-task measurement or policy activation. Full RFC 8785 cross-language interoperability and actual provider output compatibility need further checks. Previous local W6.1/W6.2 readiness verdicts remain non-pass; W6.2 criticality dispute awaits human resolution. Cloud readiness deferred per owner instruction, not a waiver. No new CI, branch, service or product-runtime change.

The next candidate work is W6.5 advisory suitability, W6.6 trace aggregation, and W6.7 deterministic end-to-end compatibility; final integration must satisfy the remaining independent gates.

## W6.5–W6.7 extension — 2026-11-24

Command: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest -q workflow_exchange_test workflow_context_test workflow_exchange_adapter_test workflow_exchange_integration_test`

Result: 40/40 isolated tests PASS (33 prior + 7 synthetic pipeline/metrics/DSS tests). Exact tested Git blobs:

| File | Git blob SHA |
|---|---|
| scripts/workflow_exchange_metrics.py | c5f0daca2f3b8bf7c14bcbeca1e4405ccecbb2a8 |
| scripts/workflow_suitability.py | 098e2b83753b5f3863a58b80d7a2462d01c3ccd2 |
| scripts/workflow_exchange_integration_test.py | 93553419406ef0d8b893f473a82c9e90d3d3fe6a |

DSS is advisory and returns no authorization, rejects RRI >25 for local delegation, and returns UNKNOWN with insufficient matched verified observations. The minimum five-pair threshold is a provisional descriptive pilot threshold, not statistical certification. Attempt telemetry deduplicates exact replays and excludes conflicting runs; measured and estimated tokens remain separate. Accepted tasks, end-to-end latency and fallback rates remain unknown without trusted supervisor closure and complete traces. W6.7 exercises candidate functions on synthetic context/peer artifacts only. Actual provider transport, live legacy hooks, full repository regression tests, independent gates and Mac measurements remain pending. No new model calls, local delegation, CI, branch, or activation.


## W6.4a/b, W6.6a/b, W6.7a — native shadow integration

Baseline 112f23a3309e4668c1c3e88acd4560a786f75f23. Candidate shadow observation hooks now connect existing peer review (primary, fallback, advisory retries), delegate and aggregator entry points. Explicit opt-in: FENIX_W6_SHADOW=1; FENIX_W6_RUN_ID must be supplied uniquely by supervisor for one task run. Peer task ID from task frontmatter and delegation --task-id must match for correlation. Logs are private, exclusive per-attempt JSON files under logs/workflow-shadow; no prompts, patch bodies, exception text or raw verdict findings added. Writes are best effort, nonauthoritative. Prepared prompt bytes are not provider tokens. Codex native --base is separately classified; all delivery states remain unknown. Actual strict TaskPacket/ContextReceipt/ResultEnvelope delivery and trusted supervisor closure are not connected. The candidate adapter was corrected to accept the native out_of_scope review status.

Read separate aggregate: `python3 scripts/workflow_audit_aggregate.py --w6-shadow-dir logs/workflow-shadow`. Existing default aggregate shape is preserved. Do not turn on shadow globally; enable for bounded authorized benchmark runs only. The independent review and RRI/HITL gates are untouched.

Isolated cloud test command (run from staged Python workspace): `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest -q workflow_exchange_test workflow_context_test workflow_exchange_adapter_test workflow_exchange_integration_test workflow_shadow_test workflow_shadow_hooks_test`. Result: 60/60 PASS (40 prior, 1 native status case, 12 shadow cases, 7 native hook tests). The native hook tests executed the exact extracted peer invoke_reviewer/main, delegation main/_main and aggregator main with mocked provider/IO dependencies. Their extracted source hashes matched the staged Git blobs. Advisory retry and native local fallback transport bodies were inspected but not individually exercised; full native repository suite is not run because the cloud container cannot clone GitHub. Existing repository tests, full CLI compatibility, W6.2 criticality resolution, independent review and live benchmark remain pending. No provider/model invocation during these tests.

| File | Staged Git blob SHA |
|---|---|
| scripts/workflow_shadow.py | da2431c171b24be26266e441d04da5559659a959 |
| scripts/workflow_shadow_test.py | 9fd4d0b825394c074f75b9a0627504f97a603acf |
| scripts/workflow_shadow_hooks_test.py | 92ea71a878f1ef1b106bb819fafa7c54ecee8f0b |
| scripts/workflow_exchange_adapter.py | 1ce5ef1889129cd897ac1e559781b2b99978c4ca |
| scripts/workflow_exchange_adapter_test.py | c521c738ad52297cc42e60fd8ac72e79f5be4d9c |
| scripts/peer-workflow-review.py | c13e2dacb551576de1e3cdbc84fb17c52de3ee27 |
| scripts/delegate-low-rri.py | 21f11f5962aaed343abf3d81a809e304d1e82df8 |
| scripts/workflow_audit_aggregate.py | b6e48b66ca81655136b703de2df3fd5507ad5f3c |

Exact extracted source fragment blobs: peer f376d96040441f1277e573752b92a2a75b670d39; delegation 9e4ca7a399cfac3e132e1ce62821e21f11af77f0; aggregate 62b9bd61fca318d5ffdee4214909fde415f97b42. These are test input fragments, not replacement scripts.


## W6 candidate full/delta transport

Baseline e73d94580841226149cdcccb34960523246aaac3. Added serialization and receiver-side reconstruction without a network dependency. The supervisor supplies the separately validated packet, scoped cache, export authority and budgets. Full capsule, partial delta and cached references reconstruct identical bytes. Invalid digest, cache miss/corruption, unknown fields, replay under changed identity, unauthorized export and budget excess reject the transfer. The supervisor may resend a full capsule after rejection; no automatic retry or inference occurs. Reconstruction ACK verifies content/transfer binding but is not authentication and cannot establish provider receipt or acceptance. Existing shadow hooks and default workflow are unchanged. A synthetic test verifies fewer serialized transfer bytes with a cached 10,000-byte section; it does not measure model input tokens or latency.

Command: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest -q workflow_exchange_test workflow_context_test workflow_exchange_adapter_test workflow_exchange_integration_test workflow_shadow_test workflow_shadow_hooks_test workflow_exchange_transport_test`. Result: 70/70 PASS (60 prior + 10 transport tests), isolated cloud workspace with extracted/mocked native hook functions, not full repository QA. Existing datetime.utcnow deprecation warnings observed in extracted delegation function; behavior unchanged. No local model, provider invocation, new CI or branch.

| File | Git blob SHA |
|---|---|
| scripts/workflow_exchange_transport.py | 28ad3f39d46de73d5f1be2ecb94e5ffe4c074f06 |
| scripts/workflow_exchange_transport_test.py | d35199915a4b4ec15c6d25cb427579988c5fb28d |

Remaining: actual trusted provider boundary integration, full repository regressions, independent governance gates/criticality resolution, matched benchmark. No claim of cloud/local production integration or savings.


## Native input equivalence checkpoint — 2026-11-24

The published transport is now connected to a callable offline native-input equivalence bridge (not the active provider invocation path). It accepts a supervisor-authorized packet/cache and trusted renderer, reconstructs context, and requires byte-identical legacy input before returning prepared bytes. Claude readiness/post-code, Codex readiness and Ollama delegation are supported for this boundary test; Codex native post-code --base is explicitly excluded. No provider is invoked and no delivery/acceptance or token-usage evidence is created.

Fresh independent execution in the current cloud container: 9/9 bridge tests PASS using exact previously published wire and transport blobs. The earlier 70-test suite is historical evidence and was not rerun in this container. A synthetic fixture transferred 20,750 bytes full vs 2,065 bytes delta (transfer envelope only; separately transmitted packet and cache population excluded). Both rendered native inputs were exactly 14,041 bytes. This is a transport-only benefit, not an inference-token reduction. Actual current peer and delegation entry points still render their existing full/compacted native input; the strict bridge does not replace those paths.

Decision: do not run an expensive Mac benchmark expecting token savings from this transport alone. Separate transport equivalence (deterministic, zero inference) from an actual model-efficiency candidate (provider-supported prefix caching, validated prompt compaction, or retained provider context). For that latter lane require measured model input/cached token semantics, complete verified closures and comparable QA/peer outcomes; no additional or removed reviewer calls by default. Existing -30% input-token / -25% latency / -50% fallback targets are unproven. W6.8 remains HOLD for adoption-grade efficiency benchmarking until an actual distinct candidate and full regression/gate evidence exist. Do not activate a policy or route to achieve targets by bypassing controls.

Verification command: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest -v workflow_exchange_bridge_test` (9/9, 0.098 seconds, isolated cloud workspace, synthetic fixture renderer; no actual CLI or API). Dependencies rehydrated and verified via git hash-object: workflow_exchange.py f6040853a040db06cf4b70ec759ceacde3432a03; workflow_exchange_transport.py 28ad3f39d46de73d5f1be2ecb94e5ffe4c074f06.

| File | Tested Git blob SHA |
|---|---|
| scripts/workflow_exchange_bridge.py | b3781b91b1bcfe43696ddd12ba993a90916c0c2d |
| scripts/workflow_exchange_bridge_test.py | 22e0a90db658f6bc63689aa220a4f9b6ca14126a |
