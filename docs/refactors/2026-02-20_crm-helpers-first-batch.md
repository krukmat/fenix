# Primer lote CRM — reducción de duplicación en servicios

## Patrón aplicado

- Patrón principal: **Extract Method / Utility Module**
- Componentes impactados:
  - `internal/domain/crm/service_helpers.go`
  - `internal/domain/crm/activity.go`
  - `internal/domain/crm/attachment.go`
  - `internal/domain/crm/case.go`
  - `internal/domain/crm/deal.go`
  - `internal/domain/crm/lead.go`
  - `internal/domain/crm/note.go`

## Problema previo

Había duplicación transversal en servicios CRM para:

- `time.Now().UTC().Format(time.RFC3339)`
- parseo de fechas RFC3339 (incluyendo punteros opcionales)
- mapeo repetitivo de slices (`for i := range rows { ... }`)

Esto elevaba el ruido de `dupl` y aumentaba costo de mantenimiento.

## Motivación

Se eligió extracción de utilidades compartidas por ser el cambio más seguro y de bajo riesgo para un primer lote de remediación, sin alterar contratos públicos de servicios.

## Before

- Múltiples servicios CRM repetían las mismas construcciones de fecha y loops de mapeo.
- Conversión de `row` a entidad repetía parseo manual de `time.Parse`.

## After

- Se incorporó `service_helpers.go` con:
  - `nowRFC3339()`
  - `parseRFC3339Time()`
  - `parseOptionalRFC3339()`
  - `mapRows()`
- Se migraron servicios CRM seleccionados a helpers compartidos.

## Riesgos y rollback

- Riesgo: parseo silencioso de fechas inválidas (comportamiento ya existente al ignorar error de parse).
- Rollback: revertir commit del lote; no hay migraciones ni cambios de schema.

## Tests

- Ejecutado: `go test ./internal/domain/crm/...`
- Resultado: OK en paquete CRM.

## Métricas

- `dupl` (gate global): **32 -> 26** coincidencias (mejora tras primer lote).
- Próximo foco identificado por `dupl`: duplicación en `internal/api/handlers/*`.
