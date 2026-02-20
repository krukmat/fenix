# Batch 2 (PR1) — API Handlers: Activity + Attachment

## Patrón aplicado

- Patrón principal: **Template Method liviano + Extract Utility**
- Componentes impactados:
  - `internal/api/handlers/helpers.go`
  - `internal/api/handlers/activity.go`
  - `internal/api/handlers/attachment.go`

## Problema previo

Los handlers repetían bloques casi idénticos para:

- obtener `workspace_id`
- decodificar body JSON
- serializar respuestas JSON
- devolver payload paginado `{data, meta}`
- manejar `sql.ErrNoRows` en endpoints `Get`

## Motivación

Reducir duplicación estructural (`dupl`) y estandarizar manejo de errores HTTP sin cambiar contratos funcionales.

## Before

- Cada handler implementaba manualmente el mismo flujo de control.
- Alto solapamiento entre `activity.go` y `attachment.go`.

## After

- Nuevos helpers reutilizables en `helpers.go`:
  - `requireWorkspaceID`
  - `decodeBodyJSON`
  - `writeJSONOr500`
  - `writePaginatedOr500`
  - `handleGetError`
- Refactor de `activity.go` y `attachment.go` para reutilizar esos helpers.

## Riesgos y rollback

- Riesgo: cambiar semántica de errores HTTP por centralización.
- Mitigación: mantener mismos códigos y mensajes previos.
- Rollback: revertir commit del batch (sin cambios de schema ni migraciones).

## Tests

- Ejecutado: `go test ./internal/api/handlers/...`
- Resultado: **OK**.

## Métricas

- Gate `pattern-opportunities-gate` en modo warn:
  - Go `dupl`: **26 → 22**
  - Evidencias de refactor: **2 → 3**
  - TypeScript `jscpd`: se mantiene en **90%** (pendiente siguiente fase)
