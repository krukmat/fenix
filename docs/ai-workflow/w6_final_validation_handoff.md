---
doc_type: plan
id: FENIX-AI-W6-FINAL-VALIDATION
title: W6 consolidated execution-environment validation
status: pending_required_environment_and_governance
created: 2026-11-24
---

# W6 — one consolidated validation block

Purpose: avoid repeated handoffs or unnecessary inference. ChatGPT prepared candidate modules, shadow hooks, offline transport and stable-prefix pilot. Native workflow defaults and required RRI/HITL/independent gates unchanged. Historical readiness non-pass is not waived. The cloud cannot clone the full checkout or run authenticated Claude/Codex/Ollama CLIs. This handoff requests validation, not further delegated development.

## A. Deterministic checkout validation (no models)

On clean main, sync with origin using git pull --ff-only, record full HEAD and Python/CLI versions (version/help only). Preserve any user changes; stop if sync would be unsafe. No CI, new branch, model downloads or activation. Use repository root:

```sh
export PYTHONDONTWRITEBYTECODE=1
mkdir -p logs/w6-final
PYTHONPATH=scripts python3 -m unittest discover -s scripts -p 'workflow_*_test.py' > logs/w6-final/workflow-tests.log 2>&1
```

Check exit status and log before continuing. Then run the native regression modules with shadow disabled (all provider calls in these suites are mocked):

```sh
FENIX_W6_SHADOW=0 PYTHONPATH=scripts python3 -m unittest scripts/peer_workflow_review_test.py scripts/delegate_low_rri_test.py scripts/gemma_local_test.py scripts/rri_test.py > logs/w6-final/native-tests.log 2>&1
```

Check exit status and log. Then:

```sh
python3 scripts/workflow_prompt_pilot.py > logs/w6-final/prompt-preflight.json
```

Check exit status. Record case count, prompt byte counts, common prefixes and section hashes; compare compact/prefix section equality. The fixed historical fixture is not current gate evidence. Report actual counts and failures, not an assumed 96-test suite. If any deterministic regression fails, stop and return compact evidence for ChatGPT correction. Do not launch models for failures.

## B. Independent gates (only genuinely required)

Reconcile W6.1/W6.2 non-pass artifacts, W6.2 human criticality dispute and current task-specific RRI/HITL requirements. Independent readiness is deferred in cloud only, not waived; actual independent readiness/post-code review per active policy is required before adoption. Include all current applicable task/plan/spec/policy/QA evidence explicitly, not just links. Report authentic verdicts and local artifact paths. Stop for unresolved authority/criticality or required non-pass; do not self-certify. No optional local-model review/delegation; any unavoidable mandatory local-policy attempt or advisory requirement must be explicitly identified rather than bypassed. Do not duplicate previously valid gates without a material scope/snapshot change.

## C. Capability check before any experimental inference

Confirm installed CLI supports measured total and cached/uncached input usage for the exact provider/version (documentation/help/previous authorized artifact; no new model call merely to discover flags). No model switch between arms. Anthropic API total input = uncached input + cache creation + cache read when those exact fields and their semantics are exposed. OpenAI total input already includes cached input when documented for the actual response schema. CLI output may differ; if counters/semantics are unavailable, report UNKNOWN and stop the efficiency experiment. No raw prompts, secrets or unredacted provider artifacts exported.

## D. Bounded live smoke pilot, only after A–C pass and required approvals

Use approved frozen current readiness fixtures, one known defect and one clean case; same complete required evidence, expected checks, reviewer/model/version/timeouts. Use workflow_prompt_layout.prepare(packet, native_renderer, layout='native' or 'stable_prefix') for exact native vs reordered prompt, preserving every block. The pinned historical preflight is only a size fixture, not an authorization to send private repository content. Use the existing authorized primary provider command in a distinct experiment, not as formal gate or automatic fallback; preserve read-only behavior. Alternate paired arm order; explicitly separate cold/warm and cache creation/read. Record prompt/section hashes, all attempted usage, observed wall time, native verdict, expected defect result, and verified closure if available. Two pairs are a smoke check, not an adoption-grade benchmark. Stop on critical miss, policy regression, missing measurements or unsafe memory pressure. Do not start broad W6.8 without evidence that the candidate provides a meaningful benefit; report to ChatGPT for W6.9. Original total-input-token target is unchanged; caching alone is not reduction of total input.

## Return format (brief)

HEAD; deterministic tests PASS/FAIL and counts; preflight bytes/prefix; independent gate verdicts/artifacts; CLI usage capability; bounded pilot outcomes only if eligible; blockers. No raw prompt, credentials, model downloads, new CI or branches. Leave workflow defaults unchanged.

## Local report reconciliation (2026-11-24)

The first run on fb50b02f602a1b209130f08eb7c5f66105685230 returned workflow 126 PASS, native 3 FAIL of 239, preflight 9 PASS. Three native mocks corrected without changing runtime gate; two negative/precedence regressions added. Rerun native suite only initially, since no workflow code changed. Preserve prior workflow/preflight results as pinned evidence, not a certification of a new checkout. C source evidence: https://github.com/openai/codex/blob/rust-v0.171.0-alpha.13.2/codex-rs/exec/src/exec_events.rs declares usage counters. Runtime completeness and exact CLI behavior still unknown. No D, inference or optional reviewer calls until prerequisites. Human W6.2 criticality decision pending; do not relabel without it.
