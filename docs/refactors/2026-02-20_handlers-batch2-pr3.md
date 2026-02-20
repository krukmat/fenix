# Batch 2 (PR3) — API Handlers: Account + Lead

## Patrón aplicado

- Patrón principal: **Template Method liviano + Extract Utility**
- Componentes impactados:
  - `internal/api/handlers/account.go`
  - `internal/api/handlers/lead.go`

## Problema previo

Se mantenía duplicación amplia entre handlers de `account` y `lead` en:

- obtención/validación de `workspaceId`
- decode JSON del body
- encode JSON de respuesta
- respuesta paginada
- manejo de `sql.ErrNoRows`

## Motivación

Se eligió **Template Method liviano + Extract Utility** porque el flujo HTTP era prácticamente idéntico entre handlers (validación de contexto, parseo, serialización y manejo de errores) y eso permitía consolidar sin introducir una jerarquía compleja.

Alternativas descartadas:

- **Base handler por herencia/composición grande**: agregaba demasiada abstracción para el tamaño actual del módulo.
- **Mantener duplicación con reglas de lint**: no resolvía deuda técnica ni riesgo de divergencia de comportamiento.

## Before

- `account.go` y `lead.go` repetían bloques de:
  - `workspace_id` obligatorio
  - `json.NewDecoder(...).Decode(...)`
  - `json.NewEncoder(...).Encode(...)`
  - respuesta paginada `{data, meta}`
  - ramas `sql.ErrNoRows` + error interno
- Paths relevantes:
  - `internal/api/handlers/account.go`
  - `internal/api/handlers/lead.go`
  - (previo a extracción) bloques locales de manejo de errores/JSON en cada handler

## After

Se unificó el flujo con helpers compartidos:

- `requireWorkspaceID`
- `decodeBodyJSON`
- `writeJSONOr500`
- `writePaginatedOr500`
- `handleGetError`

## Tests

- Ejecutado: `go test ./internal/api/handlers/...`
- Resultado: **OK**

## Métricas

- `pattern-opportunities-gate` (warn):
  - Go `dupl`: **19 → 14**
  - TypeScript `jscpd`: **90%** (sin cambios en este lote)

## Riesgos y rollback

- Riesgo: diferencias de status/headers al centralizar serialización.
- Mitigación: se mantuvieron códigos HTTP y mensajes.
- Rollback: revertir commit de este lote (sin impacto en schema/migraciones).
