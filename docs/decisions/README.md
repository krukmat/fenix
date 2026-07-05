# ADR Index — fenix

All Architecture Decision Records for this repository. Each row links to the ADR file; status must match the `## Status` prose section inside the file.

| ADR | Title | Status |
|---|---|---|
| [ADR-001](ADR-001-uuid-full-slug.md) | Usar UUID completo como sufijo de slug (sin truncado) | accepted |
| [ADR-002](ADR-002-sqlite-vec-multitenant.md) | JOIN explícito para vector search multi-tenant (sqlite-vec) | accepted |
| [ADR-003](ADR-003-testmain-jwt-secret.md) | TestMain + os.Setenv para JWT_SECRET en tests (no t.Setenv) | accepted |
| [ADR-004](ADR-004-tparallel-tsetenv.md) | Avoid t.Setenv in parallel tests — use os.Setenv in TestMain | accepted |
| [ADR-005](ADR-005-goconst-shadow-handlers.md) | Centralize string constants in helpers.go and avoid variable shadowing in handlers | accepted |
| [ADR-006](ADR-006-complexity-gate.md) | Enforce cyclomatic complexity threshold of 7 via gocyclo | accepted |
| [ADR-007](ADR-007-pkgauth-alias.md) | Use pkgauth alias when importing pkg/auth from internal/domain/auth | accepted |
| [ADR-008](ADR-008-route-structure.md) | Route structure — public vs protected, JWT claims replace X-Workspace-ID header | accepted |
| [ADR-009](ADR-009-bff-thin-proxy.md) | BFF is a thin proxy — zero business logic, zero database access | accepted |
| [ADR-010](ADR-010-sqlite-pure-go.md) | Use modernc.org/sqlite (pure-Go) to guarantee CGO_ENABLED=0 builds | accepted |
| [ADR-011](ADR-011-fts5-triggers.md) | Sync FTS5 index via AFTER INSERT/UPDATE/DELETE triggers on knowledge_item | accepted |
| [ADR-012](ADR-012-chunking-strategy.md) | Chunking strategy: 512-token chunks with 50-token overlap, whitespace tokenizer | accepted |
| [ADR-013](ADR-013-embedding-async.md) | Embedding generation is async: status=pending → event bus → embedder → retry | accepted |
| [ADR-014](ADR-014-hybrid-search-rrf.md) | Hybrid search uses Reciprocal Rank Fusion (RRF, k=60) over BM25 + vector results | accepted |
| [ADR-015](ADR-015-evidence-deduplication.md) | Evidence pack deduplication using cosine similarity threshold (>0.95) and confidence tiers | accepted |
| [ADR-016](ADR-016-cdc-reindex-sla.md) | CDC reindex SLA: changes visible in search within 60 seconds | accepted |
| [ADR-017](ADR-017-quality-gates.md) | Quality gates: gocognit ≤10, maintidx ≥20, gocyclo ≤7, applied to production code only | accepted |
| [ADR-018](ADR-018-bdd-pipeline-strategy.md) | BDD pipeline: Go stack implemented (godog), BFF/Mobile as placeholders; UC→FR→TST traceability via cmd/frtrace | accepted |
| [ADR-019](ADR-019-product-category-governed-ai-layer.md) | Product category shift: governed AI layer for customer operations, not broad CRM replacement | accepted |
| [ADR-020](ADR-020-cost-governance-runtime-concern.md) | Cost governance is a runtime concern and requires first-class usage metering | accepted |
| [ADR-021](ADR-021-integration-first-context-strategy.md) | Integration-first context strategy: native CRM tables plus external system provenance | accepted |
| [ADR-022](ADR-022-mobile-deprioritized-for-wedge.md) | Mobile is supported but not a universal wedge gate | accepted |
| [ADR-023](ADR-023-approve-role-validation.md) | APPROVE role validation: deferred to runtime, workspace-scoped, with abstention on unknown role | accepted |
| [ADR-024](ADR-024-defer-type-enum-action-connector.md) | Defer TYPE, ENUM, ACTION, CONNECTOR grammar — no implementation until runtime contracts exist | accepted |
| [ADR-025](ADR-025-bff-unified-client-gateway.md) | BFF is the unified client gateway — web, mobile, and future clients all route through BFF | accepted |
| [ADR-026](ADR-026-web-builder-stack.md) | Web builder surface uses HTMX + Express (BFF) over a separate React SPA | accepted |
| [ADR-027](ADR-027-design-md-agent-visual-context.md) | DESIGN.md as agent visual context contract | accepted |
| [ADR-028](ADR-028-snapshot-approval-dual-seed.md) | Dual approval seed for snapshot runner approve/reject coverage | accepted |
| [ADR-029](ADR-029-bff-admin-shell.md) | BFF admin shell: HTMX read-only surface at /bff/admin/* over existing Go governance endpoints | accepted |
| [ADR-030](ADR-030-gemma-local-adjudication.md) | Gemma local reviewer multi-pass contract and context-isolated adjudication (D14) | accepted |
| [ADR-031](ADR-031-support-approval-trigger-contract.md) | Support approval trigger contract: approval gates sensitive case mutation, handoff remains a human-routing fallback | accepted |
| [ADR-032](ADR-032-workspace-bootstrap-defaults.md) | Workspace bootstrap defaults: a freshly registered workspace has no usable role or pipeline | proposed |
| [ADR-034](ADR-034-mobile-primary-ux-surface.md) | Mobile is the primary UX surface for the Verifiable Trust wedge | accepted |
| [ADR-035](ADR-035-peer-review-gate-unconditional-block.md) | Peer review and code review gates are unconditionally blocking, no waiver or BLOCKED-terminal exception | accepted |
| [ADR-100](ADR-100-agentic-blackboard-architecture.md) | Agentic Blackboard Architecture | accepted |
| [ADR-101](ADR-101-relationship-memory-engine.md) | Relationship Memory Engine | accepted |
| [ADR-102](ADR-102-deterministic-agent-evaluation.md) | Deterministic Agent Evaluation Framework | accepted |
