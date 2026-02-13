# Task 2.7: CDC & Auto-Reindex — Planning Document

**Status**: 🟡 Planned
**Phase**: 2 (Knowledge & Retrieval)
**Duration**: 1 day
**Depends on**: Task 2.5 (Hybrid Search) ✅, Task 2.6 (Evidence Pack) ✅

---

## Objective

Implement an explicit CDC (Change Data Capture) flow so CRM record changes are reflected in search indexes automatically, with freshness SLA tracking.

Target outcomes:
- Subscribe to `record.created`, `record.updated`, `record.deleted`
- Trigger reindex actions over knowledge/search data
- Expose manual reindex endpoint for operational fallback
- Track freshness SLA from event time to index refresh time

---

## Scope (MVP for Task 2.7)

1. **Event subscription**
   - Consume domain events from event bus for record changes.
   - Event payload contract:
     ```json
     {
       "entity_type": "case_ticket|note|activity|attachment|...",
       "entity_id": "uuid",
       "workspace_id": "uuid",
       "change_type": "created|updated|deleted",
       "occurred_at": "timestamp"
     }
     ```

2. **Reindex orchestrator**
   - Map event → impacted knowledge items/chunks.
   - Re-run indexing path depending on change type:
     - `created/updated`: refresh chunk + embedding + searchable projection
     - `deleted`: de-index/remove stale searchable records

3. **Operational API fallback**
   - Add manual endpoint:
     - `POST /api/v1/knowledge/reindex`
   - Supports scoped reindex by workspace and/or entity.

4. **SLA instrumentation**
   - Measure latency: `event.occurred_at` → `index_refreshed_at`.
   - Acceptance target:
     - Dev: `<60s`
     - Future prod optimization target: `<10s`

---

## Implementation Plan (Actionable)

### 2.7.1 Domain contracts
- Add/confirm typed CDC event model in `internal/domain/knowledge` or shared event package.
- Define idempotency key strategy (`workspace_id + entity_type + entity_id + change_type + occurred_at`).

### 2.7.2 CDC consumer
- Implement consumer in `internal/domain/knowledge` (or `internal/infra/eventbus` integration layer).
- Subscribe to `record.created|updated|deleted`.
- Validate payload + workspace isolation before processing.

### 2.7.3 Reindex service
- Create `ReindexService` with methods:
  - `HandleRecordCreated(...)`
  - `HandleRecordUpdated(...)`
  - `HandleRecordDeleted(...)`
- Reuse existing ingest/embed/search infrastructure to avoid duplicate logic.

### 2.7.4 API handler
- Implement `POST /api/v1/knowledge/reindex` handler in `internal/api/handlers`.
- Register route in `internal/api/routes.go`.
- Return processing summary (`received`, `reindexed`, `skipped`, `errors`).

### 2.7.5 Tests
- Unit tests: routing logic for change types + idempotency handling.
- Integration tests:
  - emit `record.updated` → verify searchable content refresh
  - emit `record.deleted` → verify de-index behavior
  - workspace isolation hard-check
- Handler tests for manual reindex endpoint.

### 2.7.6 SLA checks
- Add metric/log fields with timestamps for event and refresh.
- Add integration assertion for `<60s` in local/dev execution.

---

## Acceptance Criteria

- [ ] Consumer receives and processes `record.created|updated|deleted`
- [ ] Reindex path refreshes search visibility after update/create
- [ ] Delete path removes stale searchable artifacts
- [ ] Manual endpoint `POST /api/v1/knowledge/reindex` works
- [ ] Multi-tenant boundaries preserved (`workspace_id` enforced)
- [ ] SLA telemetry available and tested (`event`→`refresh`)
- [ ] Tests green in affected packages

---

## Risks & Mitigations

- **Duplicate event processing**
  - Mitigation: idempotency key + dedupe guard in service.
- **Event ordering issues**
  - Mitigation: compare timestamps/version and ignore stale events.
- **High reindex cost on burst updates**
  - Mitigation: batch/coalesce by entity key (future optimization if needed).

---

## Source of Truth (Audit)

1. `docs/tasks/task_2.7.md` (this file)
2. `docs/implementation-plan.md`
3. `docs/architecture.md`

Support references:
- `docs/implementation-plan-corrections.md`
- `docs/implementation-plan-summary.md`
- `docs/CORRECTIONS-APPLIED.md`
