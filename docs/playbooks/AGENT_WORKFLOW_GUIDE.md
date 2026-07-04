---
doc_type: playbook
title: "Agent Workflow Guide"
governs: "portable agent workflow order, task readiness, execution discipline, and closure reporting"
status: active
---

# Agent Workflow Guide

> **Status:** Active. This playbook is the source of truth for the portable,
> agent-agnostic workflow sequence in Fenix. Agent-specific entrypoints such as
> `CLAUDE.md` and `AGENTS.md` should reference this guide instead of duplicating
> the sequence. If a local authority file states a stricter repository rule, the
> stricter local rule wins until the documents are reconciled.

## Mandatory workflow before implementation

1. **Orient** - read `README_AGENT_ORDER.md`, then follow the governing local
   entrypoint for the active agent (`CLAUDE.md`, `AGENTS.md`, or equivalent).
2. **Analyze** - identify affected files and read the documents that constrain
   them: architecture, ADRs, current plan, task file, BDD/design docs, policies,
   and CI or hook rules.
3. **Plan** - confirm the active plan in `docs/plans/`. If no plan covers the
   work, create or update the plan before implementation.
4. **Task ledger** - ensure an individual `docs/tasks/task_*.md` file exists with
   required frontmatter and the task-card fields. Development tasks must include
   high-level pseudocode; new services or components must also include a system
   context diagram before pseudocode.
5. **Gate by RRI** - run `python3 scripts/rri.py` with the affected paths and
   measured/judged variables. Use `docs/policies/RRI_POLICY.md` for scoring
   rules and `docs/policies/HITL_AUTONOMY_POLICY.md` for approval gates. When
   the script emits `criticality_suggested` / `criticality_reason`, treat that
   as advisory input to the task's declared `criticality` label rather than an
   automatic classification.
6. **Peer readiness review** - before presenting the task card, run the peer
   workflow review script against the task file and governing docs. Resolve the
   reviewer from the caller's provider (see `## Peer review` below). A non-pass
   verdict blocks task-card presentation until the task file is revised, the user
   explicitly waives review, or the verdict is reported as blocked (peer CLI
   unavailable). Include the result in the task card under a peer-review approval
   field that makes the evidence file and approval state explicit, for example:
   `Peer readiness review approval: reviewer=<reviewer>; artifact=<artifact path>; status=PASS`
   (or `status=BLOCKED`). The artifact value is the proof file written by the
   review script; the status value is the approval verdict. For tasks that
   declare `criticality`, the readiness reviewer must explicitly concur with or
   dispute that label and its stated basis. Peer review does not replace
   RRI/HITL human approval.
7. **Present or execute** - for RRI 0-25, execute directly within the bounded
   low-band rules. For RRI 26+, present the task card and wait for explicit human
   approval before editing.
8. **Implement one task** - work only on the approved task. Do not start a
   downstream task just because it is listed in the plan.
9. **Verify** - run the relevant local QA gates for the files touched. If a
   required gate cannot run, stop and report that before any push.
10. **Peer code review** - before closing a development task, run the peer
    workflow review script against the diff. Resolve the reviewer from the
    caller's provider (see `## Peer review` below). A non-pass verdict blocks
    closure until the code is revised, the user explicitly waives review, or the
    verdict is reported as blocked. Include the result in the closure report under
    a peer-review approval field that makes the evidence file and approval state
    explicit, for example:
    `Peer code review approval: reviewer=<reviewer>; artifact=<artifact path>; status=PASS`
    (or `status=BLOCKED`). The artifact value is the proof file written by the
    review script; the status value is the approval verdict. For `critical`
    tasks, the primary cross-agent reviewer still governs the exit code and any
    local advisory review remains non-blocking. Peer review does not replace
    RRI/HITL human approval.
11. **Sync status** - update materially affected task, plan, ADR, dashboard,
    handoff, or audit artifacts before reporting completion.
12. **Close and stop** - report result, verification, files affected, reasoning
    effort, recommended model, and token estimate. Then present the next task card
    only if a next task is known, and wait for explicit confirmation.

## Task definition requirements

- Every discrete task needs its own `docs/tasks/task_*.md` record before
  implementation begins.
- Task cards must use unambiguous agentic English and the field names required by
  `AGENTS.md`.
- Task cards may recommend models from either OpenAI or Anthropic. Use a single
  provider-specific model id when one provider is clearly preferred for the
  task, or use a provider-qualified dual recommendation when both are valid.
- Canonical single-provider examples: `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.5`,
  `claude-sonnet-4-6`, `claude-opus-4-8`.
- Canonical dual-provider format: `OpenAI: <model id> | Anthropic: <model id>`.
- **Default rule:** when no single provider is clearly preferred for the task,
  task cards should use the canonical dual-provider format rather than picking a
  single provider arbitrarily.
- Default balanced recommendations are `gpt-5.4` for OpenAI and
  `claude-sonnet-4-6` for Anthropic. Use `gpt-5.4-mini` when the task
  explicitly prioritizes cost or speed over reliability. Use `gpt-5.5` or
  `claude-opus-4-8` only when the task clearly needs higher autonomy, broader
  codebase navigation, or harder multi-step reasoning.
- If a non-default model is recommended, justify the deviation in the task card
  summary or effort note.
- Development tasks must include concise, language-agnostic pseudocode in the task
  file and task card.
- A task that introduces a new service or component must include an ASCII system
  context diagram in the task file before pseudocode. The diagram must show
  upstream triggers, the component role, downstream consumers, and key invariants.
- Behavioral happy-path and edge-case examples are recommended for development
  work and required when a task opts into automated behavioral coverage
  certification.
- Docs-only, ADR, README, audit, handoff, and planning tasks may omit pseudocode
  and behavioral examples unless the task's main risk is behavioral correctness.
- Tasks that use the critical-task workflow should declare `criticality:
  critical | standard` plus a `criticality_basis:` field in the task record and
  task card, following the wrapper contract in `AGENTS.md` / `CLAUDE.md`.

## RRI, approval, and model guidance

The workflow derives approval gates, effort, model tier, and decomposition from
RRI instead of agent-specific templates.

| RRI band | Agent action |
|---|---|
| 0-25 | Execute directly after computing RRI; include the score in closure. |
| 26-40 | Present the task card and wait for explicit approval. |
| 41-55 | Present plan context, explicit acceptance criteria, and task card before approval. |
| 56+ | Decompose before implementation and get approval on the decomposed task. |

`docs/policies/RRI_POLICY.md` is authoritative for the formula, anchor rubric,
band table, decomposition triggers, and model-tier mapping. Do not copy those
tables into agent-specific templates.

`docs/policies/HITL_AUTONOMY_POLICY.md` is authoritative for explicit approval,
low-band handling, destructive action approval, commit/push approval, and blocked
verification reporting.

## Verification and push discipline

- `git push` is a final publishing step after local validation.
- Install hooks in a new environment with `make install-hooks`.
- Run the local QA gate that matches the touched area. For mobile/shared CI
  changes, the minimum gate is `bash scripts/qa-mobile-prepush.sh` or the
  equivalent commands listed in `AGENTS.md`.
- Do not push if a required local gate fails or cannot be executed. Report the
  blocker with the exact command and reason.

## Knowledge-management sync

Use Obsidian-facing docs as project tracking artifacts, not product behavior.
When work changes architecture, scope, requirements, roadmap, APIs, data model,
delivery status, or operating assumptions, update the relevant durable artifact in
the same task.

Use `doc_type` frontmatter for new tracking artifacts. The closed vocabulary is
enforced by `scripts/check_okf_frontmatter.py`.

## Agent-specific wrappers

Agent entrypoints should stay thin:

- `CLAUDE.md` should keep Claude Code setup, attribution, and repository-specific
  project context, then delegate workflow order to this playbook.
- `AGENTS.md` should keep connector-neutral push, hooks, reporting, and knowledge
  rules, then delegate shared workflow sequence to this playbook.
- Prompt or task templates should reference this playbook, `RRI_POLICY.md`, and
  `HITL_AUTONOMY_POLICY.md` rather than restating the full workflow.

## Peer review

Provider-aware reviewer resolution. The caller is the agent or CLI that is
executing the task. The reviewer is the independent peer that checks the work.

| Caller provider | Reviewer |
|---|---|
| `claude-code` | Codex |
| `codex` | Claude Code |
| `local-provider` (any local model) | Claude Code |
| `remote-provider` (any remote model except Claude Code) | Claude Code |
| `unknown` | Claude Code |

**Fallback reviewer policy:**

- The primary reviewer must always run first. A local-model fallback reviewer is
  permitted only after the primary reviewer returns a blocked condition caused by
  timeout or reviewer unavailability.
- The fallback reviewer is a backup path, not a normal reviewer selection. It
  must be recorded explicitly in the review artifact together with the primary
  reviewer failure that triggered it.
- A fallback verdict does not relax the fail-closed contract. Invalid fallback
  output, missing fallback infrastructure, or another blocked fallback attempt
  still produce a blocked artifact.

**Critical-task advisory reviewer policy:**

- The fallback reviewer policy above is unchanged for normal blocking review.
- For `post-code-review` runs with `--criticality critical`, run the primary
  cross-agent reviewer first exactly as usual and keep its verdict as the only
  exit-code-governing result.
- After the primary reviewer completes, run one additional advisory-only local
  reviewer attempt (`local-qwen`) and record it in the same artifact as an
  `advisory-local` attempt.
- The advisory reviewer never changes the process exit code. If it times out or
  hits a runtime failure, retry exactly once, unload the model after every
  attempt, and degrade to a recorded non-blocking advisory-blocked result if
  both attempts fail.

**Failure modes:**

- If the designated peer CLI is unavailable or unauthenticated, write a
  `blocked` artifact and report `BLOCKED` in the task card or closure report.
  The caller must stop rather than self-review.
- A non-pass verdict (`fail` or `blocked`) at the readiness gate blocks
  task-card presentation. A non-pass verdict at the code-review gate blocks
  task closure. In both cases, the caller must revise the work or obtain an
  explicit user waiver before proceeding.
- Peer review is a workflow reporting contract. It does not replace the RRI
  autonomy gate or any human approval required by `HITL_AUTONOMY_POLICY.md`.

**Invocation.** The gate is `scripts/peer-workflow-review.py`. Run the mocked
test suite via `make qa-peer-workflow-review`. Live review is invoked manually at
the two checkpoints:

```
# Before presenting a task card (readiness review):
python3 scripts/peer-workflow-review.py task-readiness \
  --caller-kind <claude-code|codex|local-provider|remote-provider|unknown> \
  --task docs/tasks/task_<id>.md \
  --plan docs/plans/<plan>.md \
  --task-card <task-card-preview-path>

# Before closing a development task (post-code review):
python3 scripts/peer-workflow-review.py post-code-review \
  --caller-kind <claude-code|codex|local-provider|remote-provider|unknown> \
  --task docs/tasks/task_<id>.md \
  --plan docs/plans/<plan>.md \
  --base <base-ref> \
  --criticality <standard|critical> \
  --verification-log <verification-log-path>
```

The script resolves the reviewer from `--caller-kind`, writes a redacted artifact
under `logs/peer-workflow-review/`, and exits 0 only on a `pass` verdict.

**Reviewer CLI overrides.**

- Set `FENIX_CODEX_BIN` to force the Codex executable path when Claude Code work
  must be reviewed by Codex and the shell environment does not expose `codex`
  on `PATH`.
- Set `FENIX_CLAUDE_BIN` to force the Claude Code executable path when Codex or
  another provider must be reviewed by Claude Code and the shell environment
  does not expose `claude` on `PATH`.
- Overrides win over `PATH` lookup. If an override is set but is not
  executable, the script fails closed with a `blocked` artifact that names the
  bad override.

**Troubleshooting order.**

1. Confirm whether an explicit reviewer override is needed:
   `FENIX_CODEX_BIN=/absolute/path/to/codex` or
   `FENIX_CLAUDE_BIN=/absolute/path/to/claude`.
2. Confirm the reviewer CLI exists and is executable in the current session:
   `which codex`, `which claude`, or the resolved absolute path.
3. Confirm the reviewer CLI is authenticated and usable outside the workflow
   gate: `codex login` / `codex doctor` or `claude` in non-mutating print mode.
4. Re-run `scripts/peer-workflow-review.py`. If the primary reviewer still
   returns `reviewer_unavailable` or `reviewer_timeout`, rely on the documented
   local fallback path and report the resulting artifact instead of self-review.

**Artifact:** the review script writes a JSON artifact at a path determined by
the script. Include the artifact path in the task card or closure report field.

## Related

- `README_AGENT_ORDER.md`
- `CLAUDE.md`
- `AGENTS.md`
- `docs/policies/RRI_POLICY.md`
- `docs/policies/HITL_AUTONOMY_POLICY.md`
- `docs/policies/TASK_LEDGER_CONTRACT.md`
