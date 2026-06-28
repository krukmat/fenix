---
doc_type: policy
title: "Required Reasoning Index (RRI) Policy"
governs: "task complexity scoring and model selection"
status: active
---

# Required Reasoning Index (RRI) Policy

> **Status:** Active. `CLAUDE.md` is the highest authority; this file is the
> detailed procedure it delegates to for complexity-and-risk scoring, model-tier
> selection, and autonomy-gate determination.

## Purpose

RRI estimates how much reasoning, context, caution, and verification a task
requires before an AI agent may safely implement it.

RRI **determines the approval gate and evidence required** before an agent may
implement a task. For bands **RRI 26+**, the HITL approval checkpoint is
mandatory; what the band controls is what evidence the agent must bring to it.
For band **RRI 0–25**, the agent executes directly without presenting a full
approval packet (see `docs/policies/HITL_AUTONOMY_POLICY.md` for the full rule).

## Formula

```
RRI = 100 × ((0.18·C + 0.12·F + 0.15·D + 0.15·T + 0.12·A + 0.12·K + 0.10·P + 0.06·X) / 5)
    + Penalties
```

Weight verification: 0.18 + 0.12 + 0.15 + 0.15 + 0.12 + 0.12 + 0.10 + 0.06 = **1.00** ✓

Each variable is scored **0–5**. The base term is therefore in **[0, 100]**.
Penalties push the score above 100.

## Variables

### How to obtain each variable

Objective variables must be **measured**, not estimated.
Subjective variables must be **judged using the anchor rubric** below so that
two independent agents score the same task to the same number.

| Var | Name | Nature | How to obtain |
|---|---|---|---|
| **C** | Cyclomatic complexity | Objective (proxy) | Count: `if`, `else if`, `switch` arm, `for`, `range`, `select`, `&&`/`\|\|` in conditions. Or run `gocyclo` (Go), `eslint complexity` (JS/TS), `radon cc` (Python). |
| **F** | Files affected | **Objective** | `git diff --name-only <base>...HEAD` — count the files. |
| **D** | Domain complexity | Subjective — anchor rubric | Classify the task's target path using the anchor table below. |
| **T** | Test-coverage risk | Semi-objective | Check coverage output for the affected file/module. If no tests exist in the area, score high. |
| **A** | Task ambiguity | Subjective | Is there a task file with acceptance criteria + happy/edge examples? Score near 0. Vague tasks score 5. |
| **K** | Coupling / side effects | Subjective — anchor rubric | Classify using the anchor table. |
| **P** | Public API / security / data impact | Subjective — anchor rubric (ADR-anchored) | Classify using the anchor table. |
| **X** | Context size required | Subjective | How many files/modules must the agent hold in mind? |

### Scoring bands per variable

**C — Cyclomatic complexity**

| Score | CC range |
|---|---|
| 0 | 1–5 |
| 1 | 6–10 |
| 2 | 11–20 |
| 3 | 21–30 |
| 4 | 31–50 |
| 5 | 50+ |

**F — Files affected**

| Score | Files |
|---|---|
| 0 | 1 |
| 1 | 2 |
| 2 | 3–5 |
| 3 | 6–10 |
| 4 | 11–20 |
| 5 | 20+ |

**D — Domain complexity**

| Score | Domain |
|---|---|
| 0 | Documentation, naming, formatting |
| 1 | Simple logic, constants, copy |
| 2 | Normal business logic |
| 3 | Integrations, workflows, state management |
| 4 | Platform-specific core logic, async orchestration, agent orchestration, permissions |
| 5 | Security, authentication, compliance, financial or critical data logic |

**T — Test-coverage risk**

| Score | Test state |
|---|---|
| 0 | Strong specific tests exist for the area |
| 1 | Reasonable tests exist |
| 2 | Partial tests exist |
| 3 | Weak or fragile tests |
| 4 | No tests in the affected area |
| 5 | No tests and critical logic |

**A — Task ambiguity**

| Score | Ambiguity |
|---|---|
| 0 | Exact task with acceptance criteria and happy/edge examples |
| 1 | Mostly clear |
| 2 | Some missing details |
| 3 | Requires significant interpretation |
| 4 | Very open-ended |
| 5 | Vague ("improve this", "make it better") |

**K — Coupling / side effects**

| Score | Coupling |
|---|---|
| 0 | Pure function |
| 1 | Isolated module with no side effects |
| 2 | Internal module with contained side effects |
| 3 | Database, API, filesystem, external service, or framework integration |
| 4 | Async behavior, events, queues, transactions, platform side effects |
| 5 | Distributed system behavior or critical external side effects |

**P — Public API / security / permissions / data impact**

| Score | Impact |
|---|---|
| 0 | No impact |
| 1 | Minor internal impact |
| 2 | Changes internal behavior |
| 3 | Changes internal API |
| 4 | Changes public API, permissions, ownership, data visibility, or persisted data |
| 5 | Security, authentication, authorization, data loss, compliance, or critical business risk |

**X — Context size required**

| Score | Scope |
|---|---|
| 0 | One function |
| 1 | One class or file |
| 2 | 2–5 files |
| 3 | One complete module |
| 4 | Several modules |
| 5 | Global architecture context |

## Fenix anchor rubric

Use this table to derive the **minimum floor** for D, P, and K when the task
touches these paths. Score higher if the specific change within the path warrants
it; **never score lower than the floor**.

| Task touches | D floor | P floor | K floor | ADR anchor |
|---|---|---|---|---|
| `docs/**`, `features/**`, naming, formatting | 0 | 0 | 0 | — |
| `scripts/**` | 1 | 0 | 0 | — |
| `bff/**` | 1 | 1 | 1 | ADR-009 |
| `mobile/**` | 2 | 1 | 2 | ADR-022 |
| `internal/domain/copilot/**`, `internal/domain/eval/**` | 2 | 2 | 2 | ADR-102 |
| `internal/domain/knowledge/**` | 2 | 2 | 3 | ADR-012 |
| `internal/domain/**` (any other subdomain) | 2 | 2 | 2 | — |
| `internal/api/**`, `internal/infra/**`, `pkg/**` | 2 | 2 | 2 | ADR-008 |
| `internal/domain/agent/**`, `internal/domain/workflow/**` | 3 | 2 | 3 | ADR-100 |
| `internal/domain/tool/**`, `internal/domain/policy/**` | 3 | 3 | 3 | ADR-017 |
| `internal/domain/auth/**` | 3 | 4 | 3 | ADR-017 |
| `infra/migrations/**` | 4 | 5 | 4 | ADR-017 |
| `internal/domain/audit/**` | 4 | 5 | 4 | ADR-017 |
| Secrets, credential storage, authentication/authorization system boundary | 5 | 5 | 5 | ADR-017 |

## Penalties

Apply each penalty independently; they are additive.

| Condition | Penalty |
|---|---|
| Refactor and functional behavior change combined in the same task | +8 |
| Tests missing **and** public/security/data impact is high (P ≥ 4) | +10 |
| Cyclomatic complexity > 30 (C ≥ 4) **and** domain complexity ≥ 3 (D ≥ 3) | +10 |
| Task touches authn, authz, permissions, security, ownership, or sensitive data | +10 |
| Task is likely to affect more than 10 files (F ≥ 4) | +8 |
| An architecture or process/policy decision is required | +12 |
| No clear verification strategy exists | +15 |

## Bands, autonomy gates, and model tiers

The HITL approval requirement applies at every band **except RRI 0–25**. For all
other bands, the band controls the evidence and gates the agent must satisfy
before and after that approval.

Effort, capability, thinking, and gate are each derived **in parallel** from the
RRI band — never derive one output from another.

| RRI band | Label | Effort | Model (economy path) | Model (balanced/premium path) | Thinking | Gate |
|---|---|---|---|---|---|---|
| **0–25** | Low | **S** | `claude-sonnet-4-6` | `claude-sonnet-4-6` | Off | Execute directly. No full approval packet required. |
| **26–40** | Moderate | **M** | `claude-sonnet-4-6` | `claude-sonnet-4-6` | Off | Present task card and wait for explicit approval. Confirm tests exist in the affected area. |
| **41–55** | Med-high | **L** | `claude-sonnet-4-6` | `claude-opus-4-8` | On | Plan + explicit acceptance criteria required before approval. |
| **56–70** | Complex | **L** | `claude-opus-4-8` | `claude-opus-4-8` | On | Plan first. **Decompose into subtasks before implementation.** Human reviews the plan. |
| **71–85** | High | **XL** | `claude-opus-4-8` | `claude-opus-4-8` | On | Characterization tests + explicit acceptance criteria + human reviews the **diff**. **Decomposition mandatory.** |
| **86–100** | Very high | **XL** | `claude-opus-4-8` | `claude-opus-4-8` | On | Do not implement directly. Produce an ADR + risk analysis + decompose into subtasks. |
| **> 100** | Excessive | **XL** | `claude-opus-4-8` | `claude-opus-4-8` | On | Architecture/design work must happen first. Re-scope before any implementation. |

### Model tier resolution

Model IDs above are current as of 2026-06-28. When selecting a model for a task,
verify against official Anthropic documentation — do not rely on stale memory for
"latest" or "best".

Thinking mode: activate for Med-high and above when the task requires multi-step
reasoning that cannot be validated incrementally. Do **not** activate for config
edits, doc updates, or tasks where the strategy is fully pre-defined.

### Low RRI handling

For final **RRI 0–25**, the active agent executes directly. No full approval
presentation is required. The agent must still include the RRI score in the task
summary and verify against acceptance criteria before marking the task complete.

## Decomposition triggers

Split a task into subtasks before implementing if **any** of the following apply:

- Final RRI ≥ 56. This is the default hard gate for Complex, High, Very high, and Excessive tasks.
- RRI > 70, or base RRI > 100 (before penalties).
- F ≥ 4 **and** K ≥ 3 — large change surface with high coupling; isolate each seam.
- C ≥ 4 **and** D ≥ 3 — the +10 penalty activates; separate complex logic into a testable unit first.
- The +8 penalty is active (refactor + behavior change combined) — always separate refactor from functional change into distinct tasks/commits.
- T ≥ 4 **and** P ≥ 4 (no tests + high impact) — first subtask must be characterization tests; implementation is the second subtask.

**Split target:** divide until each subtask scores RRI ≤ 55 with A ∈ {0, 1}
(own acceptance criteria + happy/edge examples per the workflow guide).

## Reporting format

Before every implementation, compute the RRI as a table. For RRI 26+, present it
in the task approval packet. For RRI 0–25, include it in the task file and final
report instead of presenting the full task for approval.

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 0–5 | How obtained (gocyclo / eslint / radon / estimate) | High / Medium / Low |
| F files | 0–5 | `git diff` count or `--touches` count | High |
| D domain | 0–5 | Anchor rubric row | High / Medium |
| T coverage | 0–5 | Coverage output or "no tests found" | High / Medium |
| A ambiguity | 0–5 | Task file has/lacks criteria + examples | High |
| K coupling | 0–5 | Anchor rubric row | High / Medium |
| P impact | 0–5 | Anchor rubric row + ADR cited | High |
| X context | 0–5 | Files/modules required | Medium |

Then state:

- **Base value:** `100 × (Σ / 5) = ___`
- **Penalties applied:** list each triggered penalty and its value.
- **Final RRI:** base + penalties = ___ → band ___ → tier ___ / thinking ___.
- **Gates for this band:** list the gates that apply.

Low-confidence scores on D, P, or K are themselves a signal: treat the variable
as one step higher when confidence is Low.

## Script automation

**Agents must run `python3 scripts/rri.py` instead of computing the formula,
floors, or penalties manually.** The script is the canonical RRI calculator.

### What the script decides vs. what the agent supplies

| Decided by `scripts/rri.py` (objective / derivable) | Supplied by the agent (irreducible judgment) |
|---|---|
| F score — counts `--touches` paths or `git diff`, maps to 0–5 | **C** — agent measures raw CC (gocyclo/eslint/radon), passes as `--cc <raw>` (or `--auto-cc`) |
| C score — maps raw CC to 0–5 via the policy CC table | **T** — agent measures via coverage tooling, passes as `--T` |
| D / P / K floors — derived from the anchor rubric; raises agent input, never lowers | **A** — task ambiguity (has acceptance criteria + examples?) |
| `many_files`, `complex_and_domain`, `no_tests_high_impact`, `auth_security` penalties | **X** — context size required |
| Band, Effort (S/M/L/XL), model tiers, thinking, gate | **D / P / K above the floor** + 3 intent penalties: `refactor_and_behavior`, `arch_decision`, `no_verification` |
| Decomposition-trigger detection | — |

### Invocation

**At task-presentation time** (before any code is written; diff is empty):

```bash
python3 scripts/rri.py \
  --touches internal/domain/auth/service.go \
  --touches internal/api/handlers/auth.go \
  --cc 14 \
  --D 0 --K 0 --P 0 \
  --T 2 --A 0 --X 2 \
  [--penalty refactor_and_behavior] \
  [--penalty arch_decision] \
  [--penalty no_verification]
```

`--touches` feeds both the F file count and the anchor-rubric floor derivation.
Repeat it once per affected path. The script raises D/P/K to their rubric floors
automatically and reports any raise in the evidence column.

**Post-implementation** (diff is available; omit `--touches`):

```bash
python3 scripts/rri.py --cc <raw> --D <0-5> --K <0-5> --P <0-5> \
  --T <0-5> --A <0-5> --X <0-5>
# F measured automatically from git diff --name-only main...HEAD
```

Use `--F <0-5>` only when git is unavailable.
Use `--C <0-5>` only when the raw CC value is unavailable (pre-computed score).

**JSON output** (for tooling or CI):

```bash
python3 scripts/rri.py ... --json
```

### Measuring C (raw cyclomatic complexity)

- **Go:** `gocyclo -over 0 <file.go>` (install: `go install github.com/fzipp/gocyclo/cmd/gocyclo@latest`)
- **TypeScript/JS (BFF/mobile):** `eslint --no-eslintrc --rule '{"complexity":["warn",0]}' <file>`
- **Python (scripts):** `python3 -m mccabe --min 1 <file>` or `radon cc -s <file>`
- **Manual:** `CC = E − N + 2P` (count: `if`, `else if`, `for`, `range`, `select case`, `&&`/`||` in conditions)

The `--auto-cc` flag delegates measurement to the platform-detected tool. When
the tool is absent or returns no results, C falls back to 0 with a Low-confidence
marker — the Low-confidence bump then raises the score by +1. When gocyclo is
available (installed via `make install-tools`), `--auto-cc` is reliable for Go paths.
