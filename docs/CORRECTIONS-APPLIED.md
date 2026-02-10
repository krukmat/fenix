# Correcciones Aplicadas al Plan de Implementación

> **Fecha**: 2026-02-09
> **Estado**: ✅ Completado
> **Documentos actualizados**: `docs/implementation-plan.md`

---

## Resumen Ejecutivo

Se realizó una auditoría de coherencia entre `docs/architecture.md` y `docs/implementation-plan.md`, identificando **7 gaps críticos** que podrían causar bloqueos en la ejecución. Todos los hallazgos fueron corregidos sin impacto en el cronograma de 13 semanas.

---

## Correcciones Críticas Aplicadas

### 1. ✅ Auditoría Adelantada (Task 1.7)

**Problema**: Auditoría estaba en Week 13, pero arquitectura exige trazabilidad inmutable desde inicio.

**Corrección**:
- Nueva Task 1.7 en Week 3 (Phase 1)
- Crea tabla `audit_event` (append-only, immutable)
- Implementa `domain/audit/service.go` básico
- Conecta logging a auth + CRM CRUD
- Middleware de auditoría captura todas las requests

**Impacto**: Phase 2-4 ahora tienen auditoría funcional desde el inicio. No más "retrospective audit impossible".

**Rationale**: Sistemas governed requieren audit-first, no audit-last. Esta es arquitectura foundational, no feature final.

---

### 2. ✅ Entidades CRM Completas en Phase 1 (Task 1.5 Expanded)

**Problema**: `activity`, `note`, `attachment`, `timeline_event` estaban parcialmente definidos. Herramientas en Phase 3 (`create_task` → `activity`, `send_reply` → `note`) tenían dependencias rotas.

**Corrección**:
- Task 1.5 expandida de 3 → 4 días
- Nueva migración `008_crm_supporting.up.sql`
- Servicios completos: `activity.go`, `note.go`, `attachment.go`, `timeline.go`
- Timeline auto-recording conectado al event bus
- Handlers CRUD para todas las entidades

**Impacto**: Phase 3 tool implementation ahora tiene todas las tablas requeridas. No hay dependencias faltantes.

---

### 3. ✅ Multi-Tenancy en Vector Search (Task 2.1 Security Fix)

**Problema**: sqlite-vec NO soporta índices multi-columna. Sin filtro explícito de `workspace_id`, queries vectoriales podrían "saltar" entre tenants.

**Corrección**:
- Documentado query pattern obligatorio:
  ```sql
  -- CORRECTO (tenant-safe):
  SELECT e.id, e.chunk_text, e.distance
  FROM vec_embedding v
  JOIN embedding_document e ON v.id = e.id
  WHERE e.workspace_id = ?  -- ← CRÍTICO
  AND v.embedding MATCH ?
  ORDER BY v.distance LIMIT ?;
  ```
- Test de seguridad explícito: workspace A nunca ve docs de workspace B
- sqlc queries generados con filtro obligatorio

**Impacto**: **P0 Security Blocker** resuelto. Multi-tenancy garantizado en retrieval vectorial.

**Rationale**: Sin este fix, un tenant podría acceder a embeddings de otro via ANN search. Esto es data leak crítico.

---

### 4. ✅ CDC/Auto-Reindex Explícito (Task 2.7 New)

**Problema**: Arquitectura asume "cambios visibles en <60s" pero no había mecanismo de reindex explícito. CRM updates no reflejados en búsquedas.

**Corrección**:
- Nueva Task 2.7 en Week 6 (Phase 2)
- Implementa CDC: subscribe a `record.created|updated|deleted`
- Reindex consumer actualiza FTS5 + re-embeds si contenido cambió
- Handler manual: `POST /api/v1/knowledge/reindex`
- SLA tracking: event timestamp → refresh timestamp (<60s target)

**Impacto**: Knowledge freshness garantizado. Updates en CRM → visibles en Copilot/Agent searches.

---

### 5. ✅ Prompt Versioning Explícito (Task 3.9 New)

**Problema**: Arquitectura muestra `agent_definition.active_prompt_version_id` FK pero no había task para implementarlo. Sin versioning, no hay rollback ni eval-gating.

**Corrección**:
- Nueva Task 3.9 en Week 10 (Phase 3)
- Migración `015_prompt_versioning.up.sql`
- Handlers: create, promote, rollback
- Integración con Agent Orchestrator (carga prompt activo)
- Tests: versioning independiente por agent

**Impacto**: Capability mínima para change management de prompts. Rollback en 1 click.

**Decisión Pendiente**: ¿Mantener en P0 o mover a P1? Recomendación: **Mantener** (1 día, arquitectura lo requiere).

---

### 6. ✅ Auditoría Avanzada en Phase 4 (Task 4.5 Updated)

**Problema**: Task 4.5 original creaba tabla audit desde cero, pero ahora existe desde Phase 1.

**Corrección**:
- Task 4.5 actualizada: NO crea tabla
- Foco en features avanzadas:
  - Query con filtros complejos (date range, actor, entity, outcome)
  - Full-text search en campo `details` (JSON)
  - Export CSV/JSON con streaming
  - Suscripción completa al event bus (agent, tool, policy events)

**Impacto**: Phase 4 complementa auditoría básica con capabilities de query/export enterprise.

---

### 7. ✅ Observabilidad Explícita (Task 4.8 New)

**Problema**: Arquitectura requiere "metrics endpoint, agent run dashboard" pero no había task explícita.

**Corrección**:
- Nueva Task 4.8 en Week 13 (Phase 4)
- Endpoint `/api/v1/metrics` (Prometheus-compatible)
- Métricas expuestas:
  - HTTP requests (latency, status codes)
  - Agent runs (count, duration, cost, tokens por agent_type)
  - Tool calls (count, outcome por tool_name)
  - Evidence retrieval (latency)
- Health check: `/api/v1/health` (DB, LLM, event bus)
- Dashboard básico: stats últimas 24h

**Impacto**: Observability mínima para MVP. Grafana dashboards quedan para P1.

---

## Nuevos Artefactos Creados

### 1. Traceability Matrix (Section 2)

Tabla living document que mapea:
- Architecture Component (ERD entity)
- Implementation Status (✅ Completed | ⚠️ Partial | ❌ Pending | 🔵 Out of scope)
- Phase + Task
- Notes (correcciones aplicadas)

**Propósito**: Garantizar cobertura 100% de componentes arquitectónicos. Detectar gaps temprano.

**Uso**: Actualizar a medida que se completan tasks. Al final de cada phase, validar que todos los componentes de esa phase están ✅.

---

### 2. ADR-001: Project Structure (Section 11)

**Decisión**: Option B (con `internal/`)

**Estructura aprobada**:
```
fenixcrm/
├── cmd/fenixcrm/main.go
├── internal/                # Private application code
│   ├── domain/             # Business logic
│   ├── infra/              # Infrastructure adapters
│   ├── api/                # HTTP handlers
├── pkg/                     # Public shared libraries
├── tests/
├── docs/
├── Makefile, go.mod, sqlc.yaml, etc.
```

**Rationale**:
- FenixCRM es aplicación, no librería → `internal/` previene imports externos
- Go convention: `internal/` = encapsulación explícita
- Import paths: `github.com/yourorg/fenixcrm/internal/domain/crm`

**Acción pendiente**: Actualizar `docs/architecture.md` Appendix para converger con esta estructura.

---

## Impacto en Cronograma

| Phase | Cambios | Días Agregados | Días Redistribuidos | Duración Final |
|-------|---------|----------------|---------------------|----------------|
| Phase 1 | Task 1.5 expanded (3→4d), Task 1.7 new (1d) | +2 días | Task 1.6 reduced (2→1d) | 3 semanas (sin cambio) |
| Phase 2 | Task 2.7 new (1d) | +1 día | Task 2.1 adjusted | 3 semanas (sin cambio) |
| Phase 3 | Task 3.9 new (1d) | +1 día | Task 3.8 adjusted | 4 semanas (sin cambio) |
| Phase 4 | Task 4.8 new (1d), Task 4.5 reduced (2→1.5d) | +0.5 días | Task 4.7 adjusted | 3 semanas (sin cambio) |

**Total**: **13 semanas** (sin cambio en deadline)

**Estrategia**: Redistribución interna de días. Tareas menos críticas ajustadas para acomodar nuevas tareas críticas.

---

## Criterios de Salida Actualizados

### Phase 1 (Foundation)
- ✅ **NEW**: Audit logging functional
- ✅ **NEW**: Timeline events auto-generated
- ✅ **NEW**: Multi-tenancy verified (workspace_id isolation)

### Phase 2 (Knowledge)
- ✅ **NEW**: Multi-tenant vector search verified
- ✅ **NEW**: CDC/Auto-reindex working (<60s SLA)

### Phase 3 (AI Layer)
- ✅ **NEW**: Prompt versioning functional (create, promote, rollback)

### Phase 4 (Integration)
- ✅ **NEW**: Observability endpoints functional (/metrics, /health, dashboard)

---

## Verificación de Coherencia

### ✅ Checklist Final

- [x] Cada entidad MVP del ERD tiene migración/task explícita
- [x] Cada endpoint MVP de arquitectura está mapeado a una task o de-scope formal
- [x] Los 4 enforcement points de policy están reflejados en backlog ejecutable
- [x] La auditoría funciona desde Fase 1 para acciones críticas
- [x] Multi-tenancy garantizado en vector retrieval (JOIN obligatorio)
- [x] CDC/Reindex explícito con SLA target
- [x] Prompt versioning con promote/rollback capability
- [x] Observabilidad con métricas + health checks

---

## Decisiones Pendientes

### 1. Prompt Versioning en P0 vs P1

**Opción A**: Mantener Task 3.9 en P0
- **Pros**: Arquitectura lo requiere (FK existe), rollback capability crítica, 1 día de esfuerzo
- **Cons**: +1 día en Phase 3

**Opción B**: Mover a P1
- **Pros**: Reduce scope P0
- **Cons**: Requiere actualizar `docs/architecture.md` ERD (remover `active_prompt_version_id` FK), sin rollback en MVP

**Decisión**: ✅ **Opción A APROBADA (2026-02-09)** — Prompt versioning permanece en P0. Task 3.9 en Week 10 confirmada.

---

### 2. Estructura de Carpetas

**Decisión tomada**: Option B (con `internal/`)

**Acción requerida**:
- Actualizar `docs/architecture.md` Appendix (líneas 1274-1316) para reflejar estructura con `internal/`
- Congelar decisión en ADR-001

---

## Próximos Pasos

1. **Review de correcciones** con el equipo (este documento)
2. **Decisión final** sobre prompt versioning (mantener o diferir)
3. **Actualizar `docs/architecture.md`** para convergir estructura de carpetas
4. **Comenzar Phase 1, Task 1.1** (Project Setup)

---

## Archivos Afectados

- ✅ `docs/implementation-plan.md` — Correcciones aplicadas
- ✅ `docs/CORRECTIONS-APPLIED.md` — Este documento (resumen ejecutivo)
- ⏳ `docs/architecture.md` — Pendiente: actualizar Appendix con estructura `internal/`
- ⏳ `CLAUDE.md` — Pendiente: agregar referencia a plan de implementación

---

**Fin del Documento de Correcciones**
