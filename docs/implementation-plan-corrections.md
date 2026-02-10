# Correcciones al Plan de Implementación (MVP P0)

> **Estado**: Propuesto (resultado de auditoría de coherencia)
> **Fecha**: 2026-02-09
> **Documento base**: `docs/implementation-plan.md`
> **Referencia de alineación**: `docs/architecture.md`

---

## 1) Objetivo

Este documento corrige huecos e incoherencias detectados entre arquitectura e implementación, sin cambiar el alcance funcional P0 aprobado, salvo cuando se explicite una despriorización formal.

---

## 2) Resumen de correcciones clave

1. **Auditoría adelantada**: mover `audit_event` y logging mínimo a Fase 1.
2. **Trazabilidad arquitectura → implementación**: añadir matriz de cobertura por entidad y endpoint.
3. **Consistencia multi-tenant en vector search**: ajustar `vec_embedding` para `workspace_id`.
4. **Completar base CRM en Fase 1**: incluir `activity`, `note`, `attachment`, `timeline_event` antes de herramientas/agentes.
5. **Prompt versioning explícito**: definir tareas concretas en P0 (o mover formalmente a P1).
6. **CDC/Reindex explícito**: detallar flujo y criterios de aceptación.
7. **Estructura de proyecto única**: resolver divergencia entre árboles de carpetas.

---

## 3) Cambios concretos al plan por fase

## Fase 1 (Semanas 1–3) — Foundation

### 3.1 Nueva tarea: Auditoría base (mover desde W13)

**Nueva Task 1.7 (W3, 1 día)**

**Acciones**:
- Crear migración `009_audit_base.up.sql` con tabla `audit_event` (append-only).
- Implementar `domain/audit/service.go` mínimo:
  - `Log(ctx, event) error`
  - Inserción inmutable en `audit_event`.
- Conectar logging mínimo desde:
  - Auth (`login`, `register`, denegaciones 401/403)
  - CRUD account/contact/lead/deal/case (create/update/delete)

**Tests**:
- Integración: acción CRUD genera `audit_event`.
- Integración: acceso no autorizado genera evento con `outcome=denied`.

**Motivo**: la arquitectura exige auditabilidad transversal desde fases tempranas.

---

### 3.2 Ampliación de entidades CRM base en Fase 1

**Actualizar Task 1.5** para incluir migraciones/servicios mínimos de:
- `activity`
- `note`
- `attachment`
- `timeline_event`

**Motivo**: son dependencias directas de herramientas, handoff y trazabilidad UC-C1.

---

## Fase 2 (Semanas 4–6) — Knowledge & Retrieval

### 3.3 Corrección de esquema vectorial multi-tenant

**Actualizar Task 2.1**:
- Ajustar virtual table/vector index para incluir `workspace_id` (o garantizar filtro equivalente robusto por join indexado).
- Documentar estrategia exacta de filtrado tenant-safe en ANN.

**Criterio de aceptación**:
- Ninguna búsqueda vectorial retorna documentos de otro `workspace_id`.

---

### 3.4 CDC/Reindex explícito

**Nueva Task 2.7 (W6, 1 día)**

**Acciones**:
- Definir flujo `record.created|updated|deleted` → cola de reindex.
- Implementar consumidor que refresca FTS + embeddings según tipo de cambio.
- Registrar eventos de reindex en auditoría.

**Tests**:
- Integración: update de `case_ticket` provoca reindex del knowledge ligado.
- Integración: índice refleja cambios dentro de SLA interno (ej. <60s en dev).

---

## Fase 3 (Semanas 7–10) — AI Layer

### 3.5 Prompt versioning explícito en P0

**Nueva Task 3.9 (W10, 1 día)**

**Acciones**:
- Crear migración `014_prompt_versioning.up.sql`:
  - `prompt_version`
- Implementar handlers mínimos:
  - `GET/POST /api/v1/admin/prompts`
  - `PUT /api/v1/admin/prompts/{id}/promote`
  - `PUT /api/v1/admin/prompts/{id}/rollback`

**Tests**:
- Integración: promote activa nueva versión.
- Integración: rollback revierte versión activa.

> Si producto decide sacar esto de P0, debe marcarse explícitamente como **de-scope a P1** y reflejarse también en `docs/architecture.md`.

---

## Fase 4 (Semanas 11–13) — Integration & Polish

### 3.6 Ajuste de auditoría en Fase 4

**Actualizar Task 4.5**:
- Ya no crear tabla base (movida a Fase 1).
- Foco en capacidades avanzadas:
  - Query con filtros complejos
  - Export CSV/JSON
  - Suscripción completa al event bus

---

### 3.7 Observabilidad explícita

**Nueva Task 4.8 (W13, 0.5–1 día)**

**Acciones**:
- Endpoint de métricas (mínimo técnico).
- Dashboard básico de latencia/coste por agent run.

**Tests**:
- Integración: endpoint responde métricas básicas.

---

## 4) Matriz mínima de trazabilidad obligatoria

Añadir en `docs/implementation-plan.md` una sección con tabla:

| Componente arquitectura | Estado en plan | Task | Gap |
|---|---|---|---|
| `audit_event` | Parcial | 1.7 + 4.5 | Resuelto con adelanto |
| `prompt_version` | Ausente | 3.9 | Pendiente hasta aplicar |
| `activity/note/attachment/timeline_event` | Parcial | 1.5 (ampliada) | Pendiente |
| CDC reindex | Implícito | 2.7 | Pendiente |
| Observabilidad endpoint/dashboard | Parcial | 4.8 | Pendiente |

Esta tabla debe mantenerse viva durante ejecución.

---

## 5) Criterios de aceptación de coherencia

Se considera corregida la incoherencia cuando:

1. Cada entidad MVP del ERD tiene migración/tarea explícita.
2. Cada endpoint MVP de arquitectura está mapeado a una task o de-scope formal.
3. Los 4 enforcement points de policy están reflejados en backlog ejecutable.
4. La auditoría funciona desde Fase 1 para acciones críticas.
5. Multi-tenancy queda garantizado también en vector retrieval.

---

## 6) Decisión de estructura de carpetas (pendiente de cierre)

Definir y congelar una opción única:

- **Opción A**: arquitectura como fuente de verdad (sin `internal/`, con módulos en `domain/`, `infra/`, `api/`).
- **Opción B**: implementación como fuente de verdad (con `internal/` y `pkg/`).

**Acción obligatoria**: registrar ADR breve (`docs/adr/ADR-001-project-structure.md`) y actualizar ambos documentos para converger.

---

## 7) Impacto en cronograma

- El plan sigue en 13 semanas si se redistribuyen tareas (auditoría adelantada, ajustes pequeños en W6/W10/W13).
- Riesgo de +2 a +4 días si prompt versioning se implementa completo en P0.

---

## 8) Próximo paso recomendado

Actualizar `docs/implementation-plan.md` incorporando estas correcciones y añadir la matriz de trazabilidad como sección fija de control.
