# Task BDD Fix - Restore Doorstop Pipeline Integrity

**Status**: In Progress
**Phase**: BDD Strategy and Traceability Stabilization
**Depends on**: `reqs/UC/.doorstop.yml`, CI `doorstop-check`
**Required by**: green CI baseline before Wave 3 execution

---

## Objective

Fix the GitHub pipeline regression introduced by the new UC Doorstop family so the
existing `doorstop-check` job accepts the requirements tree again.

---

## Scope

1. Diagnose the CI failure from the GitHub Actions run
2. Fix the Doorstop hierarchy so `UC` is not treated as a second root
3. Revalidate locally with Doorstop and `cmd/frtrace`
4. Push the fix and confirm the pipeline state

---

## Root Cause

The initial UC Doorstop family was created as an independent root document tree:

- `FR` root already existed
- `UC` was introduced without `parent`

That caused Doorstop integrity validation to fail in CI with:

`ERROR: multiple root documents: FR and UC`

---

## Implemented

- set `parent: FR` in `reqs/UC/.doorstop.yml`
- aligned `cmd/frtrace` UC testdata with the same Doorstop hierarchy
- normalized `UC -> FR` links to Doorstop UID format such as `FR001`
- updated `cmd/frtrace` to normalize compact Doorstop link IDs back to repo file IDs

---

## Pending Verification

- local Doorstop integrity check
- local `go test ./cmd/frtrace`
- push fix and inspect GitHub Actions result

---

## Implementation References

- `reqs/UC/.doorstop.yml`
- `cmd/frtrace/testdata/reqs/UC/.doorstop.yml`
- `docs/tasks/task_bdd_pipeline_doorstop_fix.md`
