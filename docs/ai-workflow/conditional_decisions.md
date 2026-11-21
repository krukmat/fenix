---
doc_type: plan
id: FENIX-AI-WORKFLOW-W4-W5-DECISIONS
status: rules_prepared_measurements_pending
updated: 2026-11-21
---

# W4.4 + W5.5 — Conditional model decisions and workflow optimization

**Scope: development-agent workflow only.** These rules are precommitted before the final Mac measurement block to avoid selecting winners after seeing data. There are no real current Mac results yet.

## W4.4: per-role evidence and decisions

For each role (bounded fix, complex implementation, documentation/analysis, optional structured precheck), record:

| Evidence | Requirement |
|---|---|
| Task correctness | Acceptance criteria and deterministic tests against frozen source and packets |
| Independent review | Required readiness and post-code verdict artifacts; no silent waiver |
| Human intervention | Retries, repairs, escalations and supervisor actions |
| Time | End-to-end seconds per accepted task plus local cold/warm microbenchmarks |
| Cloud expense | **Measured** paid provider tokens per accepted task; keep null if unavailable |
| Memory | Host memory pressure and model residency, not fabricated per-model RSS |
| Confidence | Sample size, outcome variance, reproducibility and missing evidence |

### Decision rules

1. **Safety regression or missing required review** → **REJECT / BLOCK**, regardless of apparent speed.
2. **No frozen equivalent test or too few valid cases** → **INCONCLUSIVE**, retain existing defaults.
3. **Quality worse or higher verified repair rate** → **RETAIN incumbent**, even if candidate generates faster.
4. **Quality non-inferior and sustained improvement in either wall time or actual cloud tokens per accepted task** → **PILOT** in a narrowly bounded lane, with reversible opt-in; threshold must be recorded against observed baseline.
5. **No distinct value over an existing model** → **DO NOT ADD** to the required worker stack.
6. Nimble structured prechecks must demonstrate avoided supervisor effort greater than their own latency and complication; they are not a mandatory additional inference call.
7. Exact Ollama tags and hardware context are part of evidence. Historical installation records do not establish current availability.

Output fields: `role`, `incumbent`, `candidate`, `attempted`, `accepted`, `verified_completion_rate`, `actual_cloud_tokens_per_accepted`, `median_wall_s`, `p95_wall_s`, `peak_memory_pressure`, `review_coverage`, `confidence_execution_pct`, `decision`, `reason`. `null` for missing metrics, never zero-fill.

## W5.5: optimization backlog and ranking

| Candidate | Existing evidence | Risk | Measurement needed | Default action |
|---|---|---|---|---|
| Reduce local reviewer 3→1 passes | Existing default = 3 and reconciliation logic | High: missed defects | Defect recall vs cost on frozen review corpus | Hold; no default change |
| Eliminate repeated model load | Ollama calls on 32 GB Mac can contend | Medium: resident memory | Cold/warm time, memory pressure and other workloads | Measure before keep-alive tuning |
| Trim packets | PAW-F17 existing compaction | Medium: evidence loss | Token difference AND retained acceptance evidence | Reuse; measure |
| Simplify fallbacks | Primary + permitted local fallback + critical advisory | High: blocked gates | Stage failure counts and latency | Preserve fail-closed behavior |
| Add Nimble precheck | Structured parser exists | Medium: extra call | Net time/tokens saved per accepted task | Opt-in only |
| Role-based routing | Low-RRI opt-in pilot exists | Medium: wrong model/task match | Verified completion by lane | Hold defaults pending W4.3 |
| Audit input-token coverage | Existing logs lacked measured prompt token fields | Low: instrumentation quality | Proportion of records carrying actual usage | Retain instrumentation |

Rank only with evidence: first safety, then measured cost/time reduction, then integration effort and rollback ease. `potential benefit` is hypothesis until tested. Do not convert token/s directly to workflow savings.

## Completion conditions

W4.4 and W5.5 remain **AWAITING_LOCAL_EVIDENCE** until W4.3 + W5.4 return reproducible results and applicable repository QA/peer gates are recorded. The current artifact is a conditional analysis template, not an endorsement of any model or performance claim.
