# Handoff Técnico — Task 1.5

**Proyecto:** FenixCRM  
**Fuente de verdad:** `docs/implementation-plan.md`  
**Task:** **1.5 — Lead, Deal, Case + Supporting Entities**  
**Objetivo:** implementar el bloque CRM restante end-to-end para desbloquear tools y flujos de agentes.

---

## 1) Contexto y estado actual

- ✅ Task 1.1 completada
- ✅ Task 1.2 completada técnicamente
- ✅ Task 1.3 completada (Account)
- ✅ Task 1.4 completada (Contact)
- ⏳ Siguiente bloque: **Task 1.5**

### Estado real validado (2026-02-10, actualizado 12:21 CET)

- ✅ Migraciones de Task 1.5 **ya creadas**.
- ⚠️ El orden numérico final quedó así (válido funcionalmente):
  - `004_crm_pipelines.up.sql`
  - `005_crm_leads.up.sql`
  - `006_crm_deals.up.sql`
  - `007_crm_cases.up.sql`
  - `008_crm_supporting.up.sql`
- ✅ También existen `*.down.sql` para 004–008.
- ✅ Handlers/rutas añadidos para: lead, deal, case, pipeline+stages, activity, note, attachment, timeline.
- ✅ Timeline automático integrado en servicios core (create/update/delete según entidad).
- ✅ `go test ./...` en verde tras integración.

---

## 2) Alcance de Task 1.5

Implementar de punta a punta:

1. `lead`
2. `deal`
3. `case_ticket`
4. `pipeline`
5. `pipeline_stage`
6. `activity`
7. `note`
8. `attachment`
9. `timeline_event`

Incluye: migraciones, queries sqlc, servicios de dominio, handlers HTTP, rutas y tests.

---

## 3) Entregables obligatorios

## A) Migraciones

Crear en `internal/infra/sqlite/migrations/`:

- `004_crm_pipelines.up.sql` ✅
- `005_crm_leads.up.sql` ✅
- `006_crm_deals.up.sql` ✅
- `007_crm_cases.up.sql` ✅
- `008_crm_supporting.up.sql` ✅ (activity, note, attachment, timeline_event)

### Criterios DB

- PK UUID v7 (`TEXT`)
- multi-tenancy con `workspace_id`
- soft delete donde aplique (`deleted_at`)
- índices por `workspace_id`, `owner_id`, `deleted_at`, fechas
- FKs coherentes con ERD

---

## B) SQL queries + sqlc

Crear en `internal/infra/sqlite/queries/`:

- `lead.sql`
- `deal.sql`
- `case.sql`
- `pipeline.sql`
- `activity.sql`
- `note.sql`
- `attachment.sql`
- `timeline.sql`

### Patrón mínimo por entidad

- Create
- GetByID
- ListByWorkspace (paginado)
- Update
- SoftDelete
- Count

Luego ejecutar:

```bash
sqlc generate
```

---

## C) Servicios de dominio

Crear en `internal/domain/crm/`:

- `lead.go`
- `deal.go`
- `case.go`
- `pipeline.go`
- `activity.go`
- `note.go`
- `attachment.go`
- `timeline.go`

### Convenciones a mantener

- filtro por `workspace_id` en todas las operaciones
- `nullString()` para campos nullable
- fechas RFC3339
- soft delete
- patrón `Service + sqlc Querier` (igual que account/contact)

---

## D) API / Handlers / Rutas

Crear handlers en `internal/api/handlers/` y registrar en `internal/api/routes.go`.

### Mínimos esperados

- CRUD para lead/deal/case
- pipeline + stages
- activity/note/attachment
- endpoint de timeline por entidad (si entra en esta task)

---

## E) Timeline automático

Cada create/update/delete de entidades core debe generar `timeline_event` mínimo con:

- `workspace_id`
- `entity_type`
- `entity_id`
- `event_type`
- `old_value/new_value` (si aplica)
- `created_at`

Puede implementarse directo en servicios (sin event bus completo todavía).

---

## 4) Tests requeridos (Definition of Done técnico)

## Unit / Integration / API tests

- CRUD por cada entidad nueva
- soft delete exclusion en list/get
- aislamiento multi-tenant (`workspace_id`)
- FKs críticas:
  - deal → account/stage/pipeline
  - case/deal ownership
- pipeline stage transitions
- activity polymorphic (`entity_type` + `entity_id`)
- timeline auto-generado
- attachment con `storage_path` válido

### Comandos de validación

```bash
go test ./internal/domain/crm ./internal/api/handlers ./internal/infra/sqlite ./internal/api
go test ./...
```

---

## 5) Criterio de aceptación final

Task 1.5 se considera cerrada cuando:

1. migraciones aplican sin error,
2. `sqlc generate` genera código limpio,
3. servicios + handlers + rutas funcionan,
4. tests pasan en paquetes objetivo y global,
5. `docs/implementation-plan.md` queda actualizado con **Task 1.5 = ✅ COMPLETED** + evidencias.

---

## 6) Checklist operativo (para ejecución)

- [x] Crear migraciones 004–008
- [x] Crear queries SQL de las 8 entidades
- [x] Ejecutar `sqlc generate`
- [x] Implementar servicios de dominio
- [x] Implementar handlers HTTP
- [x] Registrar rutas en router
- [x] Implementar timeline automático
- [x] Completar tests unit/integration/API
- [x] Ejecutar `go test` focalizado + global
- [x] Marcar Task 1.5 como completada en documentación
