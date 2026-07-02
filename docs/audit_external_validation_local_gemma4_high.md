---
doc_type: audit
title: "External Validation Audit — Local Model (gemma4:26b-a4b-it-qat)"
status: complete
task: EXTVAL-AUDIT-LOCAL-001
created: 2026-07-01
---

# External Validation Audit — Local Model

## Runtime Manifest

| Field | Value |
|---|---|
| model | `gemma4:26b-a4b-it-qat` |
| parameters | 25.2B (A4B MoE, ~4B active/token) |
| quantization | Q4_0 |
| context_length.requested | 16384 |
| context_length.model_max | 262144 |
| ollama_version | 0.31.1 |
| host | Apple M5, 10 cores (4 Super + 6 Efficiency), 32 GB RAM |
| run_timestamp | 2026-07-01T13:56:37Z |
| prompt_eval_tokens | 7597 |
| gen_tokens | 2799 |
| throughput | 35.9 tok/s |
| total_duration_s | 90.5 s |

---

## Model Findings

*Verbatim output from `gemma4:26b-a4b-it-qat`. Not yet evaluated — see Our Evaluation below.*

### Summary

The environment is currently **not ready** for external validation. While the hardware and core software are sufficient, there are critical blockers regarding shell configuration (Go, Java, Android SDK), missing required LLM assets (`nomic-embed-text`), and uninstalled dependencies in the BFF directory. Additionally, there is a discrepancy between the audited installed models and the fallback model specified in the test plan that must be resolved to ensure smoke tests pass.

### Findings

1. **LLM Model Discrepancy (Fallback Model)**
   * **Issue:** Inconsistency regarding the available fallback model for agentic tasks.
   * **Location:** Document 1 (Local Host Baseline) vs. Document 2 (Target Stack / S0.3).
   * **Conflict:** Document 1's audit of installed models lists `gemma4:12b-it-qat`, but Document 2 specifies `gemma4:12b-mlx` as the fallback model and claims it is "installed on host." The audit in Document 1 does not show `gemma4:12b-mlx` as an installed model.
   * **Risk:** Smoke tests (S0.3) and T5 (Sales Brief/Copilot) will fail if the operator attempts to use the model specified in the plan that is not actually present on the host.

2. **Missing Embedding Model**
   * **Issue:** Required embedding model is not installed locally.
   * **Location:** Document 1 (Local Host Baseline / Backend Runtime) and Document 2 (Target Stack).
   * **Conflict:** Both documents identify `nomic-embed-text` as a requirement for knowledge ingestion and evidence generation, but the audit confirms it is currently missing from the local Ollama installation.
   * **Risk:** T2 (Knowledge and Evidence) will fail immediately upon attempting to ingest or search knowledge.

3. **Critical Shell Environment Blockers**
   * **Issue:** Essential binaries are either missing from `PATH` or not linked, preventing automation and backend execution.
   * **Location:** Document 1 (Local Host Baseline).
   * **Conflict:** Go is not on `PATH`; Java is installed via Homebrew but not linked; `ANDROID_HOME` is unset and `adb` is not on `PATH`.
   * **Risk:** The operator will be unable to run the backend (`go run`), execute Maestro tests, or interact with the Android emulator for T7 (Mobile Real-Mode Navigation).

4. **Missing BFF Dependencies**
   * **Issue:** Backend-for-Frontend (BFF) runtime environment is incomplete.
   * **Location:** Document 1 (Local Host Baseline / BFF Runtime).
   * **Conflict:** `bff/node_modules` does not exist; the audit notes that `npm ci` must be run.
   * **Risk:** The BFF service will fail to start during the "Runtime Start Sequence," blocking all mobile and web-based validation (T1, T3, T4, T5).

### Gaps Not Covered By Either Document

* **Data Seeding Requirements for Support Agent:** While Document 2 (T3) notes that a non-empty `subject` is required to avoid a 400 error, there is no specific instruction or script provided to ensure the test data used during validation contains these mandatory fields.
* **Latency/Performance Thresholds:** Document 1 identifies LLM latency as a potential risk for external demos, but neither document defines the "agreed external demo threshold" mentioned in T5.
* **Verification of Negative Fixture Check:** While the plan includes a `curl` command to check that the screenshot-mode endpoint returns a `404`, there is no instruction on how to handle or log a failure if it returns a `200` (which would indicate fixtures are active).

### Recommended Actions

1. **Fix Shell Configuration:** Export `JAVA_HOME`, set `ANDROID_HOME`, and update `PATH` for Go, Java, and Android platform-tools as outlined in Document 2, Section S0.1.
2. **Synchronize LLM Models:** Verify if the fallback model is `gemma4:12b-it-qat` or `gemma4:12b-mlx`. Pull the correct version via Ollama to ensure it matches the Test Battery Plan (S0.3).
3. **Install Required Assets:** Execute `ollama pull nomic-embed-text` and run `npm ci` in both `bff/` and `mobile/` directories.
4. **Validate Environment Variables:** Ensure a local `.env` file is created with non-placeholder values for `JWT_SECRET`, `SESSION_SECRET`, and `BACKEND_URL` before attempting the "Runtime Start Sequence."
5. **Perform Smoke Test S0.3:** Run the provided chat smoke test using the fallback model to confirm Ollama connectivity and model availability before proceeding to T1.

---

## Our Evaluation

Adjudication of each model finding against current repository evidence (`git diff HEAD`, `ollama list`, live file reads). Each finding is labeled: `confirmed`, `partially-correct`, `incorrect`, or `hallucinated`.

### Finding 1 — LLM Model Discrepancy (Fallback Model): PARTIALLY-CORRECT

**Evidence:** `ollama list` (2026-07-01) shows `gemma4:12b-mlx` is installed. The original environment audit (Document 1) listed `gemma4:12b-it-qat` — that was the state at audit time (2026-06-29). During readiness reconciliation for EXTVAL-AUDIT-LOCAL-001, the plan was corrected to `gemma4:12b-mlx` and the environment audit was updated to reflect the installed model.

**Verdict:** The model caught a real inconsistency that *existed in the documents as fed to it* — Document 1 still referenced `gemma4:12b-it-qat` at the time of the prompt. The underlying install state is now correctly reflected: `gemma4:12b-mlx` is installed and is the designated fallback. **No action needed on the model side; the document inconsistency was resolved prior to this run but not yet reflected in Document 1 at prompt time.** The environment audit (`docs/external_validation_environment_audit.md`) should be updated to confirm `gemma4:12b-mlx` as installed.

### Finding 2 — Missing Embedding Model (`nomic-embed-text`): CONFIRMED

**Evidence:** `ollama list` shows `gemma4:12b-mlx`, `gemma3:27b`, `gemma4:26b-a4b-it-qat` — `nomic-embed-text` is absent. This is a real blocker for T2 (Knowledge and Evidence) and for any backend path that calls the embedding service.

**Verdict:** Confirmed. `ollama pull nomic-embed-text` must be run before the battery. This is already listed as a Go prerequisite in the plan (S0.3) but has not been executed. **Action required: pull the model before battery.**

### Finding 3 — Critical Shell Environment Blockers (Go, Java, Android SDK): CONFIRMED

**Evidence:** The environment audit (Document 1) explicitly records Go as `command not found`, Java as not linked, and `ANDROID_HOME` as unset. These are known blockers documented in the audit. The plan (S0.1) lists the exact commands to fix them.

**Verdict:** Confirmed, and already documented as blockers. Not new information — the model correctly surfaced them as the highest-priority pre-battery action. **Action required: run the S0.1 setup commands before battery.**

### Finding 4 — Missing BFF Dependencies: CONFIRMED

**Evidence:** Document 1 records `bff/node_modules` as absent. `npm ci` in `bff/` is a documented prerequisite (S0.2).

**Verdict:** Confirmed. Already known and documented. **Action required: `cd bff && npm ci` before starting the BFF.**

### Gap: Data Seeding for Support Agent: PARTIALLY-CORRECT

**Evidence:** T3 in the plan requires creating a support case with `subject`, priority, account, and contact before triggering the agent. The plan does describe the steps but has no seed script or fixture. The model correctly identifies the absence of a script — the plan relies on the operator to create the data manually via API calls described in T1 and T3.

**Verdict:** Valid observation. The risk is low for a manual battery but real for a scripted one. Worth noting as a future follow-up (readiness script), not a blocker for a manual first run.

### Gap: Latency/Performance Thresholds: CONFIRMED

**Evidence:** T5 references "agreed external demo threshold" but no specific SLA is defined anywhere in the documents. The CLAUDE.md NFR targets `≤2.5s p95` for copilot Q&A and `≤5s p95` for summaries, but these are product targets, not external-demo thresholds. With Gemma 26B local (~35 tok/s) and a typical copilot response of 200-400 tokens, expect 6-12s for a copilot answer — above the NFR target.

**Verdict:** Confirmed gap. The demo threshold is undefined. **Recommend explicitly documenting the acceptable latency range for the external observer before the battery, and considering a pre-warming call so the model is loaded.**

### Gap: Negative Fixture Check Failure Handling: CONFIRMED

**Evidence:** The plan calls for `curl -s -o /dev/null -w '%{http_code}\n' ...` and expects `404`, but only says "Expected result: 404" without a stop/alert instruction if it returns `200`. `bff/src/routes/copilot.ts` line 177 shows the endpoint is wired and can return `200` if `ENABLE_SCREENSHOT_FIXTURES=true`.

**Verdict:** Confirmed gap. The negative fixture check must be a hard gate: if not `404`, stop and do not proceed. **Recommend adding an explicit `if [ $CODE != 404 ]; then abort; fi` gate to the pre-battery checklist.**

---

## Actionability Assessment

| Finding | Actionable? | Blocker for battery? | Owner action |
|---|---|---|---|
| F1 — Fallback model doc inconsistency | Yes | No (model is installed) | Update env audit doc to confirm `gemma4:12b-mlx` |
| F2 — `nomic-embed-text` missing | Yes | **Yes** (T2 fails) | `ollama pull nomic-embed-text` |
| F3 — Shell env (Go/Java/Android) | Yes | **Yes** (backend/Maestro) | Run S0.1 setup commands |
| F4 — BFF `node_modules` missing | Yes | **Yes** (BFF won't start) | `cd bff && npm ci` |
| Gap — Latency threshold undefined | Yes | No (operational risk) | Define threshold before external session |
| Gap — Fixture check no abort | Yes | No (process risk) | Add hard gate to checklist |
| Gap — Seed script missing | No (manual battery) | No | Future follow-up only |

**Battery go/no-go:** NOT GO until F2, F3, F4 are resolved. F1 is a doc update only.

**Model quality assessment:** `gemma4:26b-a4b-it-qat` produced 4 findings and 3 gaps with zero hallucinations. It correctly identified all real blockers present in the documents. Finding 1 reflects a document-state lag (the model saw the pre-reconciliation version of Document 1) rather than a model error. The gaps section added genuine value: the latency-threshold and fixture-check gaps were not flagged in the original human audit. Model output is assessed as **high quality for this task type**.
