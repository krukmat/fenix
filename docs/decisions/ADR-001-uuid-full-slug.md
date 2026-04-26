---
id: ADR-001
title: "Usar UUID completo como sufijo de slug (sin truncado)"
date: 2026-01-15
status: accepted
deciders: [matias]
tags: [adr, database, testing]
related_tasks: [task_1.3]
related_frs: [FR-001, FR-002]
---

# ADR-001 — Usar UUID completo como sufijo de slug (sin truncado)

## Status

`accepted`

## Context

Los registros CRM (Account, Contact, Lead, Deal, Case) tienen un campo `slug` generado
automáticamente para URLs amigables. La implementación inicial usaba los primeros 8
caracteres del UUID v7 como sufijo:

```go
slug = fmt.Sprintf("%s-%s", baseName, uuid[:8])
```

UUID v7 codifica el timestamp en milisegundos en los primeros bits. Dos registros
creados en el mismo milisegundo comparten el mismo prefijo de 8 caracteres.

En tests, especialmente con `t.Parallel()`, múltiples goroutines crean registros
simultáneamente. Esto causaba colisiones en la constraint `UNIQUE(slug)` de SQLite,
haciendo que los tests fallaran de forma no determinista (flaky tests).

## Decision

Usar el UUID completo (36 caracteres) como sufijo del slug, sin truncado:

```go
slug = fmt.Sprintf("%s-%s", baseName, id) // id es el UUID v7 completo
```

## Rationale

- UUID v7 garantiza unicidad global — truncarlo elimina esa garantía
- El slug no necesita ser "bonito" para ser funcional; es para URLs internas
- Elimina completamente la posibilidad de colisión, incluso bajo carga paralela
- Sin costo adicional en performance ni storage significativo (36 vs 8 chars)

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| UUID[:8] truncado | Colisiones garantizadas en tests paralelos y posibles en prod bajo carga |
| UUID[:16] truncado | Reduce colisiones pero no las elimina; falso sentido de seguridad |
| Slug basado solo en nombre + counter | Requiere query adicional para calcular el counter; race condition posible |
| Nanoid / ULID como sufijo | Dependencia externa innecesaria cuando UUID v7 ya está disponible |

## Consequences

**Positive:**
- Tests completamente deterministas, sin flakiness por colisión de slugs
- Unicidad garantizada en producción sin lógica de retry
- Sin dependencias externas adicionales

**Negative / tradeoffs:**
- Slugs menos legibles: `acme-corp-018e1234-5678-7abc-def0-123456789abc`
- Si en el futuro se exponen slugs en URLs públicas, será necesario un mecanismo de "friendly slug" separado

## References

- Go test runner parallelism: `t.Parallel()` crea goroutines concurrentes en el mismo proceso
- UUID v7 spec (RFC 9562): timestamp en bits 0-47, garantiza orden cronológico pero no unicidad en el mismo ms sin los bits aleatorios subsiguientes
