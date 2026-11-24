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
