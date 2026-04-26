---
id: ADR-002
title: "JOIN explícito para vector search multi-tenant (sqlite-vec)"
date: 2026-01-20
status: accepted
deciders: [matias]
tags: [adr, security, vector-search, multi-tenant]
related_tasks: [task_2.1, task_2.5]
related_frs: [FR-090, FR-092]
---

# ADR-002 — JOIN explícito para vector search multi-tenant (sqlite-vec)

## Status

`accepted`

## Context

`sqlite-vec` implementa búsqueda vectorial como una virtual table de SQLite. La query
natural para obtener los K vecinos más cercanos es:

```sql
SELECT id, distance
FROM vec_embedding
WHERE embedding MATCH ?
LIMIT 10
```

Esta query es un data leak: devuelve resultados de **todos los workspaces**. sqlite-vec
no soporta índices multi-columna nativamente, por lo que no es posible filtrar por
`workspace_id` directamente en la virtual table.

## Decision

Usar un JOIN explícito con la tabla `embedding_document` para aplicar el filtro de
workspace **antes** de rankear resultados:

```sql
SELECT e.id, e.chunk_text, e.knowledge_item_id, v.distance
FROM vec_embedding v
JOIN embedding_document e ON v.id = e.id
WHERE e.workspace_id = ?
  AND v.embedding MATCH ?
ORDER BY v.distance
LIMIT 10
```

Este patrón es **obligatorio** en todas las queries de vector search del sistema.

## Rationale

- Seguridad primero: un leak cross-tenant es un fallo crítico, no un bug de performance
- El JOIN sobre `embedding_document.id` usa el índice primario — costo negligible
- El patrón es explícito y auditable en code review
- sqlite-vec evalúa el MATCH después del JOIN, lo que puede ser subóptimo para grandes
  datasets, pero es correcto y seguro para el MVP

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Query directa a vec_embedding sin JOIN | Data leak cross-tenant — inaceptable |
| Row-level security en SQLite | SQLite no tiene RLS nativo |
| Una vec_embedding por workspace | Explosión de tablas; sqlite-vec no soporta nombres dinámicos |
| Filtrar en aplicación post-query | Devuelve datos del servidor de otros tenants antes de filtrar — leak de información |

## Consequences

**Positive:**
- Aislamiento de datos por workspace garantizado a nivel de query
- Patrón auditable y fácil de verificar en code review
- Sin cambios de schema necesarios

**Negative / tradeoffs:**
- Performance subóptima para workspaces con millones de embeddings (sqlite-vec no puede
  pre-filtrar antes del ANN scan)
- Mitigación P1: migrar a PostgreSQL + pgvector que sí soporta filtrado pre-ANN

## References

- sqlite-vec GitHub: no soporta composite indexes en virtual tables
- Patrón documentado en `docs/tasks/task_2.1.md` como mandatory pattern
