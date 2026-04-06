---
title: FenixCRM — Project Status Dashboard
last_updated: 2026-04-06
tags: [dashboard]
---

# FenixCRM — Project Status Dashboard

> Este archivo es un dashboard vivo. Requiere el plugin **Dataview** en Obsidian para renderizar las queries.
> Las queries leen el frontmatter del vault y filtran por `doc_type` cuando aplica.

---

## Tasks en progreso

```dataview
TABLE title, phase, week, blocked_by, files_affected
FROM "tasks"
WHERE doc_type = "task" AND status = "in_progress"
SORT phase ASC, week ASC
```

---

## Tasks pendientes por fase

```dataview
TABLE title, phase, week, fr_refs, blocked_by
FROM "tasks"
WHERE doc_type = "task" AND status = "pending"
SORT phase ASC, week ASC
```

---

## Tasks completados

```dataview
TABLE title, phase, week, completed, fr_refs
FROM "tasks"
WHERE doc_type = "task" AND status = "completed"
SORT completed DESC
```

---

## Resumen por fase

```dataview
TABLE rows.file.name AS tasks, length(rows) AS total
FROM "tasks"
WHERE doc_type = "task" AND status != null
GROUP BY phase
SORT phase ASC
```

---

## Tasks bloqueados

```dataview
TABLE title, phase, blocked_by
FROM "tasks"
WHERE doc_type = "task" AND blocked_by != null AND length(blocked_by) > 0
SORT phase ASC
```

---

## Architecture Decision Records

```dataview
TABLE title, date, status, related_tasks
FROM "decisions"
WHERE doc_type = "adr"
SORT date DESC
```

---

## Strategic Summaries and Audits

```dataview
TABLE doc_type, title, date, status
FROM ""
WHERE doc_type = "summary" OR doc_type = "audit"
SORT date DESC
```

---

## Referencias rápidas

- [[plans/fenixcrm_strategic_repositioning_implementation_plan|Strategic Repositioning Implementation Plan]]
- [[implementation-plan|Implementation Plan]]
- [[architecture|Architecture]]
- [[requirements|Requirements]]
- [[plans/fenixcrm_strategic_repositioning_spec|Strategic Repositioning Spec]]
- [FR & UC Status Dashboard](dashboards/fr-uc-status.md)
- [[strategic-realignment-summary|Strategic Realignment Summary]]
- [templates/task-template](templates/task-template.md)
- [templates/adr-template](templates/adr-template.md)
