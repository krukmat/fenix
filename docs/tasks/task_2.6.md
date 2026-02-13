# Task 2.6: Evidence Pack Builder — Implementation Document

**Status**: ✅ Completed
**Phase**: 2 (Knowledge & Retrieval)
**Duration**: 2 days
**Approved**: 2026-02-12
**Depends on**: Task 2.5 (Hybrid Search) — ✅ Completed

---

## Goals

Implement the Evidence Pack Builder that transforms raw hybrid search results into curated, deduplicated, and confidence-scored evidence packs for the AI layer (Phase 3).

---

## Sub-Tasks

| # | Sub-Task | Status | Files |
|---|----------|--------|-------|
| 2.6.1 | Update EvidencePack model (add counts) | ✅ Done | `internal/domain/knowledge/models.go` |
| 2.6.2 | Add GetKnowledgeItemByID query for freshness check | ✅ Done | `internal/infra/sqlite/queries/knowledge.sql` |
| 2.6.3 | Write tests for EvidencePackService | ✅ Done | `internal/domain/knowledge/evidence_test.go` |
| 2.6.4 | Implement EvidencePackService | ✅ Done | `internal/domain/knowledge/evidence.go` |
| 2.6.5 | Write handler tests | ✅ Done | `internal/api/handlers/knowledge_evidence_test.go` |
| 2.6.6 | Implement evidence handler | ✅ Done | `internal/api/handlers/knowledge_evidence.go` |
| 2.6.7 | Register routes | ✅ Done | `internal/api/routes.go` |
| 2.6.8 | Run all tests and verify | ✅ Done | `go test ./internal/domain/knowledge ./internal/api/handlers ./internal/api` |

---

## Architecture

### EvidencePackService
```
BuildEvidencePack(ctx, query, workspaceID, limit) → (*EvidencePack, error)
  1. HybridSearch(query, workspaceID, 50) → rawResults
  2. Permission filter (stub for Phase 3)
  3. Freshness check (warn if >30 days)
  4. Deduplicate (cosine similarity > 0.95)
  5. Top-K selection (default 10)
  6. Confidence calculation (high/medium/low)
  7. Persist evidence records
  8. Return EvidencePack
```

### Confidence Levels
- **High**: top_score > 0.8
- **Medium**: 0.5 < top_score ≤ 0.8
- **Low**: top_score ≤ 0.5

---

## Files Modified After Completion

| File | Lines Changed | Description |
|------|---------------|-------------|
| `internal/domain/knowledge/models.go` | existing | EvidencePack with `TotalCandidates`, `FilteredCount` already present |
| `internal/infra/sqlite/queries/knowledge.sql` | existing | `GetKnowledgeItemByID` reused for freshness check |
| `internal/domain/knowledge/evidence.go` | new | EvidencePackService implementation |
| `internal/domain/knowledge/evidence_test.go` | updated | Integration/unit tests + helper fixes |
| `internal/api/handlers/knowledge_evidence.go` | new | POST `/api/v1/knowledge/evidence` handler |
| `internal/api/handlers/knowledge_evidence_test.go` | new | Handler integration tests |
| `internal/api/routes.go` | updated | Route registration for evidence endpoint |

---

## API Contract

### POST /api/v1/knowledge/evidence
```json
// Request
{
  "query": "customer pricing inquiry",
  "limit": 10
}

// Response
{
  "data": {
    "sources": [...],
    "confidence": "high",
    "total_candidates": 23,
    "filtered_count": 2,
    "warnings": ["2 items deduplicated", "1 item stale"]
  }
}
```

---

## Test Coverage Targets

- [x] Confidence calculation (high/medium/low thresholds)
- [x] Deduplication (cosine similarity > 0.95)
- [x] Freshness warnings (TTL > 30 days)
- [x] Top-K limiting (default 10, max 50)
- [x] Workspace isolation (security)
- [x] Evidence persistence to DB

---

## Related Tasks

- **Task 2.5**: Hybrid Search (prerequisite) — ✅ Complete
- **Task 3.7**: Agent Orchestrator (consumer) — ⏳ Phase 3
- **Task 3.1**: Policy Engine (permission filtering) — ⏳ Phase 3

---

## Notes

- Deduplication uses in-memory cosine similarity on existing embeddings
- Permission filtering is stubbed; will integrate with Policy Engine in Phase 3
- Evidence records are persisted for audit trail (FR-070)

---

## Source of Truth (Audit)

Para auditoría de Task 2.6, la jerarquía de referencia oficial es:

1. **`docs/tasks/task_2.6.md`** (este documento)
   - Alcance específico, subtareas, estado de cierre, contrato API y cobertura objetivo.
2. **`docs/implementation-plan.md`**
   - Guía de implementación transversal (fases, dependencias y criterios de salida).
3. **`docs/architecture.md`**
   - Restricciones y lineamientos arquitectónicos que gobiernan la implementación.

### Referencias de apoyo

- `docs/tasks/task_2.2_to_2.7_summary.md` (contexto y dependencias de Phase 2)
- `docs/tasks/task_2.5.md` y `docs/tasks/task_2.5_audit.md` (prerrequisito de Hybrid Search)

### Criterio de resolución de conflictos

Si hay discrepancias entre documentos, prevalece este orden:

`task_2.6.md` → `implementation-plan.md` → `architecture.md` → `as-built` (código + tests).
