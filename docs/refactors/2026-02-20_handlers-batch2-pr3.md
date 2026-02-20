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

## Riesgo y rollback

- Riesgo: diferencias de status/headers al centralizar serialización.
- Mitigación: se mantuvieron códigos HTTP y mensajes.
- Rollback: revertir commit de este lote (sin impacto en schema/migraciones).
