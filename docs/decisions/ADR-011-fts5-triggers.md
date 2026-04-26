---
id: ADR-011
title: "Sync FTS5 index via AFTER INSERT/UPDATE/DELETE triggers on knowledge_item"
date: 2026-01-20
status: accepted
deciders: [matias]
tags: [adr, sqlite, fts5, knowledge, search]
related_tasks: [task_2.1]
related_frs: [FR-090]
---

# ADR-011 — Sync FTS5 index via AFTER INSERT/UPDATE/DELETE triggers on knowledge_item

## Status

`accepted`

## Context

FenixCRM uses SQLite FTS5 (Full-Text Search 5) for BM25 keyword ranking as the first
leg of hybrid search. The FTS5 virtual table (`knowledge_item_fts`) must stay in sync
with the source table (`knowledge_item`) — any drift causes stale or missing search
results.

Two synchronization strategies were considered:

1. **Application-level sync**: After every INSERT/UPDATE/DELETE on `knowledge_item`,
   the application code also writes to `knowledge_item_fts`.
2. **Database trigger sync**: SQLite AFTER triggers automatically maintain the FTS5 table
   whenever the source table changes.

## Decision

Use AFTER triggers defined in the migration SQL:

```sql
-- Sync INSERT
CREATE TRIGGER knowledge_item_fts_insert
AFTER INSERT ON knowledge_item BEGIN
    INSERT INTO knowledge_item_fts(rowid, title, content, source_type)
    VALUES (new.rowid, new.title, new.content, new.source_type);
END;

-- Sync UPDATE
CREATE TRIGGER knowledge_item_fts_update
AFTER UPDATE ON knowledge_item BEGIN
    UPDATE knowledge_item_fts
    SET title = new.title, content = new.content, source_type = new.source_type
    WHERE rowid = new.rowid;
END;

-- Sync DELETE
CREATE TRIGGER knowledge_item_fts_delete
AFTER DELETE ON knowledge_item BEGIN
    DELETE FROM knowledge_item_fts WHERE rowid = old.rowid;
END;
```

## Rationale

- Triggers run atomically within the same SQLite transaction as the source write —
  the FTS5 index is never inconsistent, even on partial failures
- Application code does not need to know about the FTS5 table — ingestion service only
  writes to `knowledge_item`
- BM25 scoring is automatic — FTS5 maintains its own internal statistics
- Zero risk of forgetting to sync in a code path (triggers are database-level guarantees)

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Application-level sync after every write | Risk of drift if any write path forgets to sync; no transaction guarantee |
| Batch reindex on schedule | Stale search results between reindex runs; violates CDC SLA of <60s |
| `contentless_delete` FTS5 mode | More complex to maintain; `rowid` tracking required explicitly |

## Consequences

**Positive:**
- FTS5 index always consistent with source data
- Ingestion service has no FTS5-specific code
- Transactionally safe — no partial states

**Negative / tradeoffs:**
- Triggers add a small write overhead on every `knowledge_item` mutation
  (negligible for expected ingestion volumes)
- Schema changes to `knowledge_item` require updating the trigger definitions in a
  new migration

## References

- `internal/infra/sqlite/migrations/` — migration containing trigger definitions
- `docs/tasks/task_2.1.md` — knowledge tables and FTS5 design
- SQLite FTS5 docs: https://www.sqlite.org/fts5.html
