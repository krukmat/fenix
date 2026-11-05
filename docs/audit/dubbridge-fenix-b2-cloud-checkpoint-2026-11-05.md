---
doc_type: audit
id: DB-FENIX-B2-CLOUD-2026-11-05
title: "B2.A–B2.C isolated implementation and tests — cloud branch checkpoint"
status: isolated_pass_full_qa_and_peer_pending
date: 2026-11-05
---

# DB-FENIX-INT — cloud work checkpoint

**Branch:** `feature/int-v2-dubbridge-consumes-fenix` in `krukmat/fenix`; consumer DubBridge remains the caller. **RRI: N/A** as directed for cloud-agent development. Real workflows and agents are still out of scope until usage-driven B6.

## Source implemented, but not activated

| Task | Source and scope | Status |
|---|---|---|
| B2.A1 | `internal/domain/auth/service_token.go` + existing tests: 256-bit random opaque credentials, strict canonical parser, constant-time digest verification | SOURCE COMMITTED, scoped tests PASS, peer PENDING |
| B2.A2 | `internal/domain/auth/service_principal_store.go` + tests: DB-backed principal scope, digest-only credential state, expiry, revocation and fail-closed lookup | SOURCE COMMITTED, scoped tests PASS; **proposed SQLite schema remains outside embedded runtime migrations**; rotation and full SQLite-Go integration pending |
| B2.A3 | `internal/api/middleware/dubbridge_service_auth.go` + tests: isolated bearer validation, trusted identity context, exact source org/project comparison, request body limit 64 KiB and safe HTTP errors | SOURCE COMMITTED, scoped httptest PASS; **not attached to routes.go**, no live HTTP service |
| B2.A4 | Run full Fenix project QA and independent security code review; operator provisioning/runbook | NOT DONE |
| B2.B1–B3 | `internal/domain/integration/s2s_capability_registry.go` + tests: empty initial registry, default-deny, exact principal/workspace/key/schema-version binding, disable and concurrent reads | SOURCE COMMITTED, scoped race tests PASS; **not wired or persisted**, no model/workflow dispatch |
| B2.B4 | Audit, durable operator registration and effective ToolRegistry policy review | PENDING |
| B2.C | proposed operation SQL in `docs/contracts/dubbridge-fenix-b2c-operation-proposal.{up,down}.sql` + isolated Python SQLite test | PROPOSAL ONLY, 5 constraint negatives, principal-scoped replay, indeterminate crash path and rollback PASS |
| B2.D–B2.E / B3–B5 | Real provider/consumer transport, durable operation Go code, end-to-end and closure | NOT STARTED |

## Reproduced local test evidence

An isolated Go 1.23.2 module copied **exact GitHub blob content** (matching SHA for each new Go file). The real Fenix module requires Go 1.25.13 and many dependencies not resolvable here, so **these are not full-repository test receipts**:

```bash
GOTOOLCHAIN=local go test -count=50 ./...
GOTOOLCHAIN=local go test -race -count=8 ./...
GOTOOLCHAIN=local go vet ./...
gofmt -l internal/domain/auth/service_principal_store*.go \
  internal/api/middleware/dubbridge_service_auth*.go \
  internal/domain/integration/s2s_capability_registry*.go
python scripts/int_b2a2_store_sqlite_smoke.py
python scripts/int_b2c_operation_sqlite_smoke.py
```

Results: **PASS** for 50 repeats, 8 race repeats, go vet, gofmt, real SQLite Go-query smoke, operation idempotency proposal, SQLite rollback and five invalid input cases per SQL test. Isolated coverage: auth **84.5%**, middleware **96.3%**, registry **93.3%**. Baseline A1 separately covered **95.7%** when tested alone.

## Exact current source blob hashes

- A2 store: `8aea9d640f90c36b04705df5f9ebad5612266d8a`; A2 Go tests: `540843b6d0c1aaabc2ea10d1c4414eba87586877`.
- A3 middleware: `36be9f6dee6671d8ad748c857cf3592b440f0615`; middleware tests: `f483bf6a5cc7d4fd38b2e642098357070180c6ec`.
- B2.B registry: `7d6b18e827bb0279b46e179935d0744c6b52bf28`; registry tests: `c62187e76a7626d8da6b1da27cbb42e0ba6a4c4e`.
- SQLite Go-query smoke: `3a9fb0724a78619e4916f3252f340ee3d3d5ea0e`; B2.C proposed SQLite operation smoke: `064337cfb09d66f854d09028b7a58b164a9195a8`.

## Invariants / blockers

- No API route is activated, no human JWT code has been changed, no live service tokens have been issued, no workspace access is inferred from client source JSON.
- The **SQLite principal and operation DDLs reside in docs/contracts and are not registered with embedded MigrateUp**. Tests ran only on disposable in-memory databases.
- **No independent qualified security code review, full Fenix Go1.25 QA, real SQLite-Go integration, operator provisioning/rotation or real two-service HTTP E2E** is demonstrated.
- Existing Fenix policies require explicit HITL permission for auth, SQLite migrations and live changes; do not interpret the scoped RRI exception as peer-review exemption.
- Both repos remain on their assigned feature branches; no merge to `main` or GitHub Actions reactivation.

**Decision:** SAFE TO CONTINUE design/isolated cloud work. **NOT ACCEPTED for live service identity or production integration.** Next critical path: independent security review + Fenix Go1.25 full QA, then approved migration and handler/router wiring, followed by B2.C Go persistence and B3 client integration.
