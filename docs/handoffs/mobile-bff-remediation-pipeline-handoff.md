# Mobile/BFF Remediation Pipeline Handoff

Use this prompt when reviewing CI failures after pushing the mobile/BFF/API remediation changes.

## Prompt

You are reviewing a GitHub Actions failure for the mobile/BFF/API remediation rollout.

Your job is to diagnose the failing job, propose the smallest safe fix, and avoid reverting the remediation unless the failure proves a real contract break.

Source of truth documents to reference before making any code decision:

1. `docs/mobile-bff-api-audit-remediation-plan.md`
2. `docs/tasks/task_mobile_bff_remediation_1.md`
3. `docs/tasks/task_mobile_bff_remediation_2.md`
4. `docs/tasks/task_mobile_bff_remediation_3.md`
5. `docs/tasks/task_mobile_bff_remediation_4.md`
6. `docs/tasks/task_mobile_bff_remediation_5.md`
7. `docs/mobile-agent-spec-transition-gap-closure-plan.md`
8. `docs/deployment-plan-digitalocean.md`
9. `docs/architecture.md`
10. `docs/openapi.yaml`

Working assumptions that must remain true unless the docs above are updated:

- Mobile should no longer depend on custom BFF CRM aggregation routes such as `/bff/accounts`, `/bff/deals`, `/bff/cases`, or `/:id/full`.
- BFF should remain minimal: `auth`, `copilot`, `health`, and `/bff/api/v1/*` relay.
- `GET /bff/health` should validate backend readiness through Go `/readyz`.
- CRM signal counts needed by mobile should come from the Go backend contract, not BFF aggregation.
- Local project documentation is the reference point for intent and expected architecture.

When analyzing the failure:

1. Identify the exact failing job, command, and file or package.
2. Classify the failure as one of: stale test expectation, compile/type break, contract drift, environment issue, or unrelated pre-existing failure.
3. Check whether the failure contradicts the documented target contract or only an outdated assertion.
4. Prefer fixing tests, mocks, or docs if production behavior already matches the documented target.
5. Only restore removed BFF aggregation behavior if the source-of-truth docs clearly require it.

Expected remediation boundaries:

- Safe fixes:
  - update stale mobile/BFF tests
  - update mock payloads for `active_signal_count`
  - fix OpenAPI or architecture wording drift
  - fix route wiring or handler compilation issues introduced by the remediation
- Avoid by default:
  - reintroducing `aggregated.ts`
  - moving CRM aggregation logic back into BFF
  - changing public mobile routes away from `/bff/api/v1/*` relay unless docs are revised

Output format:

- `Failure summary`
- `Why it failed`
- `Docs consulted`
- `Smallest safe fix`
- `Risk check`
- `Patch or PR summary`

