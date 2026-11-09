---
doc_type: plan
id: FENIX-GO-PREPUSH-UNBLOCK
title: "Clear Go pre-push gates for the DubBridge integration branch"
status: completed
date: 2026-11-09
tags: [fenix, dubbridge, go, qa, pre-push]
---

# Clear Go pre-push gates for the DubBridge integration branch

## Current direction

The owner directed the agent to resolve related local blockers so the B2.A3.T3 formatting commit can be pushed. The branch contains the T3 formatting commit `367a70c` and a separate mechanical formatting commit `43d935c` for the 26 files reported by `fmt-check`. Each formatted file matches `gofmt` of its parent revision byte for byte. Scoped fixes addressed complexity, lint and wrapcheck. The full `GOTOOLCHAIN=go1.25.13 bash scripts/qa-go-prepush.sh` passed: application coverage 83.0%, TDD coverage 84.7%, deadcode 0, race stability PASS, pattern gate PASS. `go vet ./...` also passed. The branch hook's operator pause was not used as QA evidence. Track the existing debt in [issue #4](https://github.com/krukmat/fenix/issues/4) and the T3 result in [issue #7](https://github.com/krukmat/fenix/issues/7).

## Execution

1. Confirm each blocker against the actual branch and keep the unrelated local `.gitignore` change out of commits.
2. Apply `gofmt` only to the files named by `fmt-check`; verify the diff equals Go's formatter output and commit that mechanical change separately.
3. Run each remaining gate, capture failures, and make the smallest behavior-preserving fixes. Keep auth and security-boundary changes separately reviewed and obtain task-scoped owner approval before any such change. Do not weaken gates or activate a route.
4. Confirm coverage against the actual gate and fill a demonstrated test gap. Triage vulnerability findings without silently changing the required Go toolchain or dependencies.
5. Run the full Go pre-push QA with the required runtime, obtain independent post-code review on each scoped change, then push the branch only after the required gates pass. Publish SHA, logs, hashes, and gate outcomes in the coordination issues.

## Exit

The requested branch has a remote SHA containing the T3 format commit, the required local gates pass or have an explicit applicable owner-authorized exception, and no unrelated local work is included.
