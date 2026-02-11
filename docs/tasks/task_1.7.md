pues # Task 1.7: Audit Logging Foundation

> **Status**: 🟡 In Progress (base de auditoría implementada; integraciones API/auth pendientes)
> **Priority**: P0 (Phase 1, Week 3)
> **Duration**: 1 day
> **Approach**: TDD (Test-First)
> **Based on**: `docs/implementation-plan.md` Section 3 — Phase 1

---

## Overview

Implementation of immutable audit logging foundation for FenixCRM. This task was **moved from Week 13 (Phase 4) to Week 3 (Phase 1)** as a critical correction — the architecture mandates immutable audit trails from inception.

**Rationale**: Without audit from Phase 1, all Phase 2-4 actions (CRM changes, agent runs, tool executions) would not be recorded retrospectively. This is non-negotiable for governed systems.

### As-built Status (2026-02-10)

**Completado**
- ✅ Migraciones `010_audit_base.up/down.sql`
- ✅ Dominio `internal/domain/audit` (`types.go`, `service.go`, `service_test.go`)
- ✅ Queries/sqlc para `audit_event`

**Pendiente**
- ❌ `internal/api/middleware/audit.go` + tests
- ❌ Integración del middleware en `internal/api/routes.go`
- ❌ Integración de eventos de auditoría en auth (login/logout/failures)
- ❌ Integración de auditoría en CRUD handlers principales
- ❌ Cierre de criterios E2E de Task 1.7

---

## Goals

1. Create `audit_event` table (append-only, immutable)
2. Implement `AuditService` with `Log()` method
3. Create audit middleware for automatic request logging
4. Integrate with auth events (login/logout/failures)
5. Integrate with CRM entity changes (create/update/delete)
6. Log authorization denials (401/403)

---

## Files to Create/Modify

### New Files

| File | Description | Lines (est.) |
|------|-------------|--------------|
| `internal/infra/sqlite/migrations/010_audit_base.up.sql` | Migration: audit_event table + indexes | ~45 |
| `internal/infra/sqlite/migrations/010_audit_base.down.sql` | Rollback migration | ~10 |
| `internal/infra/sqlite/queries/audit.sql` | SQL queries for sqlc | ~35 |
| `internal/domain/audit/service.go` | AuditService implementation | ~120 |
| `internal/domain/audit/types.go` | Audit types (AuditEvent, ActorType, Outcome) | ~50 |
| `internal/domain/audit/service_test.go` | Integration tests | ~200 |
| `internal/api/middleware/audit.go` | Audit logging middleware | ~80 |
| `internal/api/middleware/audit_test.go` | Middleware tests | ~120 |

### Modified Files

| File | Changes |
|------|---------|
| `internal/api/routes.go` | Add audit middleware to protected routes |
| `internal/domain/auth/service.go` | Add audit logging for login/logout |
| `internal/api/handlers/account.go` | Add audit logging for CRUD operations |
| `internal/api/handlers/contact.go` | Add audit logging for CRUD operations |
| `internal/api/handlers/lead.go` | Add audit logging for CRUD operations |
| `internal/api/handlers/deal.go` | Add audit logging for CRUD operations |
| `internal/api/handlers/case.go` | Add audit logging for CRUD operations |
| `internal/server/server.go` | Inject AuditService into dependencies |
| `docs/implementation-plan.md` | Mark task as completed |

---

## Database Schema

### Migration 010_audit_base.up.sql

```sql
-- audit_event table: append-only, immutable audit trail
CREATE TABLE audit_event (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL CHECK(actor_type IN ('user', 'agent', 'system')),
    action TEXT NOT NULL,                    -- e.g., 'login', 'create_account', 'delete_deal'
    entity_type TEXT,                        -- e.g., 'account', 'contact', 'case' (nullable for auth actions)
    entity_id TEXT,                          -- UUID of affected entity (nullable)
    details JSON,                            -- Flexible JSON: { old_value, new_value, changes, metadata }
    permissions_checked JSON,                -- Permissions verified: [{ permission, result }]
    outcome TEXT NOT NULL CHECK(outcome IN ('success', 'denied', 'error')),
    trace_id TEXT,                           -- For distributed tracing correlation
    ip_address TEXT,                         -- Client IP
    user_agent TEXT,                         -- Client user agent
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (workspace_id) REFERENCES workspace(id)
);

-- Indexes for common query patterns
CREATE INDEX idx_audit_workspace ON audit_event(workspace_id);
CREATE INDEX idx_audit_actor ON audit_event(actor_id);
CREATE INDEX idx_audit_entity ON audit_event(entity_type, entity_id);
CREATE INDEX idx_audit_created ON audit_event(created_at);
CREATE INDEX idx_audit_outcome ON audit_event(outcome);
CREATE INDEX idx_audit_action ON audit_event(action);
CREATE INDEX idx_audit_trace ON audit_event(trace_id);
```

### Migration 010_audit_base.down.sql

```sql
DROP INDEX IF EXISTS idx_audit_trace;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_outcome;
DROP INDEX IF EXISTS idx_audit_created;
DROP INDEX IF EXISTS idx_audit_entity;
DROP INDEX IF EXISTS idx_audit_actor;
DROP INDEX IF EXISTS idx_audit_workspace;
DROP TABLE IF EXISTS audit_event;
```

---

## SQL Queries (sqlc)

### internal/infra/sqlite/queries/audit.sql

```sql
-- name: CreateAuditEvent :exec
INSERT INTO audit_event (
    id, workspace_id, actor_id, actor_type, action,
    entity_type, entity_id, details, permissions_checked,
    outcome, trace_id, ip_address, user_agent, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAuditEventByID :one
SELECT * FROM audit_event WHERE id = ? LIMIT 1;

-- name: ListAuditEventsByWorkspace :many
SELECT * FROM audit_event
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAuditEventsByActor :many
SELECT * FROM audit_event
WHERE actor_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAuditEventsByEntity :many
SELECT * FROM audit_event
WHERE entity_type = ? AND entity_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountAuditEventsByWorkspace :one
SELECT COUNT(*) FROM audit_event WHERE workspace_id = ?;
```

---

## Domain Types

### internal/domain/audit/types.go

```go
package audit

import (
    "encoding/json"
    "time"
)

// ActorType represents the type of actor performing an action
type ActorType string

const (
    ActorTypeUser   ActorType = "user"
    ActorTypeAgent  ActorType = "agent"
    ActorTypeSystem ActorType = "system"
)

// Outcome represents the result of an audited action
type Outcome string

const (
    OutcomeSuccess Outcome = "success"
    OutcomeDenied  Outcome = "denied"
    OutcomeError   Outcome = "error"
)

// AuditEvent represents a single audit log entry
// This is immutable - once created, it should never be modified
type AuditEvent struct {
    ID                 string          `json:"id"`
    WorkspaceID        string          `json:"workspace_id"`
    ActorID            string          `json:"actor_id"`
    ActorType          ActorType       `json:"actor_type"`
    Action             string          `json:"action"`
    EntityType         *string         `json:"entity_type,omitempty"`
    EntityID           *string         `json:"entity_id,omitempty"`
    Details            json.RawMessage `json:"details,omitempty"`
    PermissionsChecked json.RawMessage `json:"permissions_checked,omitempty"`
    Outcome            Outcome         `json:"outcome"`
    TraceID            *string         `json:"trace_id,omitempty"`
    IPAddress          *string         `json:"ip_address,omitempty"`
    UserAgent          *string         `json:"user_agent,omitempty"`
    CreatedAt          time.Time       `json:"created_at"`
}

// EventDetails captures the specifics of an audited action
type EventDetails struct {
    OldValue  interface{} `json:"old_value,omitempty"`
    NewValue  interface{} `json:"new_value,omitempty"`
    Changes   []Change    `json:"changes,omitempty"`
    Metadata  interface{} `json:"metadata,omitempty"`
}

// Change represents a single field change
type Change struct {
    Field     string      `json:"field"`
    OldValue  interface{} `json:"old_value,omitempty"`
    NewValue  interface{} `json:"new_value,omitempty"`
}

// PermissionCheck represents a permission verification
type PermissionCheck struct {
    Permission string `json:"permission"`
    Granted    bool   `json:"granted"`
    Reason     string `json:"reason,omitempty"`
}
```

---

## Service Implementation

### internal/domain/audit/service.go

Key methods:

```go
// AuditService provides audit logging capabilities
type AuditService struct {
    db      *sql.DB
    querier sqlcgen.Querier
}

// NewAuditService creates a new audit service
func NewAuditService(db *sql.DB) *AuditService

// Log creates a new audit event (append-only, immutable)
// This is the ONLY way to create audit events - no updates, no deletes
func (s *AuditService) Log(ctx context.Context, event *AuditEvent) error

// LogWithDetails helper for common case with structured details
func (s *AuditService) LogWithDetails(
    ctx context.Context,
    workspaceID, actorID string,
    actorType ActorType,
    action string,
    entityType, entityID *string,
    details *EventDetails,
    outcome Outcome,
) error

// GetByID retrieves a single audit event by ID
func (s *AuditService) GetByID(ctx context.Context, id string) (*AuditEvent, error)

// ListByWorkspace retrieves audit events for a workspace (with pagination)
func (s *AuditService) ListByWorkspace(
    ctx context.Context,
    workspaceID string,
    limit, offset int,
) ([]*AuditEvent, int, error)

// ListByEntity retrieves audit events for a specific entity
func (s *AuditService) ListByEntity(
    ctx context.Context,
    entityType, entityID string,
    limit int,
) ([]*AuditEvent, error)
```

---

## Middleware Implementation

### internal/api/middleware/audit.go

```go
// AuditMiddleware logs all HTTP requests to the audit trail
// Should be placed AFTER AuthMiddleware so JWT claims are available
func AuditMiddleware(auditService *audit.AuditService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract context
            workspaceID := GetWorkspaceIDFromContext(r.Context())
            userID := GetUserIDFromContext(r.Context())

            // Capture response status
            wrapped := &responseRecorder{ResponseWriter: w, statusCode: 200}

            start := time.Now()
            next.ServeHTTP(wrapped, r)
            duration := time.Since(start)

            // Determine outcome from status code
            outcome := determineOutcome(wrapped.statusCode)

            // Log to audit trail (async/non-blocking if possible)
            auditService.LogWithDetails(
                r.Context(),
                workspaceID,
                userID,
                audit.ActorTypeUser,
                fmt.Sprintf("%s %s", r.Method, r.URL.Path),
                nil, nil, // entity type/id from route params if available
                &audit.EventDetails{
                    Metadata: map[string]interface{}{
                        "method":       r.Method,
                        "path":         r.URL.Path,
                        "status_code":  wrapped.statusCode,
                        "duration_ms":  duration.Milliseconds(),
                        "ip_address":   r.RemoteAddr,
                        "user_agent":   r.UserAgent(),
                    },
                },
                outcome,
            )
        })
    }
}

func determineOutcome(statusCode int) audit.Outcome {
    switch {
    case statusCode >= 200 && statusCode < 300:
        return audit.OutcomeSuccess
    case statusCode == 401 || statusCode == 403:
        return audit.OutcomeDenied
    default:
        return audit.OutcomeError
    }
}
```

---

## Integration Points

### 1. Auth Events (internal/domain/auth/service.go)

```go
// After successful login
auditService.LogWithDetails(ctx, workspaceID, userID, audit.ActorTypeUser,
    "login", nil, nil, nil, audit.OutcomeSuccess)

// After failed login (with reason)
auditService.LogWithDetails(ctx, workspaceID, "unknown", audit.ActorTypeUser,
    "login", nil, nil,
    &audit.EventDetails{Metadata: map[string]interface{}{"reason": "invalid_credentials"}},
    audit.OutcomeError)
```

### 2. CRM Entity Changes (e.g., account handler)

```go
// After successful create
auditService.LogWithDetails(ctx, workspaceID, userID, audit.ActorTypeUser,
    "create_account", strPtr("account"), &account.ID,
    &audit.EventDetails{NewValue: account}, audit.OutcomeSuccess)

// After successful update (with changes)
auditService.LogWithDetails(ctx, workspaceID, userID, audit.ActorTypeUser,
    "update_account", strPtr("account"), &account.ID,
    &audit.EventDetails{OldValue: oldAccount, NewValue: account, Changes: changes},
    audit.OutcomeSuccess)

// After delete
auditService.LogWithDetails(ctx, workspaceID, userID, audit.ActorTypeUser,
    "delete_account", strPtr("account"), &accountID,
    &audit.EventDetails{OldValue: account}, audit.OutcomeSuccess)
```

---

## Test Plan

### Integration Tests (internal/domain/audit/service_test.go)

| Test | Description |
|------|-------------|
| `TestCreateAuditEvent_Success` | Log event → verify stored correctly |
| `TestGetAuditEventByID` | Create → GetByID → fields match |
| `TestListAuditEventsByWorkspace` | Create multiple → list returns paginated |
| `TestListAuditEventsByActor` | Filter by actor_id works |
| `TestListAuditEventsByEntity` | Filter by entity_type + entity_id |
| `TestAuditEventImmutability` | Verify no UPDATE/DELETE methods exist |
| `TestAuditTenantIsolation` | Workspace A cannot see Workspace B events |

### Middleware Tests (internal/api/middleware/audit_test.go)

| Test | Description |
|------|-------------|
| `TestAuditMiddleware_LogsRequest` | Request → audit event created |
| `TestAuditMiddleware_OutcomeSuccess` | 200 status → outcome=success |
| `TestAuditMiddleware_OutcomeDenied` | 403 status → outcome=denied |
| `TestAuditMiddleware_OutcomeError` | 500 status → outcome=error |
| `TestAuditMiddleware_IncludesContext` | WorkspaceID + UserID from context |

### Handler Integration Tests

| Test | Description |
|------|-------------|
| `TestLogin_Success_AuditEvent` | Login → audit event created |
| `TestLogin_Failure_AuditEvent` | Failed login → audit event with outcome=error |
| `TestCreateAccount_AuditEvent` | Create → audit with new_value |
| `TestUpdateAccount_AuditEvent` | Update → audit with changes |
| `TestDeleteAccount_AuditEvent` | Delete → audit with old_value |
| `TestUnauthorized_Access_AuditEvent` | 403 → audit with outcome=denied |

---

## Build Sequence

Order of implementation (TDD approach):

1. **Write tests first**: `service_test.go` skeleton with failing tests
2. **Create migration**: `010_audit_base.up.sql` + `.down.sql`
3. **Create queries**: `audit.sql` for sqlc
4. **Run**: `make sqlc-generate` → generates `sqlcgen/audit.go`
5. **Implement types**: `types.go` with domain types
6. **Implement service**: `service.go` with Log method
7. **Run tests**: Verify service tests pass
8. **Implement middleware**: `middleware/audit.go`
9. **Run tests**: Verify middleware tests pass
10. **Integrate**: Add middleware to routes, add logging to handlers
11. **Run full suite**: `go test ./...` must pass
12. **Update docs**: Mark task complete in implementation-plan.md

---

## Requirements Coverage

| Requirement | Coverage |
|-------------|----------|
| FR-070 (Audit trail) | Foundation — all actions logged |
| NFR-031 (Traceability) | From Phase 1 — immutable log |
| FR-060 (RBAC) | Permission checks logged |
| NFR-030 (Observability) | Request logging active |

---

## Exit Criteria

- [x] Migration 010 applied successfully
- [x] `AuditService.Log()` creates events in DB
- [ ] Middleware logs all HTTP requests
- [ ] Auth events (login/logout) logged
- [ ] CRM CRUD operations logged
- [ ] 401/403 denials logged with outcome=denied
- [x] Integration tests del dominio audit pasan
- [ ] Tenant isolation verified (cross-workspace leak test end-to-end)
- [x] No UPDATE/DELETE methods on audit events (immutability check)
- [ ] `go test ./...` passes con cobertura de integraciones Task 1.7 completas

---

## Notes & Risks

### Design Decisions

1. **Immutable by convention**: No DB-level enforcement (SQLite limitation), but service layer has no Update/Delete methods.

2. **JSON fields for flexibility**: `details` and `permissions_checked` are JSON to accommodate varying schemas without schema migrations.

3. **Async logging consideration**: For high throughput, consider async logging with channel + background worker. For MVP, synchronous is acceptable.

4. **PII handling**: IP addresses and user agents are logged — ensure this complies with privacy requirements.

### Performance Considerations

- Audit table can grow large quickly
- Indexes cover common query patterns (workspace, actor, entity, date)
- Future: Partitioning by date (if migrating to PostgreSQL)

### Migration Conflict

Migration 009 is already used by `009_auth_index`. Use **010** for audit.

---

## Related Documents

- `docs/implementation-plan.md` — Section 3, Phase 1, Task 1.7
- `docs/architecture.md` — Section 6 (Audit & Telemetry)
- `internal/infra/sqlite/migrations/` — Existing migrations
- `internal/domain/` — Domain service pattern reference

---

## Task Log (Update as you progress)

| Date | Action | Status |
|------|--------|--------|
| 2026-02-10 | Task document created | ✅ |
| 2026-02-10 | Base implementada (migraciones + dominio + sqlc) | ✅ |
| 2026-02-10 | Estado corregido a In Progress y checklist actualizado | ✅ |
| | | |
| | | |
| | | |

---

**Next Action**: Start with test file `internal/domain/audit/service_test.go` — write failing tests for `Log()` and `GetByID()`.
