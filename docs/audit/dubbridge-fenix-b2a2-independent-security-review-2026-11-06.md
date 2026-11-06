---
doc_type: audit
id: DB-FENIX-B2A2-SECREVIEW-2026-11-06
title: "B2.A2 — independent security and QA review (S2S principal/credential store)"
status: fail
date: 2026-11-06
reviewer: claude-opus-5-5
branch: feature/int-v2-dubbridge-consumes-fenix
commit_reviewed: 376209e
tags: [dubbridge, s2s, security-review, qa, b2a2]
related:
  - ../tasks/dubbridge-fenix-b2a-s2s-identity.md
  - ../decisions/ADR-DB-FENIX-S2S-PROPOSED.md
  - dubbridge-fenix-b2-cloud-checkpoint-2026-11-05.md
---

# B2.A2 — independent security and QA review

**Verdict: FAIL**

`Peer code review approval: reviewer=claude-opus-5-5; artifact=docs/audit/dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md; status=FAIL`

`RRI: N/A — cloud-agent implementation, DB-FENIX-INT`

The B2.A2 store works correctly on real SQLite (modernc) with Go 1.25.13 for the
basic lifecycle. It fails acceptance because rotation is not atomic, revocation
can be refused (fail-open), the repository QA gate does not pass, and several
packet requirements exist only in documentation.

## Scope

| Item | Value |
|---|---|
| Reviewed code | `internal/domain/auth/service_principal_store.go`, `internal/domain/auth/service_token.go`, their tests |
| Packet compared | `docs/tasks/dubbridge-fenix-b2a-s2s-identity.md` (B2.A2 section), `docs/decisions/ADR-DB-FENIX-S2S-PROPOSED.md`, `docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.{up,down}.sql`, `scripts/int_b2a2_*.py` |
| Adjacent surface checked | `internal/api/middleware/dubbridge_service_auth.go` (A3), `internal/api/routes.go`, `internal/api/middleware/ratelimit.go`, `bff/src/routes/auth.ts` |
| Toolchain | `go1.25.13 darwin/arm64`, `modernc.org/sqlite v1.45.0` |
| Constraints honoured | No code changes, no migrations applied, no real credentials, no GitHub Actions, RRI N/A |

No separate external packet was received; the in-branch packet above was treated as the B2.A2 package.

## Findings

### High

**H1 — Credential rotation is not atomic; concurrent rotation leaves multiple live credentials.**
- Files: `internal/domain/auth/service_principal_store.go:62-88` (`StoreServiceCredential`), `:132-151` (`RevokeServiceCredential`); `docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.up.sql:17-26` (no single-active constraint).
- No rotate primitive exists. Rotation can only be composed as `Store(new)` + `Revoke(old)`: two autocommit statements, no transaction.
- Evidence (probe P2, modernc): 5 concurrent rotators → `revokeOK=1 liveNewTokens=5 activeRows=5`.
- Packet requirement not met: "concurrent rotation, atomic revoke/refresh" (task line 57).
- Fix: add `RotateServiceCredential` in one `BEGIN IMMEDIATE` transaction (revoke all active credentials of the principal + insert the new one), plus a partial unique index `ON service_credential(principal_id) WHERE revoked_at_unix IS NULL`. If a grace overlap is required, make it explicit and bounded.

**H2 — Revocation is refused under clock skew and the credential/principal keeps authenticating (fail-open kill-switch).**
- Files: `internal/domain/auth/service_principal_store.go:138` (`issued_at_unix<=?`), `:161` (`created_at_unix<=?`).
- Evidence (probe P3):
  - credential revoke at `issued-5s` → `service identity not found or inactive`; authenticate at `issued+5s` → `err=<nil>`.
  - principal revoke at `created-5s` → refused; principal keeps authenticating.
- Fix: credential → drop the guard and set `revoked_at_unix = MAX(?, issued_at_unix)` (keeps the `CHECK` at `up.sql:24` satisfied); principal → drop the `created_at_unix` guard.

**H3 — Repository QA gate fails.**
- `scripts/qa-go-prepush.sh` stops at `fmt-check` (26 unformatted files, none from B2.A2).
- Remaining gates run individually:
  - `complexity` (threshold 7) fails on B2.A2: `AuthenticateServiceToken` 14, `StoreServiceCredential` 13, `RegisterServicePrincipal` 11, `RevokeServicePrincipal` 9, `RevokeServiceCredential` 9; also A3 `DubBridgeServiceAuth` 20.
  - `lint`: `gocognit` at `service_principal_store.go:94` and `dubbridge_service_auth.go:51`; `gosec G101` at `service_token.go:16` (false positive on the prefix constant, needs a justified exclusion); discarded `w.Write` result at `dubbridge_service_auth.go:120`.
  - `wrapcheck-gate` fails on non-B2.A2 code in the branch.
  - `govulncheck` RC=3 (see M3).

### Medium

**M1 — Packet items not integrated in code (must not be treated as done).**
- S2S DDL exists only in `docs/contracts/`; probe P5 finds **0** S2S tables after embedded `MigrateUp`. At runtime the store would always return `ErrServiceIdentityUnavailable`.
- Packet fields absent from schema/code (task line 55): credential version, audit linkage, capability allowlist. Register/revoke methods emit no audit events.
- No Go test against real SQLite (task line 57): `service_principal_store_test.go:16-141` uses a fake `database/sql/driver`; `scripts/int_b2a2_store_sqlite_smoke.py` uses Python `sqlite3`, not modernc.
- Fix: Go integration test using `internal/infra/sqlite.NewDB` + the real migration, once the migration receives HITL approval.

**M2 — Regression in existing auth: login rate limit becomes global behind the BFF.**
- Files: `internal/api/routes.go:89` (`ClientIPFromRemoteAddr`), `internal/api/middleware/ratelimit.go:68-71` (no longer reads `X-Real-IP`); the BFF forwards `X-Real-IP` at `bff/src/routes/auth.ts:29-32`.
- Effect: the 5/min login limiter (`routes.go:116`) keys on the BFF IP, so one actor can lock out login for all users.
- Origin: commits `47501cc`, `6130ca4` (outside B2.A2, but on this branch).
- Evidence: source reading only; not runtime-tested.
- Fix: honour `X-Real-IP` only when the TCP peer is the configured BFF address.

**M3 — govulncheck reports 12 vulnerabilities.**
- Standard library in go1.25.13 (`crypto/x509`, `net/http`), fixed only in go1.26.x → toolchain decision for the owner.
- `golang.org/x/net v0.55.0` (bumped on this branch from v0.53.0), fixed in v0.60.0.

## Verified OK

Probes P1 and P4 on modernc via `NewDB` (WAL, `foreign_keys=ON`) + `MigrateUp` + proposal DDL:

- Tenant isolation: each token resolves only its own workspace/org/project; unknown workspace rejected by FK; duplicate digest rejected; scope always read from Fenix, never from the caller.
- Not-yet-valid and exact expiry boundary rejected.
- Normal credential and principal revocation deny immediately.
- No plaintext token in the `.db` or `-wal` file; error strings carry no secret.
- `down.sql` drops only S2S tables; store fails closed afterwards.
- No privilege escalation found. The store methods have no operator authorization of their own but have no callers today; they must sit behind authenticated, audited operator control before A3/A4 wiring.

## Commands executed

All in `/Users/matias/fenix` on `feature/int-v2-dubbridge-consumes-fenix@376209e`.

```bash
git switch --track origin/feature/int-v2-dubbridge-consumes-fenix && git pull --ff-only
GOTOOLCHAIN=go1.25.13 go test ./...                                   # RC=0, 43 packages ok
GOTOOLCHAIN=go1.25.13 go test -race -count=10 ./internal/domain/auth/ \
  ./internal/api/middleware/ ./internal/infra/sqlite/                 # RC=0
GOTOOLCHAIN=go1.25.13 CHANGED_FILES="go.mod go.sum" bash scripts/qa-go-prepush.sh  # RC=2 at fmt-check
GOTOOLCHAIN=go1.25.13 make complexity      # RC=2
GOTOOLCHAIN=go1.25.13 make lint            # RC=2
GOTOOLCHAIN=go1.25.13 make wrapcheck-gate  # RC=2
GOTOOLCHAIN=go1.25.13 make coverage-gate   # RC=0
GOTOOLCHAIN=go1.25.13 make coverage-tdd    # RC=0
GOTOOLCHAIN=go1.25.13 RACE_STABILITY_COUNT=1 make race-stability  # RC=0
deadcode -test ./...                       # no findings
govulncheck ./...                          # RC=3, 12 vulnerabilities
```

## Reproducing the probes

The probe test was never written into the repository; it is injected into the
build with `go test -overlay`. To reproduce:

1. Save the Appendix A source to a file outside the repo, e.g. `/tmp/b2a2probe/zz_review_probe_test.go`.
2. Create the overlay file:
   ```bash
   printf '{"Replace":{"%s":"%s"}}' \
     "$PWD/internal/domain/auth/zz_review_probe_test.go" \
     "/tmp/b2a2probe/zz_review_probe_test.go" > /tmp/b2a2probe/overlay.json
   ```
3. Run from the repo root:
   ```bash
   GOTOOLCHAIN=go1.25.13 go test -overlay=/tmp/b2a2probe/overlay.json \
     -run TestProbe -count=1 -race -v ./internal/domain/auth/
   ```

Expected on `376209e`: P1, P4, P5 PASS; P2 and P3 FAIL (these failures are the findings H1/H2). After a correct fix, P2 must show `liveNewTokens=1` and P3 must show no FINDING lines.

Observed output:

```
--- PASS: TestProbeP1RealSQLiteLifecycle
P2: concurrent rotators=5 revokeOK=1 liveNewTokens=5 activeRows=5
--- FAIL: TestProbeP2ConcurrentRotation
P3a: credential revoke(at=issued-5s) -> service identity not found or inactive ; authenticate(at=issued+5s) -> err=<nil>
P3b: principal revoke(at=created-5s) -> service identity not found or inactive ; authenticate -> err=<nil>
--- FAIL: TestProbeP3RevocationVsClockSkew
--- PASS: TestProbeP4RollbackAndRedaction
P5: S2S tables in embedded runtime migrations = 0
--- PASS: TestProbeP5RuntimeSchemaLacksS2STables
```

## Gate status for B2.A2 closure

| Gate | Status |
|---|---|
| Independent security/peer code review | **FAIL** (this report) |
| `go test ./...` | PASS |
| `-race` on affected packages | PASS |
| `scripts/qa-go-prepush.sh` | **FAIL** |
| Real SQLite (modernc) compatibility | PASS for basic lifecycle (probe only, not a repo test) |
| HITL approval for migration / auth activation | Not requested; still required |

Re-review is required after H1, H2 and H3 are fixed and M1 is either implemented or explicitly re-scoped by the owner.

## Appendix A — probe source

```go
package auth

// REVIEWER PROBE — never written into the repository. Injected only through
// `go test -overlay`. Exercises the B2.A2 store against real modernc SQLite via
// Fenix's NewDB (WAL + foreign_keys=ON), all embedded Fenix migrations and the
// B2.A2 proposal DDL in docs/contracts. Tokens are random test values only.

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	fxsqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func probeDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "probe.db")
	db, err := fxsqlite.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := fxsqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	up, err := os.ReadFile("../../../docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(up)); err != nil {
		t.Fatalf("apply proposal up: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, ws := range []string{"ws-a", "ws-b"} {
		if _, err := db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at) VALUES (?,?,?,?,?)`, ws, ws, ws, now, now); err != nil {
			t.Fatalf("seed workspace: %v", err)
		}
	}
	return db, path
}

func ident(id, ws, org, project string) ServiceIdentity {
	return ServiceIdentity{ServicePrincipal: ServicePrincipal{ID: id, Consumer: "dubbridge", WorkspaceID: ws}, DubBridgeOrgID: org, DubBridgeProjectID: project}
}

// P1: lifecycle + tenant binding + FK + no plaintext on disk, on modernc.
func TestProbeP1RealSQLiteLifecycle(t *testing.T) {
	ctx := context.Background()
	db, path := probeDB(t)
	s := NewServiceCredentialStore(db)
	now := time.Unix(2000000000, 0)
	if err := s.RegisterServicePrincipal(ctx, ident("pa", "ws-a", "org-a", "proj-a"), now); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterServicePrincipal(ctx, ident("pb", "ws-b", "org-b", ""), now); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterServicePrincipal(ctx, ident("px", "ws-missing", "org-x", ""), now); err == nil {
		t.Fatal("FK to workspace not enforced")
	}
	ta, da, _ := GenerateServiceToken()
	tb, dbb, _ := GenerateServiceToken()
	if err := s.StoreServiceCredential(ctx, "ca", "pa", da, now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.StoreServiceCredential(ctx, "cb", "pb", dbb, now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.StoreServiceCredential(ctx, "dup", "pb", da, now, now.Add(time.Hour)); err == nil {
		t.Fatal("duplicate digest accepted")
	}
	ia, err := s.AuthenticateServiceToken(ctx, ta, now.Add(time.Second))
	if err != nil || ia.ID != "pa" || ia.WorkspaceID != "ws-a" || ia.DubBridgeOrgID != "org-a" || ia.DubBridgeProjectID != "proj-a" {
		t.Fatalf("A: %+v %v", ia, err)
	}
	ib, err := s.AuthenticateServiceToken(ctx, tb, now.Add(time.Second))
	if err != nil || ib.ID != "pb" || ib.WorkspaceID != "ws-b" || ib.DubBridgeProjectID != "" {
		t.Fatalf("B: %+v %v", ib, err)
	}
	if _, err := s.AuthenticateServiceToken(ctx, ta, now.Add(-time.Second)); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("not-yet-valid: %v", err)
	}
	if _, err := s.AuthenticateServiceToken(ctx, ta, now.Add(time.Hour)); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("expiry boundary: %v", err)
	}
	if err := s.RevokeServiceCredential(ctx, "ca", now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthenticateServiceToken(ctx, ta, now.Add(3*time.Second)); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("revoked cred: %v", err)
	}
	if err := s.RevokeServicePrincipal(ctx, "pb", now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthenticateServiceToken(ctx, tb, now.Add(3*time.Second)); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("revoked principal: %v", err)
	}
	_, _ = db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	for _, f := range []string{path, path + "-wal"} {
		raw, _ := os.ReadFile(f)
		if bytes.Contains(raw, []byte(ta)) || bytes.Contains(raw, []byte(tb)) {
			t.Fatalf("plaintext token found in %s", f)
		}
	}
	t.Log("P1 PASS: modernc lifecycle, FK, unique digest, tenant/project binding, nbf/expiry, revocation, no plaintext on disk")
}

// P2: rotation = Store(new)+Revoke(old), two independent autocommit statements.
func TestProbeP2ConcurrentRotation(t *testing.T) {
	ctx := context.Background()
	db, _ := probeDB(t)
	s := NewServiceCredentialStore(db)
	now := time.Unix(2000000000, 0)
	if err := s.RegisterServicePrincipal(ctx, ident("pa", "ws-a", "org-a", ""), now); err != nil {
		t.Fatal(err)
	}
	_, d0, _ := GenerateServiceToken()
	if err := s.StoreServiceCredential(ctx, "c0", "pa", d0, now, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	const n = 5
	tokens := make([]string, n)
	var wg sync.WaitGroup
	var mu sync.Mutex
	revokeOK := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tok, d, _ := GenerateServiceToken()
			tokens[i] = tok
			if err := s.StoreServiceCredential(ctx, "c-new-"+string(rune('a'+i)), "pa", d, now.Add(time.Minute), now.Add(24*time.Hour)); err != nil {
				t.Errorf("store %d: %v", i, err)
			}
			if err := s.RevokeServiceCredential(ctx, "c0", now.Add(time.Minute)); err == nil {
				mu.Lock()
				revokeOK++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	live := 0
	for _, tok := range tokens {
		if _, err := s.AuthenticateServiceToken(ctx, tok, now.Add(2*time.Minute)); err == nil {
			live++
		}
	}
	var active int
	_ = db.QueryRow(`SELECT count(*) FROM service_credential WHERE principal_id='pa' AND revoked_at_unix IS NULL`).Scan(&active)
	t.Logf("P2: concurrent rotators=%d revokeOK=%d liveNewTokens=%d activeRows=%d", n, revokeOK, live, active)
	if live > 1 {
		t.Errorf("P2 FINDING: %d credentials simultaneously valid after concurrent rotation", live)
	}
}

// P3: revocation time earlier than issued/created time is refused.
func TestProbeP3RevocationVsClockSkew(t *testing.T) {
	ctx := context.Background()
	db, _ := probeDB(t)
	s := NewServiceCredentialStore(db)
	now := time.Unix(2000000000, 0)
	if err := s.RegisterServicePrincipal(ctx, ident("pa", "ws-a", "org-a", ""), now); err != nil {
		t.Fatal(err)
	}
	tok, d, _ := GenerateServiceToken()
	if err := s.StoreServiceCredential(ctx, "cf", "pa", d, now.Add(5*time.Second), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	rerr := s.RevokeServiceCredential(ctx, "cf", now)
	_, aerr := s.AuthenticateServiceToken(ctx, tok, now.Add(10*time.Second))
	t.Logf("P3a: credential revoke(at=issued-5s) -> %v ; authenticate(at=issued+5s) -> err=%v", rerr, aerr)
	if rerr != nil && aerr == nil {
		t.Errorf("P3a FINDING: credential revocation refused and credential later authenticates")
	}
	if err := s.RegisterServicePrincipal(ctx, ident("pf", "ws-a", "org-a", ""), now.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	tf, df, _ := GenerateServiceToken()
	if err := s.StoreServiceCredential(ctx, "cpf", "pf", df, now.Add(5*time.Second), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	perr := s.RevokeServicePrincipal(ctx, "pf", now)
	_, paerr := s.AuthenticateServiceToken(ctx, tf, now.Add(10*time.Second))
	t.Logf("P3b: principal revoke(at=created-5s) -> %v ; authenticate -> err=%v", perr, paerr)
	if perr != nil && paerr == nil {
		t.Errorf("P3b FINDING: principal kill-switch refused and principal keeps authenticating")
	}
}

// P4: proposal rollback over real Fenix schema; errors carry no secret.
func TestProbeP4RollbackAndRedaction(t *testing.T) {
	ctx := context.Background()
	db, _ := probeDB(t)
	s := NewServiceCredentialStore(db)
	tok, _, _ := GenerateServiceToken()
	_, err := s.AuthenticateServiceToken(ctx, tok, time.Now())
	if err == nil || strings.Contains(err.Error(), tok[len(ServiceTokenPrefix):]) {
		t.Fatalf("redaction: %v", err)
	}
	down, err := os.ReadFile("../../../docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down: %v", err)
	}
	var n, ws int
	_ = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name IN ('service_principal','service_credential')`).Scan(&n)
	_ = db.QueryRow(`SELECT count(*) FROM workspace`).Scan(&ws)
	if n != 0 || ws != 2 {
		t.Fatalf("rollback: tables=%d workspaces=%d", n, ws)
	}
	if _, err := s.AuthenticateServiceToken(ctx, tok, time.Now()); !errors.Is(err, ErrServiceIdentityUnavailable) {
		t.Fatalf("post-rollback must fail closed: %v", err)
	}
	t.Log("P4 PASS: down.sql drops only S2S tables; store fails closed (503 path); no secret in errors")
}

// P5: the real runtime schema (embedded migrations only) has no S2S tables,
// so the committed store cannot authenticate anything without the proposal.
func TestProbeP5RuntimeSchemaLacksS2STables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rt.db")
	db, err := fxsqlite.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := fxsqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name IN ('service_principal','service_credential')`).Scan(&n)
	t.Logf("P5: S2S tables in embedded runtime migrations = %d", n)
}
```
