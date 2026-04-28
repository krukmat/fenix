# AGENTS.md

## Push Policy

- `git push` is the final step, not the first validation step.
- Before any push, run all relevant local QA gates for the area touched by the change.
- If a required local gate cannot be executed due to environment limits, stop and report it before pushing.

## Mobile Rule

When a change touches `mobile/` or shared files that affect mobile CI, the minimum required local gates are:

- `bash scripts/check-no-inline-eslint-disable.sh`
- `cd mobile && npm run typecheck`
- `cd mobile && npm run lint`
- `cd mobile && npm run quality:arch`
- `cd mobile && npm run test:coverage`

Preferred shortcut:

- `bash scripts/qa-mobile-prepush.sh`

## Hooks

- **After cloning the repo or setting up a new environment, ALWAYS run `make install-hooks`** to activate pre-push QA gates.
- The `pre-push` hook automatically detects what changed and runs the appropriate QA gates:
  - **Go changes** (`internal/`, `cmd/`, `pkg/`, `go.mod`, `go.sum`, `.golangci.yml`, `Makefile`): runs `scripts/qa-go-prepush.sh` (fmt-check, complexity, lint, test, coverage, deadcode, traceability, govulncheck, pattern-gate)
  - **Mobile changes** (`mobile/`, mobile scripts, `ci.yml`): runs `scripts/qa-mobile-prepush.sh` (typecheck, lint, arch, coverage)
- There is no bypass. Fix the failing gate before pushing.

## Commit Attribution

- Before creating a commit, make sure the active AI attribution matches the agent doing the work.
- For Codex/GPT work, set and verify the signature before committing:
  - `git config fenix.ai-agent "chat-gpt5.4"`
  - `git config fenix.ai-agent`
- Do not reuse a previous agent signature such as `claude-sonnet-4-6` for commits authored by Codex/GPT.
- The `prepare-commit-msg` hook appends `AI-Agent` and `AI-Timestamp` trailers based on `AI_AGENTS`, `AI_AGENT`, or `git config fenix.ai-agent`.

## Reporting

- Discrete tasks must be executed one at a time. After closing a task with the required outcome report, stop and wait for explicit user confirmation before starting the next task, even when a plan lists multiple tasks or waves.
- Before starting work on each discrete task, present the task card to the user first, before reading, editing, or running task-specific commands except for minimal inspection needed to identify the next task. The task card must include:
  - `Tarea: <name or ID>`
  - `Resumen: <what will be done in 1-2 sentences>`
  - `Código afectado: <expected files or areas>`
  - `Esfuerzo/razonamiento: Bajo | Medio | Alto - <brief reason>`
  - `Modelo recomendado: <model id>`
  - `Tokens estimado: ~N`
- When closing a task, report the outcome with:
  - `Resultado: <what changed>`
  - `Verificación: <commands run, or why QA was not applicable>`
  - `Archivos afectados: <files changed>`
  - `Esfuerzo/razonamiento: Bajo | Medio | Alto - <forensic note on reasoning effort used>`
  - `Modelo recomendado: <model id>`
  - `Tokens: ~N`
- After the closing report, proactively present the next task card using the same starting-task format, including `Modelo recomendado`, but do not begin that next task until the user explicitly confirms.
- Every substantive report to the user must include:
  - `Esfuerzo/razonamiento: Bajo | Medio | Alto - <forensic note on reasoning effort used>`
  - `Tokens: ~N` (approximate estimate of the response/report size)
- Apply this to progress updates and final summaries.

## Knowledge Management

- Obsidian is the repository knowledge-management layer for project tracking docs, not a product feature.
- Maintain the doc vault proactively. If a task changes architecture, scope, requirements, roadmap, APIs, data model, delivery status, or other project-operating assumptions, update the relevant Obsidian artifacts in the same turn without waiting for an explicit user request.
- Do not treat arbitrary markdown in `docs/` as a task record unless it uses explicit YAML frontmatter.
- When creating a new tracking document for Obsidian, include YAML frontmatter at the top and set `doc_type` explicitly.
- Allowed `doc_type` values are: `task`, `adr`, `summary`, `audit`, `handoff`.
- When a change creates documentary drift, update the source document and also create or update the appropriate vault artifact (`summary`, `audit`, `adr`, or `task`) if the change affects project understanding, governance, or follow-up planning.
- `docs/tasks/` is reserved for real task records only. Do not place summaries, audits, handoffs, or scratch notes there unless the user explicitly asks for it.
- New task records in `docs/tasks/` must include at minimum:
  - `doc_type: task`
  - `id`
  - `title`
  - `status`
  - `phase`
  - `week`
  - `tags`
  - `fr_refs`
  - `uc_refs`
  - `blocked_by`
  - `blocks`
  - `files_affected`
  - `created`
  - `completed`
- ADRs belong in `docs/decisions/`, not in `docs/tasks/`.
- Durable vault artifacts that define shared project reality must remain trackable in Git. This applies by default to canonical plans in `docs/plans/` and ADRs in `docs/decisions/`.
- `docs/tasks/` may contain operational task records that are useful in Obsidian without necessarily being promoted to shared Git history. Do not assume every task record must be committed.
- If a task record becomes the canonical source for coordination, delivery tracking, or cross-session handoff, promote it to a Git-trackable artifact explicitly.
- If ignore rules block a canonical plan or ADR that should be shared, fix the ignore rule or report the conflict immediately.
- If a dashboard or Dataview query is added or updated, filter by `doc_type` instead of assuming folder contents are homogeneous.
- When strategic priorities change, update the relevant dashboards or summary notes so Obsidian continues to reflect current project reality.
