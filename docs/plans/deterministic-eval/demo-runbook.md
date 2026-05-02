---
doc_type: summary
title: Operator Runbook — Governed Support Copilot Demo (F9)
status: complete
created: 2026-05-02
---

# Operator Runbook — Governed Support Copilot Demo

This document is the authoritative step-by-step guide for running the F9 governed support demo.
A presenter with no prior context can follow this runbook and complete the demo end-to-end.

---

## Fixed Demo IDs

These IDs are stable across seed runs. Reference them directly in the demo.

| Entity | ID | Value |
|---|---|---|
| Workspace | `f9d00001-0000-4000-a000-000000000001` | FenixCRM Demo |
| Operator user | `f9d00003-0000-4000-a000-000000000001` | operator@fenix-demo.io |
| Approver user | `f9d00003-0000-4000-a000-000000000002` | approver@fenix-demo.io |
| Support case | `f9d00007-0000-4000-a000-000000000001` | "Login screen broken after update" |
| KB article | `f9d00008-0000-4000-a000-000000000001` | "Known login issue — cache invalidation" |

---

## Pre-Demo Checklist

Complete every item before starting the demo. Do not skip.

```
[ ] Backend Go is running on :8080
      make run
      # or: go run ./cmd/fenix serve --port 8080

[ ] BFF is running on :3000
      cd bff && npm start

[ ] Demo seed has been applied to the DB
      sqlite3 data/fenixcrm.db < internal/infra/sqlite/seed/demo_support.sql

[ ] Seed data verified
      sqlite3 data/fenixcrm.db \
        "SELECT id, subject, priority, status FROM case_ticket WHERE id='f9d00007-0000-4000-a000-000000000001';"
      # Expected: f9d00007-...|Login screen broken after update|high|open

[ ] Mobile app is running and connected to BFF at http://localhost:3000
      cd mobile && npx expo start

[ ] Mobile is logged in as operator@fenix-demo.io

[ ] BFF Admin is open in browser at http://localhost:3000/admin
      (token must be active — log in as approver@fenix-demo.io for the approval step)

[ ] Review Packet is accessible
      internal/domain/eval/testdata/packets/demo_support_run.md
      internal/domain/eval/testdata/packets/demo_support_run.es.md
```

---

## Demo Narrative

### Act 1 — The Case (Mobile)

**Goal**: Show that the operator has real, governed access to customer context.

1. Open the mobile app. Navigate to the **Support** tab.
2. The list shows open cases. Point out: priority badge, status, and signal count.
3. Tap the case **"Login screen broken after update"** (priority: high).
4. Show the case detail:
   - Subject, description, linked account: Acme Enterprise
   - Contact: Dana Chen, Head of Engineering
   - Priority: high — gold SLA tier
   - Signals section (if active signals exist)
   - Agent activity section (empty before the trigger)

**Talking point**: _"This is the operator's real working surface. They see the customer context, the priority, and any AI signals — all in one place."_

---

### Act 2 — Triggering the Governed Support Agent (Mobile)

**Goal**: Show the operator handing off to a governed AI agent with an explicit query.

1. From the case detail, tap **"Support Copilot"** (navigates to `/support/{id}/copilot`).
2. The Copilot screen opens with the case context pre-loaded in the banner.
3. Type in the input field:

   ```
   Customer reports login screen broken after the latest update. ~200 enterprise users affected. EMEA region. Gold SLA.
   ```

4. Press **Send**.
5. The Copilot sends an SSE query AND fires the support agent trigger with:
   ```json
   {
     "case_id": "f9d00007-0000-4000-a000-000000000001",
     "customer_query": "Customer reports login screen broken..."
   }
   ```
6. The app navigates automatically to the **Activity** screen for the new run.

**Talking point**: _"The operator doesn't write a prompt. They describe the customer's problem in plain language. The system takes that as the trigger payload — governed, audited, traceable."_

---

### Act 3 — The Agent Decides It Needs Approval (Mobile + BFF Admin)

**Goal**: Show that the agent does not act autonomously on a sensitive mutation.

**On mobile (Activity screen):**
1. Show the run status: `queued` → `awaiting_approval`.
2. Point out: the agent retrieved evidence from the KB and from the case history.
3. The agent identified `update_case` as a sensitive mutation requiring manager approval.
4. Status is `awaiting_approval` — **this is a correct outcome, not an error**.

**Talking point**: _"The agent knows the rules. It does not send a response or close the case on its own. It stops and asks a human."_

**Switch to BFF Admin (approver@fenix-demo.io):**
1. Open `http://localhost:3000/admin/approvals`.
2. Show the approval request: actor, proposed action, evidence pack summary.
3. Click **Approve**.
4. The run continues — show the status transition to `completed`.

**Talking point**: _"The approver sees exactly what the agent wants to do, the evidence it used, and the policy it checked. One click. Full audit trail."_

---

### Act 4 — Observability (BFF Admin)

**Goal**: Show that every step is traceable and auditable.

1. Open `http://localhost:3000/admin/agent-runs/{runId}`.
2. Show:
   - Tool calls: `retrieve_evidence`, `request_approval`, `update_case` (post-approval)
   - Evidence: KB article snippet, case history excerpt, confidence: high
   - Cost: tokens, latency, EUR cost per run
   - Audit events: policy check, approval request, tool execution, run completion

**Talking point**: _"Nothing is a black box. Every retrieval query, every policy check, every tool call is logged. Exportable, queryable, defensible."_

---

### Act 5 — Closing with the Review Packet

**Goal**: Show that the behavior is measurable, not just narratable.

Open the Review Packet:

```
internal/domain/eval/testdata/packets/demo_support_run.md
```

Walk through:

1. **Scenario contract** — what was the expected behavior (golden reference).
2. **Actual trace** — what the agent actually did.
3. **Expected vs Actual** comparison — field by field.
4. **Score**: 100/100
5. **Hard gates**: 0 violations
6. **Verdict**: `pass`

Read the closing line from `demo-notes.md`:

> _"In vez de pedirle a otro LLM que opine si este run parece bueno, definimos por adelantado el comportamiento gobernado esperado. Luego comparamos la traza real contra ese contrato, calculamos métricas, aplicamos hard gates y exportamos un packet que cualquier responsable técnico o de producto puede revisar directamente."_

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Backend fails to start | Missing `JWT_SECRET` in env | Add `JWT_SECRET=<min-32-chars>` to `.env` |
| Seed fails with FK error | Migrations not applied | Run `make migrate` before the seed |
| Seed fails with UNIQUE error | Data exists from previous run | `INSERT OR IGNORE` protects against this — safe to re-run |
| Trigger returns 400 | `customer_query` is empty | Type a non-empty message before pressing Send |
| Run stays `queued` for >10s | Backend unreachable from BFF | Verify `BACKEND_URL=http://localhost:8080` in BFF `.env` |
| Activity screen doesn't update | SSE connection dropped | Pull-to-refresh on Activity tab |
| Approval not visible in admin | Wrong workspace in token | Re-login as `approver@fenix-demo.io` |
| Review Packet shows wrong data | Wrong packet file opened | Use `demo_support_run.md`, not `sample_support_run.md` |

---

## Deterministic Artefacts Reference

| Artefact | Path |
|---|---|
| Demo trace (JSON) | `internal/domain/eval/testdata/demo/support_case_demo.json` |
| Review Packet (Markdown) | `internal/domain/eval/testdata/packets/demo_support_run.md` |
| Review Packet (JSON) | `internal/domain/eval/testdata/packets/demo_support_run.json` |
| Review Packet (Spanish) | `internal/domain/eval/testdata/packets/demo_support_run.es.md` |
| Scenario contract | `internal/domain/eval/testdata/scenarios/sc_support_sensitive_mutation_approval.yaml` |
| Seed SQL | `internal/infra/sqlite/seed/demo_support.sql` |
| Surfaces reference | `docs/plans/deterministic-eval/demo-execution-surfaces.md` |
| Demo notes | `docs/plans/deterministic-eval/demo-notes.md` |
