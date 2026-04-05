---
title: FenixCRM — Project Status Dashboard
tags: [dashboard]
---

# FenixCRM — Project Status Dashboard

> Este archivo es un dashboard vivo. Requiere el plugin **Dataview** en Obsidian para renderizar las queries.
> Las queries leen el frontmatter de los archivos en `docs/tasks/`.

---

## Tasks en progreso

```dataview
TABLE title, phase, week, blocked_by, files_affected
FROM "tasks"
WHERE status = "in_progress"
SORT phase ASC, week ASC
```

---

## Tasks pendientes por fase

```dataview
TABLE title, phase, week, fr_refs, blocked_by
FROM "tasks"
WHERE status = "pending"
SORT phase ASC, week ASC
```

---

## Tasks completados

```dataview
TABLE title, phase, week, completed, fr_refs
FROM "tasks"
WHERE status = "completed"
SORT completed DESC
```

---

## Resumen por fase

```dataview
TABLE rows.file.name AS tasks, length(rows) AS total
FROM "tasks"
WHERE status != null
GROUP BY phase
SORT phase ASC
```

---

## Tasks bloqueados

```dataview
TABLE title, phase, blocked_by
FROM "tasks"
WHERE blocked_by != null AND length(blocked_by) > 0
SORT phase ASC
```

---

## Architecture Decision Records

```dataview
TABLE title, date, status, related_tasks
FROM "decisions"
SORT date DESC
```

---

## Referencias rápidas

- [[implementation-plan|Implementation Plan]]
- [[architecture|Architecture]]
- [[requirements|Requirements]]
- [FR & UC Status Dashboard](dashboards/fr-uc-status.md)
- [templates/task-template](templates/task-template.md)
- [templates/adr-template](templates/adr-template.md)
