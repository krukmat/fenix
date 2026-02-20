# Batch 2 (PR2) — API Handlers: Deal + Note

## Patrón aplicado

- Patrón principal: **Template Method liviano + Extract Utility**
- Componentes impactados:
  - `internal/api/handlers/deal.go`
  - `internal/api/handlers/note.go`

## Problema previo

Persistía duplicación de flujo CRUD en handlers (`workspace`, decode, encode, not found, paginación), especialmente entre `deal` y `note`.

## Motivación

Continuar el lote de deduplicación en handlers reusando los helpers creados en PR1, reduciendo `dupl` sin alterar contratos HTTP.

## Before

- `deal.go` y `note.go` repetían validación de workspace, decode JSON y respuestas.
- Manejo de `sql.ErrNoRows` repetido por endpoint.

## After

- Ambos handlers migrados a helpers comunes:
  - `requireWorkspaceID`
  - `decodeBodyJSON`
  - `writeJSONOr500`
  - `writePaginatedOr500`
  - `handleGetError`

## Riesgos y rollback

- Riesgo: inconsistencias en status codes/mensajes por centralización.
- Mitigación: conservar mensajes y códigos originales.
- Rollback: revertir commit de PR2 (sin migraciones).

## Tests

- Ejecutado: `go test ./internal/api/handlers/...`
- Resultado: **OK**

## Métricas

- `pattern-opportunities-gate` (warn):
  - Go `dupl`: **22 → 19**
  - TypeScript `jscpd`: **90%** (sin cambios en este PR)
