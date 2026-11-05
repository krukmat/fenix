---
doc_type: audit
id: DB-FENIX-B2A1-CLOUD-2026-11-05
title: "B2.A1 service-token primitives — cloud branch evidence"
status: candidate_peer_and_full_qa_pending
date: 2026-11-05
---

# B2.A1 — cloud implementation checkpoint

**Scope:** DubBridge consumes Fenix; B2.A1 only. No user JWT modifications, no middleware, no SQLite migrations, no token issuance to a real consumer, no workflow/agent and no production route. Current branch: `feature/int-v2-dubbridge-consumes-fenix` in `krukmat/fenix`. RRI is **N/A** for this cloud-agent integration under the scoped owner decision.

## Source uploaded to the assigned integration branch

| Path | Exact Git blob SHA | Evidence |
|---|---|---|
| `internal/domain/auth/service_token.go` | `5d6c85ee791d2b205ee2db0dbdc67fee90d2d690` | isolated 256-bit opaque token generation, strict canonical parser, SHA-256 digest, constant-time digest comparison, nonhuman principal type |
| `internal/domain/auth/service_token_test.go` | `4d26f6f90e6f64c998a310288be57cd38d142111` | positive, malformed, JWT-shaped, noncanonical base64, uniqueness, no bearer leakage, separate principal context |

**Byte-level verification:** the two fetched GitHub source blobs match the locally tested files using `git hash-object`; the malformed-token escaped newline was corrected during publication. No other Go source file was modified.

## Actual local validation (isolated sources, Go 1.23.2, Linux amd64)

The following were run on the exact source files in `/mnt/data/b2a-cloud/internal/domain/auth`, with the GitHub Git object hashes compared afterward:

```sh
GO111MODULE=off go test -count=25 service_token.go service_token_test.go
GO111MODULE=off go test -race -count=5 service_token.go service_token_test.go
GO111MODULE=off go vet service_token.go service_token_test.go
gofmt -l service_token.go service_token_test.go
GO111MODULE=off go test -cover service_token.go service_token_test.go
```

Results: `go test` **PASS** (25 repeats); race detector **PASS** (5 repeats); `go vet` **PASS**; `gofmt` produced no file names; isolated coverage **95.7%**.

**These are not** full repository `go test ./...`, `scripts/qa-go-prepush.sh`, security reviewer `PASS` or production end-to-end tests. The tool runtime cannot clone the GitHub repository directly because DNS to github.com fails; the connected GitHub service was used to commit sources.

## A2 preparation only

A SQLite migration candidate exists in the local staging archive (`drafts/038_service_identity.{up,down}.sql`) and underwent isolated SQLite checks: five negative constraints, revocation and rollback **PASS**. It is **NOT** committed into Fenix's live migration directory and **NOT** run by `MigrateUp`. It still needs architectural/security review, real Go repository implementation and integration tests.

## Gates and decision

- **A1 status:** SOURCE_COMMITTED_TO_ASSIGNED_FEATURE_BRANCH; ISOLATED_GO_TESTS_PASS; **NOT REVIEWED OR FULL_QA_CERTIFIED**.
- **A2 status:** DRAFT_ONLY, no live database/schema changes.
- **A3/A4:** NOT STARTED.
- **Independent peer task-readiness / post-code reviewer:** NOT AVAILABLE here; no fabricated approval. Reviewer must check secret lifecycle, route/JWT separation, revocation and tenant isolation before acceptance.
- **No** main merge, GitHub Actions enablement, real bearer issuance, public S2S API or automatic workflow activation.

Next justified step: execute project-wide Go QA and independent cloud security review of A1; then implement A2 principal store with test-backed revocation, as a separate code task.
