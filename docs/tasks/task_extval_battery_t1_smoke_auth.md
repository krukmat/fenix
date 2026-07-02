---
doc_type: task
id: EXTVAL-BATTERY-T1-001
title: "Run battery T1 — Backend/BFF smoke and auth against live external validation runtime"
status: complete
phase: external-validation-battery
week: "2026-W27"
tags: [external-validation, battery, smoke, auth, crud, audit]
fr_refs: []
uc_refs: []
blocked_by: [EXTVAL-BATTERY-RUN-001]
blocks: [EXTVAL-BATTERY-T2-001]
files_affected: []
created: 2026-07-01
completed: 2026-07-01
---

# Task EXTVAL-BATTERY-T1-001

**Plan**: [External Validation First Test Battery Plan](../plans/external_validation_first_test_battery_plan.md#t1-backendbff-smoke-and-auth)

## Task Card

Task: EXTVAL-BATTERY-T1-001

Task file: docs/tasks/task_extval_battery_t1_smoke_auth.md

Plan file: docs/plans/external_validation_first_test_battery_plan.md

Summary: Execute battery T1 against the backend (PID 9881/10136, :8080) and BFF (PID 10726, :3000) started and verified healthy in EXTVAL-BATTERY-RUN-001. Register/login a validation operator through `/bff/auth/*`, create account/contact/deal/case through `/bff/api/v1/*` proxied routes, verify persistence via read endpoints and SQLite row counts, and verify audit events are recorded for protected calls.

Code affected: None expected. Read/write calls against the already-running system; no source files touched.

Effort/reasoning: Low - the plan already enumerates the exact steps and endpoints (T1 section); the only judgment call is redacting the bearer token in evidence output.

Recommended model: claude-sonnet-4-6

Estimated tokens: ~7000

Task type: operational validation. No dev-task pseudocode required — no code is written.

## Operational Procedure

1. `POST /bff/auth/register` with `{email, password, displayName, workspaceName}` to create a validation operator; capture the returned bearer token (redacted in reports).
2. Use the token to `POST /bff/api/v1/*` create calls for one account, one contact (linked to the account), one deal (linked to the account), and one support case (linked to account/contact).
3. Read back each created entity via its `GET` endpoint to confirm persistence.
4. Query `sqlite3 ./data/external-validation/fenixcrm.db` directly for row counts on the created entities as independent evidence, not just API responses.
5. Query the audit endpoint (or `sqlite3` on the audit table) to confirm audit events exist for the register/login and each protected CRUD call.
6. Record go/no-go per the plan: no 5xx in auth or CRUD, BFF health reports backend reachable, no fixture responses.

## Acceptance Criteria

1. Register and login succeed and return a valid bearer token. — PASS: `POST /bff/auth/register` returned `{token, userId, workspaceId}`; `POST /bff/auth/login` returned `200`.
2. Account, contact, deal, and case are created successfully (2xx) through BFF-proxied routes. — PASS: all four created with full object echoes. At the time of this 2026-07-01 run, deal/case each required a manually created pipeline+stage first; this workaround is historical and superseded for newly registered workspaces by `ADR032-BOOTSTRAP-IMPL-001`.
3. All four entities are readable back via GET and match what was created. — PASS: `GET /accounts/{id}`, `/contacts/{id}`, `/deals/{id}`, `/cases/{id}` all returned `HTTP 200` with matching data.
4. SQLite row counts independently confirm persistence (not just API echo). — PASS: queried `data/external-validation/fenixcrm.db` directly; `account`, `contact`, `deal`, `case_ticket` each show exactly 1 row, matching the created IDs by primary key lookup.
5. Audit events exist for the protected calls (at minimum: register/login, and the four create calls). — PASS: 20 audit_event rows, all `outcome=success`, covering `register`, `login`, `account.created`/`create_account`, `contact.created`/`create_contact`, `create_pipeline` (x2), `deal.created`/`create_deal`, `case.created`/`create_case`, `knowledge.reindex` (account + case_ticket), and `get_account`/`get_contact`/`get_deal`/`get_case`.
6. No 5xx responses anywhere in the sequence. — PASS: every call returned `200`.
7. No response resembles a screenshot/fixture payload. — PASS: all responses contain real generated UUIDs and the exact field values submitted; no `Snapshot fixture response` or fixture-source markers present.

## Deviations From Plan Text

- Historical note: this run exposed that the plan's T1 steps (register → login → create account/contact/deal/case) did not account for missing `deal` and `case` pipelines in a freshly registered workspace. This task created one pipeline+stage for `deal` and one for `case` as an unavoidable prerequisite on 2026-07-01. `ADR032-BOOTSTRAP-IMPL-001` now supersedes this workaround for newly registered workspaces by creating default `deal` and `case` pipelines during registration. Existing validation workspaces are not backfilled automatically.
- Plaintext token and register-response files (`data/extval-t1-token.txt`, `data/extval-t1-register.json`) were created transiently under the already-gitignored `data/` path and deleted immediately after use, once IDs were extracted into `data/extval-t1-ids.txt` (non-secret UUIDs only).

## Evidence Summary

- Workspace: `019f1e8c-0b04-7782-2983-8d06e9d59e8c`
- Account: `019f1e8c-9ae8-7996-3f78-ec7c025eca7f`
- Contact: `019f1e8c-c3b3-78ae-f6b5-8690afcca2a8`
- Deal: `019f1e8c-f9eb-7a8d-dd11-630f8f69cfe8` (pipeline `019f1e8c-c3d4-...`, stage `019f1e8c-c3ee-...`)
- Case: `019f1e8c-fa03-7cb0-d248-f497f0131cdb` (pipeline `019f1e8c-f9b2-...`, stage `019f1e8c-f9d1-...`)
- Audit event count at end of run: 20, all `success`.
