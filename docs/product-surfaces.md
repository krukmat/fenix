# Product Surfaces

All screenshots below come from the live product screenshot suites.

[← Back to README](../README.md)

## Mobile

### Entry

![Login screen](../mobile/artifacts/screenshots/01_auth_login.png)

Every action starts with a user and workspace.

### Inbox

![Inbox](../mobile/artifacts/screenshots/02_inbox.png)

The main operational queue for approvals, handoffs, signals and policy rejections.

### Signal detail

![Signal detail](../mobile/artifacts/screenshots/06_inbox_signal_detail.png)

Signals make machine judgment reviewable through confidence and supporting context.

### Support case

![Support case detail](../mobile/artifacts/screenshots/03_support_case_detail.png)

Case context, evidence, actions and governance are kept together.

### Sales brief

![Sales brief](../mobile/artifacts/screenshots/04_sales_brief.png)

Grounded account context, active risks and suggested next actions.

### Denied execution

![Denied-by-policy activity trace](../mobile/artifacts/screenshots/08_activity_run_detail_denied.png)

A denied action remains visible and inspectable.

### Governance

![Governance overview](../mobile/artifacts/screenshots/05_governance.png)

Usage, actor, tool, model, latency, cost and policy state are visible together.

### Audit

![Governance audit trail](../mobile/artifacts/screenshots/09_governance_audit.png)

Audit remains available where work happens.

### Workflow graph

![Workflow graph](../mobile/artifacts/screenshots/18b_workflow_graph.png)

Governed workflow logic is visible rather than hidden.

### CRM hub

![CRM hub](../mobile/artifacts/screenshots/19_crm_hub.png)

CRM entities provide the operational context agents reason over.

## Admin surface

The BFF exposes a session-backed operator surface at `/bff/admin`.

| Login | Dashboard | Workflows |
|---|---|---|
| ![login](../bff/artifacts/admin-screenshots/00_login.png) | ![dashboard](../bff/artifacts/admin-screenshots/01_dashboard.png) | ![workflows](../bff/artifacts/admin-screenshots/02_workflows_list.png) |

| Agent Runs | Approvals | Audit |
|---|---|---|
| ![agent runs](../bff/artifacts/admin-screenshots/06_agent_runs_list.png) | ![approvals](../bff/artifacts/admin-screenshots/08_approvals_list.png) | ![audit](../bff/artifacts/admin-screenshots/09_audit_list.png) |

The browser never needs a bearer token: authentication uses an HTTP-only session cookie.

## Workflow authoring

```mermaid
flowchart LR
    L[Workflows list] --> C[Create draft]
    C --> B[Visual / text builder]
    B --> D[Review]
    D --> A[Activate]
    D --> B
```

| Create draft | Visual builder | Review and activate |
|---|---|---|
| ![create draft](../bff/artifacts/admin-screenshots/03_workflow_create_draft.png) | ![builder](../bff/artifacts/admin-screenshots/04_workflow_builder_bound.png) | ![detail](../bff/artifacts/admin-screenshots/05_workflow_detail.png) |

- **Create** — start from a minimal draft.
- **Edit** — text and visual editing stay bound to the same workflow.
- **Validate** — saves pass through the workflow parser/judge/conformance path.
- **Activate** — promote an accepted draft to active.

To regenerate screenshots:

```bash
cd mobile && npm run screenshots
cd bff && npm run admin-screenshots
```
