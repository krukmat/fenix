---
doc_type: audit
id: DB-FENIX-B1-CONTRACT-2026-10-29
title: "B1 contract local schema validation (not E2E)"
status: offline_pass_live_unverified
date: 2026-10-29
---

# B1 — Local offline schema verification receipt

## Source and setup

- Repository: `krukmat/fenix`
- Branch: `feature/int-v2-dubbridge-consumes-fenix`
- Execution: isolated Python container with installed `jsonschema==4.26.0`; no running Fenix server or DubBridge service.
- Files were reconstructed in local runtime and verified byte-for-byte against the **same Git blob SHA** returned by GitHub for each of the following files:

| File | Git blob SHA verified |
|---|---|
| `scripts/int_b1_contract_selftest.py` | `3045c9187e9b25c689e1a0f27589df51995a5413` |
| `scripts/int_b1_contract_status_matrix.py` | `3a2beaef58f53f8ebe00969b3ee300b45888c4c2` |
| `docs/contracts/dubbridge-fenix-v1-request.schema.json` | `dd44ccd5e8e7daa773a6d5f9cc7f142c7ea6a6d2` |
| `docs/contracts/dubbridge-fenix-v1-run.schema.json` | `44f1f1b5597ba7613d035ed55666d00cf9e38ac7` |

Exact commands after reconstructing these versioned files in `/mnt/data/int-b1-verify`:

```bash
python /mnt/data/int-b1-verify/scripts/int_b1_contract_selftest.py
python /mnt/data/int-b1-verify/scripts/int_b1_contract_status_matrix.py
```

## Output (2026-10-29)

```json
{"kind":"FENIX_DUBBRIDGE_B1_CONTRACT_SELFTEST_V1","status":"PASS","positive":4,"negative":14,"scope":"offline JSON schema only; NOT live API, NOT S2S, NOT E2E"}
{"kind":"FENIX_B1_STATUS_MATRIX_V1","status":"PASS","allowed":10,"rejected":50,"combinations":60,"scope":"offline schema only; NOT provider runtime, NOT E2E"}
```

Total: **18/18 directed cases** plus **60/60 exhaustive status/result/error combinations** passed against the *proposed* JSON Schemas.

## Failure found and corrected

An earlier schema allowed `accepted+result`, `failed+result`, `succeeded+error_code`, `denied` without a required error code, and `abstained+result`. Updated `run.schema.json` now enforces:

- `succeeded | completed_with_warnings`: result must be an advisory object; no error code.
- `accepted | running | awaiting_approval | handed_off | abstained`: result must be null; no error code.
- `failed | denied | indeterminate`: result null, with a typed error code.
- No extra fields at the top level or in nested advisory output.

These are schema-level invariants only. They do not validate authentication, authorization, content safety of natural-language advice, HTTP/JSON byte limits, model quality or business correctness.

## Remaining explicit blockers

1. **B1.T1 LIVE PENDING:** existing Fenix user-JWT workflow-run diagnostic has not been executed against a real active read-only workflow. The existing `scripts/int_b1_fenix_live_probe.py` is available, but no disposable local Fenix instance and operator test JWT were provided to this tool session.
2. **B1.T2 NOT ACCEPTED:** Fenix-owned service-credential ADR remains proposed, RRI and independent security/peer review not executed here. No restricted credential issued.
3. **B1.T3 NOT FROZEN:** contract is candidate, not implemented or signed off. RRI/peer gate pending; B2 may not claim production authorization from this receipt.
4. No Go full-app build, Rust service execution, PostgreSQL source data query or cross-product E2E.

**Verdict: OFFLINE CONTRACT PASS, B1 OVERALL OPEN / E2E NO-GO.** The repository's GitHub Actions remain intentionally disabled; the above proof used no CI runs.
