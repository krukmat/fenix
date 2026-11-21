---
doc_type: playbook
id: FENIX-AI-WORKFLOW-FINAL-LOCAL
status: prepared_not_executed
updated: 2026-11-21
---

# Final Mac block: W4.3 + W5.4

Run **only after** ChatGPT's non-local preparation and the necessary project gates. Do not download models, create branches, install CI or replace the user's current Ollama configuration.

## 1. Verify environment

- Confirm current `main` and a clean worktree before comparing frozen source.
- Run `ollama list` and use **actual** installed model names. Exact tags cannot be inferred from historic audit files.
- Check the preexisting `scripts/workflow_model_router_test.py`, `scripts/workflow_audit_aggregate_test.py`, and `scripts/workflow_local_probe_test.py` under normal Python unittest discovery.
- Verify both independent peer gates using the actual supported executables and preserve their real artifacts; an absent CLI is BLOCKED, not PASS.

## 2. Collect read-only existing workflow evidence

```bash
python3 scripts/workflow_audit_aggregate.py \
  --gemma-log-dir logs/gemma-audit \
  --peer-log-dir logs/peer-workflow-review \
  --output out/ai-workflow/workflow-audit.json
```

If logs are missing, the report should say `no_records`. Do not recreate history.

## 3. Run one sequential model microbenchmark

Prepare a short synthetic prompt file with no code secrets. Example safe fixture: `docs/ai-workflow/safe_probe_prompt.txt`. Replace model names below with **exact installed tags**:

```bash
python3 scripts/workflow_local_probe.py \
  --models 'EXACT_MODEL_TAG_1' 'EXACT_MODEL_TAG_2' \
  --prompt-file docs/ai-workflow/safe_probe_prompt.txt \
  --runs 3 --max-tokens 96 \
  --output out/ai-workflow/local-probe.json
```

The probe calls only loopback Ollama and never logs prompt text or generated content. `memory_before/after` are coarse macOS VM counters, not individual Ollama model RSS. A first run is **not guaranteed cold**; note pre-existing model residency.

## 4. Replay frozen W4 cases

Choose a balanced subset across task lanes from `docs/plans/ai_workflow_w4_benchmark_spec.md`. Freeze source, packet hash and acceptance criteria. Run each case in an isolated copy, one model at a time, with deterministic checks and independent review. Do not apply a candidate patch directly to `main` and do not permit protected operations to local models.

## 5. Evidence and handback

Return redacted JSON, model inventory, frozen scenario SHA/expected checks, acceptance outcomes, attempts, timeout logs, memory pressure samples and reviewer gate artifacts. No raw credentials or private prompts. W4.4 and W5.5 can be closed only after this evidence is assessed by ChatGPT.

## Not covered by the lightweight probe

The probe **does not** establish actual software-defect resolution rate, cloud supervision tokens, or end-to-end accepted-task performance. Those require the frozen task corpus and supervisor traces.
