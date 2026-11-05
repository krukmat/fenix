---
doc_type: task
id: DB-FENIX-B2A1-READINESS
title: "B2.A1 — isolated service-token primitives: task readiness and RRI"
status: blocked_peer_readiness
date: 2026-10-31
criticality: critical
criticality_basis: "Authentication secret validation and a future privilege boundary; no runtime integration allowed"
---

# B2.A1 — service-token primitives (no runtime authorization)

## Authoritative cloud checkpoint — 2026-11-05

**Status: CODE_COMMITTED_ISOLATED_TESTS_PASS / PEER_AND_FULL_QA_PENDING.** This supersedes any older statements below that A1 source files have not been written or committed.

The user requested development on the already assigned feature branches. Fenix `feature/int-v2-dubbridge-consumes-fenix` now contains:
- `internal/domain/auth/service_token.go` — Git blob `5d6c85ee791d2b205ee2db0dbdc67fee90d2d690`.
- `internal/domain/auth/service_token_test.go` — Git blob `4d26f6f90e6f64c998a310288be57cd38d142111`.

Both blob hashes match local files actually tested via `GO111MODULE=off go test -count=25`, `go test -race -count=5`, `go vet` and `gofmt` (all PASS), with 95.7% isolated statement coverage. Evidence: [cloud B2.A1 audit](../audit/dubbridge-fenix-b2a1-cloud-2026-11-05.md).

**Not closed:** Fenix full-repository QA, independent peer/security code review and actual service authorization are NOT verified. No real S2S credential, SQLite migration or HTTP route is active. RRI remains N/A for this cloud integration.



> **Owner process correction (2026-10-31):** Because DB-FENIX-INT is executed via cloud coding agents, **RRI: N/A**. The earlier `RRI 67` estimate and its script commands below are **HISTORICAL, NON-BLOCKING**; do not request or execute them for this initiative. Task readiness remains blocked **only on the independent peer review gate**, not on RRI. Security/auth implementation still requires task-scoped approval and verification under the repo's policies.



**User approval scope:** B2.A plan was accepted on 2026-10-31. This task packet narrows the implementation to a standalone pure Go component. **No source code changes have been made yet** because the repository's mandatory independent task-readiness review cannot be run in the available tool container; do not represent approval of B2.A as waiver of the peer gate.

## Context and boundaries

```text
 [Future B2.A2 operator-only provisioning]
                | 32 bytes from CSPRNG
                v
        B2.A1 token primitives
        [generate/parse/compare]
                |
                v
    [Future B2.A2 Fenix SQLite digest-only storage]
                |
                v
    [Future B2.A3 isolated service middleware]
                |
                v
 [Future B2.B default-deny capability registry]
```

Invariants: 32 random bytes of entropy; unambiguous **non-JWT** token prefix; canonical URL-safe base64 encoding; bounded exact token length; constant-time digest comparison; never log or embed bearer; caller input never directly establishes workspace/user principal. No `JWT_SECRET`, user context, database, route, workflow, model, agent, tool, or on-demand capability is touched.

## Proposed files

- `internal/domain/auth/service_token.go` — **new** pure service-token primitives; no changes to existing user auth.
- `internal/domain/auth/service_token_test.go` — **new** focused negative and positive tests.

No actual files above currently exist on this branch. A2 and A3 are separate approved-scope **plans**, not automatically authorized implementation in this task.

## Task breakdown

- **A1.1** Define a service token wire format such as `fenix_s2s_v1_<base64url(32 random bytes)>`; reject missing/extra characters, altered alphabet, noncanonical encodings, malformed/padded strings, whitespace, JWT-shaped `a.b.c`, and any wrong-length token.
- **A1.2** Use `crypto/rand`, SHA-256 digest (acceptable for uniformly random 256-bit tokens; keep storage pepper/keyed digest as a reviewer option), `crypto/subtle.ConstantTimeCompare` for digest verification. Avoid echoing secret in error strings. Keep a typed `ServicePrincipal` domain identity separate from human `UserID` so A3 cannot create a synthetic user.
- **A1.3** Test randomness/uniqueness for many samples, 32-byte payload, canonical decoding, malformed tokens, empty/incorrect digest, constant-time API usage, no secrets in logged/errors or `fmt` output, user-JWT confusion. Test expiry/revocation at A2 after SQLite state exists; do not pretend a pure parser can validate database revocation.

### Language-agnostic pseudocode

```text
generate():
    random_bytes = CSPRNG(32)
    token = "fenix_s2s_v1_" + unpadded_base64url(random_bytes)
    return token, digest(SHA256(token))

parse(candidate):
    require fixed prefix and exact maximum length
    reject whitespace, dot/JWT-like syntax, padding, wrong alphabet, wrong decoded length
    decode and ensure canonical round-trip
    return candidate / opaque bytes, never source tenant identity

verify(candidate, expected_digest):
    if parse fails: return false
    candidate_digest = SHA256(candidate)
    return constant_time_equal(candidate_digest, expected_digest)
```

## Risk calculation — PRELIMINARY ONLY

Policy: `docs/policies/RRI_POLICY.md`, `scripts/rri.py`.

| Dimension | Score | Basis |
|---|---:|---|
| C | 1 | **forecast**, not measured code complexity; two new Go files not yet written |
| F | 1 | 2 proposed new files |
| D | 5 | authentication/credential boundary rubric |
| T | 3 | existing auth tests exist; no new service-token tests yet |
| A | 1 | acceptance and negative examples specified |
| K | 5 | credential security rubric floor, despite pure module |
| P | 5 | authentication/secret risk rubric floor |
| X | 2 | affects two files with later service identity context |

Weighted/base = round(100 × (0.18×1 + 0.12×1 + 0.15×5 + 0.15×3 + 0.12×1 + 0.12×5 + 0.10×5 + 0.06×2)/5) = **57**. Auth security penalty +10 → **forecast RRI 67 (Complex)**. **Not an official RRI tool output:** target Go files do not exist and complete checked-out repo/`scripts/rri.py` unavailable in this container. Final score must be computed with actual diff/CC and the script before an implementation gate is declared PASS.

```bash
python3 scripts/rri.py --platform fenix --C 1 \
  --touches internal/domain/auth/service_token.go \
  --touches internal/domain/auth/service_token_test.go \
  --D 5 --T 3 --A 1 --K 5 --P 5 --X 2 --json
python3 scripts/peer-workflow-review.py task-readiness \
  --caller-kind remote-provider \
  --task-card docs/tasks/task_dubbridge_fenix_b2a1_credential_readiness.md
```

Use `--cc` with measured raw CC when the implementation exists; the above `--C 1` is just an initial forecast.

## Gate status

- **Owner plan approval:** YES (B2.A tasks approved in conversation, 2026-10-31).
- **Independent peer task-readiness review:** **BLOCKED / NOT RUN**: neither `claude` nor `codex` CLI available in execution container; local checkout lacks full Fenix repository scripts; no independent peer artifact.
- **Official RRI:** **NOT RUN**; only reproducible policy-formula forecast above. Decomposition required for Complex band; this A1 task is already narrowed, but reviewer may require further split.
- **Actual Go code/tests:** **NOT STARTED**. No claim of `go test` PASS.
- **B2.A2 migration; B2.A3 middleware; B2.A4 production credentials:** **OUT OF SCOPE / NOT STARTED**.

**Next operator action:** on a full authorized Fenix checkout, run the two commands above, record the peer verdict and risk score, then implement only A1 if its independent gates allow it. Do not use a waiver inferred from a general plan acceptance.
