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
