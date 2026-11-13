# B4.Q5 — Independent two-repository security review packet (complete diff manifest)

**Purpose:** cure the prior selected-snippets L3 reviewer failure. Compiled by ChatGPT Cloud on 2026-11-13. **This is an index/requirements packet, NOT an independent security verdict.** Reviewer must open all relevant changed files from both full pinned comparisons, not merely the excerpts in this document. The diff also includes unrelated platform changes; triage them separately rather than suppressing them.

## Fénix complete comparison

- Repository: `krukmat/fenix`
- Base main commit: `d3973b3870978be549758ca07a8a74b4e7d936f0`
- Current reviewed code snapshot commit: `a85aacebb5f20fafde99a7ace2a30746de0aa664` (**review packet generation snapshot; subsequent docs-only commits require final SHA refresh**)
- Full diff: https://github.com/krukmat/fenix/compare/d3973b3870978be549758ca07a8a74b4e7d936f0...a85aacebb5f20fafde99a7ace2a30746de0aa664
- All changed paths indexed below: **157**, GitHub compare ahead 338 commits; no selection filter

## DubBridge complete comparison

- Repository: `krukmat/dubbridge`
- Base main commit: `ec3314ef594f8afcd25b487ef946789ee74cb7ee`
- Current reviewed code snapshot commit: `91844207efcc97c7332088c94110c00cc9471672` (**review packet generation snapshot; subsequent docs-only commits require final SHA refresh**)
- Full diff: https://github.com/krukmat/dubbridge/compare/ec3314ef594f8afcd25b487ef946789ee74cb7ee...91844207efcc97c7332088c94110c00cc9471672
- All changed paths indexed below: **18**, GitHub compare ahead 72 commits; no selection filter

## Mandatory review coverage (no selected-snippet substitute)

1. **Identity, credentials and boundary separation (Fénix):** `internal/domain/auth/service_token.go`, `service_principal_store.go`, `internal/api/middleware/dubbridge_service_auth.go`, `dubbridge_source_guard.go`; require unrelated human JWT rejection, principal status/revocation, credential rotation/canonical token, bound org/project/workspace and sanitized errors.
2. **Operator control / capability admission (Fénix):** `internal/api/s2s_policy_operator.go`, `internal/domain/integration/s2s_capability_policy_store.go`, `s2s_capability_schema.go`, `s2s_capability_registry.go`, SQL migrations 038–040; distinguish operator RBAC management authority from service bearer and runtime ToolRegistry permissions. Verify default disabled, audit append-only, immutable schema and version guard.
3. **Atomicity/idempotency (Fénix):** `internal/domain/integration/service_operation_store.go`, `service_operation_recovery.go`, `s2s_admission.go`, `s2s_admission_envelope.go`, `internal/api/s2s_admission_http.go`; verify scope from persisted Fénix identity, BEGIN IMMEDIATE policy recheck, compare-and-set version, replay conflict, ambiguity without redispatch, no silent operation deletion or terminal-state mutation.
4. **Transport and caller authorization (DubBridge):** `apps/api/src/fenix_scope.rs`, `crates/connectors/src/fenix_s2s.rs`, Rust module registration; verify local membership and project-to-org relation, TLS-only, no redirects, token redaction, bounded request/reply, deterministic operation IDs, no automatic retry on ambiguous outcomes.
5. **Cross-repository wire parity:** request, ack and error JSON Schemas and pinned SHA references; all denial statuses; 200=replay and 202=new; unknown result requires reconciliation. No GET/status C4, no business workflow/agent execution in B4.
6. **Routes and default-off proof:** verify normal production routers, workers, configuration, permission grants and deployment do not instantiate/enable dormant constructors. Unmounted handler is intentional **not** a standalone H finding.

## Previous L3 reviewer false positives to exclude unless independently reproven

- Calling the deliberately unmounted Fénix handler a high-severity exploit is an incorrect assertion without an unintended active route.
- Rust service bearer **does** enforce canonical Base64URL final sextets after earlier correction.
- `FenixClient::submit_once` sends once and implements **no automatic replay**.
- PostgreSQL `PgFenixScopeAuthorizer` checks org membership/role and project-to-org relation.
- Do **not** conclude all findings are false. The previous Devstral packet was incomplete; an independent reviewer must reopen each allegation against full code, tests and actual evidence.

## Mandatory output

Every accepted finding: **ID | H/M/L | affected repo | exact file:line | failure mechanism | reproducible preconditions/case | proof | required correction | blocks B4 or deferred activation**. Mark unproven allegations `UNVERIFIED`, don't promote them to security facts. Distinguish build formatting debt (Go fmt / Rustfmt / Clippy) from security issues. A valid review has access to **both complete pinned diffs below**, tests and architecture, and may ask for additional exact file ranges. GO/NO-GO stays NO-GO until full Go prepush, Cargo fmt/check/test/clippy, and independent security review all genuinely pass.

**Evidence received on 2026-11-13**: user reported L1 Python SQLite 23 PASS, focused Go and vet PASS; prepush blocked by Go fmt. L2 Cargo check and tests 321 PASS/1 ignored; fmt and Clippy blocked. Those results were for **older** pair Fénix `d07adb66` / DubBridge `bf36a9d3`, so they are not a new QA PASS for later changes. Actual log files were not attached to this Cloud review package; labels are not substitutions for primary logs.

## Fénix — complete 157-file path manifest

- `.github/workflows/ci.yml` (modified)
- `.gitignore` (modified)
- `AGENTS.md` (modified)
- `CLAUDE.md` (modified)
- `docs/audit/dubbridge-fenix-b1-contract-check-2026-10-29.md` (added)
- `docs/audit/dubbridge-fenix-b1-neutral-checkpoint-2026-10-31.md` (added)
- `docs/audit/dubbridge-fenix-b1c-gate-2026-10-31.md` (added)
- `docs/audit/dubbridge-fenix-b2-cloud-checkpoint-2026-11-05.md` (added)
- `docs/audit/dubbridge-fenix-b2a1-cloud-2026-11-05.md` (added)
- `docs/audit/dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md` (added)
- `docs/audit/dubbridge-fenix-b2a2-remediation-2026-11-06.md` (added)
- `docs/audit/dubbridge-fenix-b2a3-security-verification-2026-11-09.md` (added)
- `docs/audit/dubbridge-fenix-b2c-cloud-checkpoint-2026-11-10.md` (added)
- `docs/audit/dubbridge-fenix-b2d-nonstatus-source-audit-2026-11-10.md` (added)
- `docs/audit/dubbridge-fenix-b2e-cloud-source-review-2026-11-11.md` (added)
- `docs/audit/dubbridge-fenix-b4q-cloud-first-preflight-2026-11-13.md` (added)
- `docs/audit/dubbridge-fenix-b4q-cloud-validation-2026-11-11.md` (added)
- `docs/audit/dubbridge-fenix-b4q-q5-gate-triage-2026-11-13.md` (added)
- `docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.down.sql` (added)
- `docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.up.sql` (added)
- `docs/contracts/dubbridge-fenix-b2c-operation-proposal.down.sql` (added)
- `docs/contracts/dubbridge-fenix-b2c-operation-proposal.up.sql` (added)
- `docs/contracts/dubbridge-fenix-v1-admission-ack.schema.json` (added)
- `docs/contracts/dubbridge-fenix-v1-admission-error.schema.json` (added)
- `docs/contracts/dubbridge-fenix-v1-envelope-error.schema.json` (added)
- `docs/contracts/dubbridge-fenix-v1-envelope-request.schema.json` (added)
- `docs/contracts/dubbridge-fenix-v1-envelope-response.schema.json` (added)
- `docs/contracts/dubbridge-fenix-v1-request.schema.json` (added)
- `docs/contracts/dubbridge-fenix-v1-run.schema.json` (added)
- `docs/decisions/ADR-DB-FENIX-S2S-PROPOSED.md` (added)
- `docs/handoffs/dubbridge-fenix-b2c-fixpack-2026-11-10.md` (added)
- `docs/handoffs/dubbridge-fenix-b2c-readiness-validation-2026-11-10.md` (added)
- `docs/handoffs/dubbridge-fenix-b4q-final-local-batch-2026-11-13.md` (added)
- `docs/integration-overview.md` (modified)
- `docs/patches/b2a3-t2-mandatory-scope-guard.patch` (added)
- `docs/plans/dubbridge-consumes-fenix-integration.md` (added)
- `docs/plans/dubbridge-fenix-b1-t1-api-runtime-assessment.md` (added)
- `docs/plans/dubbridge_fenix_s2s_security_plan.md` (added)
- `docs/plans/fenix_go_prepush_unblock_plan.md` (added)
- `docs/plans/portable_agent_workflow_port_plan.md` (modified)
- `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` (modified)
- `docs/policies/HITL_AUTONOMY_POLICY.md` (modified)
- `docs/policies/LOCAL_MODEL_POLICY.md` (modified)
- `docs/policies/RRI_POLICY.md` (modified)
- `docs/tasks/.gitignore` (modified)
- `docs/tasks/dubbridge-fenix-b2a-s2s-identity.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2a1_credential_readiness.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2a2_security_remediation.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2a3_mandatory_scope_guard.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2a3_middleware_readiness.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2a3_security_handoff.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2a3_sqlite_http_integration.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2b_registry.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2c2_operation_store.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2c3_recovery.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2c4_status_resolver.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2c_operation.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2d1_denial_contract.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2d2_trace_audit.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2d3_admission_preflight.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2d_nonstatus.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2e1_inert_envelope.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2e2_inert_reservation.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2e3_inert_errors.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2e4_inert_tests.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2e5_inert_audit.md` (added)
- `docs/tasks/task_dubbridge_fenix_b2e_inert_admission.md` (added)
- `docs/tasks/task_dubbridge_fenix_b4_t1_detached_http_admission.md` (added)
- `docs/tasks/task_dubbridge_fenix_b4q_cloud_first_execution.md` (added)
- `docs/tasks/task_dubbridge_fenix_provider_int.md` (added)
- `docs/tasks/task_fenix_go_prepush_unblock.md` (added)
- `docs/tasks/task_local_primary_peer_review.md` (added)
- `internal/api/integration_runtime.go` (modified)
- `internal/api/integration_runtime_test.go` (modified)
- `internal/api/middleware/auth.go` (modified)
- `internal/api/middleware/dubbridge_service_auth.go` (added)
- `internal/api/middleware/dubbridge_service_auth_integration_test.go` (added)
- `internal/api/middleware/dubbridge_service_auth_test.go` (added)
- `internal/api/middleware/dubbridge_source_guard.go` (added)
- `internal/api/middleware/dubbridge_source_guard_test.go` (added)
- `internal/api/middleware/ratelimit.go` (modified)
- `internal/api/s2s_admission_composition.go` (added)
- `internal/api/s2s_admission_http.go` (added)
- `internal/api/s2s_admission_http_test.go` (added)
- `internal/api/s2s_admission_sqlite_test.go` (added)
- `internal/api/s2s_policy_operator.go` (added)
- `internal/api/s2s_policy_operator_test.go` (added)
- `internal/domain/audit/service_test.go` (modified)
- `internal/domain/auth/service_principal_store.go` (added)
- `internal/domain/auth/service_principal_store_test.go` (added)
- `internal/domain/auth/service_token.go` (added)
- `internal/domain/auth/service_token_test.go` (added)
- `internal/domain/blackboard/planner_test.go` (modified)
- `internal/domain/evidence/contract_test.go` (modified)
- `internal/domain/evidence/lifecycle.go` (modified)
- `internal/domain/evidence/proof_reference.go` (modified)
- `internal/domain/evidence/runtime.go` (modified)
- `internal/domain/flowinterop/contract.go` (modified)
- `internal/domain/flowinterop/contract_test.go` (modified)
- `internal/domain/flowinterop/fidelity.go` (modified)
- `internal/domain/flowinterop/governance_test.go` (modified)
- `internal/domain/governance/decision.go` (modified)
- `internal/domain/governance/decision_test.go` (modified)
- `internal/domain/governance/runtime.go` (modified)
- `internal/domain/integration/s2s_admission.go` (added)
- `internal/domain/integration/s2s_admission_envelope.go` (added)
- `internal/domain/integration/s2s_admission_projection.go` (added)
- `internal/domain/integration/s2s_admission_test.go` (added)
- `internal/domain/integration/s2s_admission_wire.go` (added)
- `internal/domain/integration/s2s_admission_wire_test.go` (added)
- `internal/domain/integration/s2s_b4_inert_interop_test.go` (added)
- `internal/domain/integration/s2s_capability_policy_store.go` (added)
- `internal/domain/integration/s2s_capability_policy_store_test.go` (added)
- `internal/domain/integration/s2s_capability_registry.go` (added)
- `internal/domain/integration/s2s_capability_registry_test.go` (added)
- `internal/domain/integration/s2s_capability_schema.go` (added)
- `internal/domain/integration/service_operation_digest.go` (added)
- `internal/domain/integration/service_operation_recovery.go` (added)
- `internal/domain/integration/service_operation_recovery_test.go` (added)
- `internal/domain/integration/service_operation_store.go` (added)
- `internal/domain/integration/service_operation_store_test.go` (added)
- `internal/domain/tool/capability.go` (modified)
- `internal/domain/tool/capability_runtime_pipeline.go` (modified)
- `internal/domain/tool/capability_runtime_pipeline_test.go` (modified)
- `internal/domain/tool/execution_pipeline.go` (modified)
- `internal/domain/tool/registry.go` (modified)
- `internal/domain/tool/registry_test.go` (modified)
- `internal/infra/integration/evidenceoutbox/recorder.go` (modified)
- `internal/infra/integration/evidenceoutbox/recorder_test.go` (modified)
- `internal/infra/integration/evidenceoutbox/store.go` (modified)
- `internal/infra/integration/m2sf/executor.go` (modified)
- `internal/infra/integration/vel/lifecycle_test.go` (modified)
- `internal/infra/integration/vel/sink.go` (modified)
- `internal/infra/sqlite/migrations/038_service_identity.down.sql` (added)
- `internal/infra/sqlite/migrations/038_service_identity.up.sql` (added)
- `internal/infra/sqlite/migrations/039_service_operation.down.sql` (added)
- `internal/infra/sqlite/migrations/039_service_operation.up.sql` (added)
- `internal/infra/sqlite/migrations/040_s2s_capability_policy.down.sql` (added)
- `internal/infra/sqlite/migrations/040_s2s_capability_policy.up.sql` (added)
- `internal/infra/sqlite/s2s_capability_policy_migration_test.go` (added)
- `internal/infra/sqlite/service_identity_migration_test.go` (added)
- `internal/infra/sqlite/service_operation_migration_test.go` (added)
- `internal/server/relationship_runtime_test.go` (modified)
- `scripts/hooks/pre-push` (modified)
- `scripts/int_b1_contract_selftest.py` (added)
- `scripts/int_b1_contract_status_matrix.py` (added)
- `scripts/int_b1_envelope_matrix.py` (added)
- `scripts/int_b1_envelope_selftest.py` (added)
- `scripts/int_b1_fenix_live_probe.py` (added)
- `scripts/int_b2a2_sqlite_proposal_selftest.py` (added)
- `scripts/int_b2a2_store_sqlite_smoke.py` (added)
- `scripts/int_b2b4_capability_sqlite_smoke.py` (added)
- `scripts/int_b2c_operation_sqlite_smoke.py` (added)
- `scripts/int_b4q_format_only.sh` (added)
- `scripts/peer-workflow-review.py` (modified)
- `scripts/peer_workflow_review_test.py` (modified)

## DubBridge — complete 18-file path manifest

- `Cargo.lock` (modified)
- `apps/api/Cargo.toml` (modified)
- `apps/api/src/fenix_scope.rs` (added)
- `apps/api/src/lib.rs` (modified)
- `crates/connectors/Cargo.toml` (modified)
- `crates/connectors/src/fenix_s2s.rs` (added)
- `crates/connectors/src/lib.rs` (modified)
- `docs/audit/dubbridge-fenix-b3-cloud-source-audit-2026-11-11.md` (added)
- `docs/audit/dubbridge-fenix-b4-offline-interop-2026-11-11.md` (added)
- `docs/contracts/fenix-b1-request.schema.json` (added)
- `docs/contracts/fenix-b4-admission-ack.schema.json` (added)
- `docs/contracts/fenix-b4-admission-error.schema.json` (added)
- `docs/contracts/fenix-b4-pinned-manifest.json` (added)
- `docs/plan/dubbridge-consumes-fenix-integration.md` (added)
- `docs/plan/dubbridge-fenix-b1-contract-v1-proposal.md` (added)
- `docs/plan/roadmap.md` (modified)
- `docs/tasks/dubbridge-fenix-service-consumption.md` (added)
- `scripts/check-fenix-b4-contract.mjs` (added)
