---
doc_type: plan
id: DO-POC-2026
title: "DigitalOcean online POC readiness and delivery plan"
status: active
created: 2026-08-24
updated: 2026-08-24
target_window:
  - "2026-09"
  - "2026-10"
tags: [digitalocean, poc, deployment, readiness, roadmap, support-wedge]
task: DO-POC-READINESS-001
source_of_truth:
  - docs/architecture.md
  - docs/plans/fenixcrm_strategic_repositioning_spec.md
  - docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md
  - docs/deployment-plan-digitalocean.md
---

# DigitalOcean Online POC Readiness and Delivery Plan

## Purpose

Establish a current, evidence-backed path from the repository's August 2026
state to a first externally reachable FenixCRM proof of concept on
DigitalOcean during September or October 2026.

This plan is the governing artifact for the readiness assessment and delivery
sequence. Task `DO-POC-READINESS-001` completed its evidence review on
2026-08-24; see [[audits/digitalocean_online_poc_readiness_audit]] for the
audit record and P0 findings.

## Decision Context

- The primary product wedge is Support Copilot and Support Agent with governed
  execution.
- Sales Copilot is secondary and must not expand the first online POC beyond a
  credible, demonstrable scope.
- Mobile breadth is not a release gate.
- The existing `docs/deployment-plan-digitalocean.md` is an important input,
  but its repository and external-service evidence dates from 2026-03-27 and
  must be revalidated.
- This planning task does not authorize provisioning, deployment, credential
  creation, production code changes, commits, pushes, or external mutations.

## Current Task

### DO-POC-READINESS-001 — Audit and refine the online POC plan

Status: completed on 2026-08-24

Required outputs:

1. A repository-state audit that distinguishes implemented, locally verified,
   documented-only, stale, and blocked capabilities.
2. A minimum online POC definition with explicit in-scope and out-of-scope
   behavior.
3. A DigitalOcean target architecture revalidated against current official
   product documentation and pricing.
4. A blocker and risk register with evidence, owners or decision roles, and
   mitigation paths.
5. A September/October 2026 delivery sequence with entry and exit gates,
   dependencies, and a go/no-go definition.
6. Reconciliation of the legacy DigitalOcean deployment plan and the project
   status dashboard with the new canonical assessment.

## Planned Analysis Dimensions

- Product-scope readiness for the primary support wedge.
- Backend, BFF, web/admin, mobile, and API surface readiness.
- LLM chat and embedding provider readiness.
- Container, reverse-proxy, TLS, DNS, persistence, backup, restore, and secret
  management readiness.
- Security, tenant isolation, audit, approvals, usage metering, observability,
  and operational recovery readiness.
- CI, local QA, smoke, BDD, external-validation, and post-deploy acceptance
  evidence.
- Current DigitalOcean service availability, constraints, and cost envelope.
- Calendar feasibility for September and October 2026.

## Scope Boundary

In scope:

- Read-only repository and environment inspection.
- Read-only consultation of current official DigitalOcean documentation and
  pricing.
- Documentation updates under `docs/`.

Out of scope:

- Product or infrastructure code changes.
- Creating DigitalOcean resources or accounts.
- Changing DNS, credentials, secrets, billing, or external services.
- Running a live deployment.
- Committing or pushing changes.

## Exit Criteria

This planning task is complete when the evidence supports a clear answer to
all of the following:

1. What is ready now?
2. What is partially ready or unverified?
3. What blocks the first online POC?
4. What is the smallest credible POC scope?
5. What must happen in September versus October 2026?
6. What objective go/no-go gates authorize an online launch?
7. What cost, operational, security, and vendor assumptions still require a
   human decision?

## Delivery Strategy

### Target

Deliver an invited, single-region, single-host online POC in **October 2026**.
September is the hardening and staging month. The primary demonstration is a
governed Support workflow with evidence, approvals/handoff, audit and usage,
plus externally hosted Copilot chat. Sales Copilot is secondary. Mobile is not
a launch gate.

### Sequence

| Window | Outcome | Entry gate | Exit gate |
|---|---|---|---|
| Sep W1 | Production deployment contract is corrected. | P0 audit findings accepted. | Persistence uses a host-mounted DO Volume path; embedding service starts in the normal topology; strict readiness and production env requirements are tested. |
| Sep W2 | POC operating prerequisites are decided. | Deployment contract merged and local QA green. | Owner approves budget, region, domain, operator/SSH CIDR, POC scope and recovery target; inference key and secrets delivery method exist outside Git. |
| Sep W3 | Disposable DigitalOcean staging host exists. | Prerequisites and cost cap approved. | HTTPS/DNS, firewall, volume mount, restart persistence, backup snapshot and restore drill pass. |
| Sep W4 | Live-provider and POC smoke evidence exists. | Staging is healthy. | Chosen Serverless model passes models/chat/error/latency/cost checks; support, knowledge, copilot, approval/handoff and audit/usage smoke cases pass without fixtures. |
| Oct W1 | Go/no-go and invited launch. | All P0 evidence is attached to the runbook. | Named owner accepts residual single-node risk; limited pilot access enabled; uptime and escalation ownership confirmed. |
| Oct W2+ | Observe and stabilize. | Pilot is live. | Review latency/cost/error/restore observations; either continue the pilot or open a separately approved scale/reliability workstream. |

### P0 Work Packages

1. **Deployment contract hardening.** Update the production compose/Caddy/env
   contract for mounted persistence, normal embedding startup, strict
   readiness, required production secrets/CORS, and private metrics. This is
   tracked by `DO-POC-DEPLOY-CONTRACT-001`; a local render and focused test
   pass are required before the P0 findings can be marked closed.
2. **Staging and recovery proof.** Create the DO environment only after owner
   approval; prove mount, restart persistence, snapshots/backups and a restore
   drill.
3. **Live inference and wedge smoke.** Validate the actual serverless key and
   selected model through the existing OpenAI-compatible adapter, then execute
   the explicit smoke script against HTTPS.

### Objective Go/No-Go

Go requires all P0 items in the audit to be closed with dated evidence, an
approved cost cap, and explicit acceptance of the single-host SQLite boundary.
Any failed restore, a degraded strict readiness check, exposed application
ports, default production secret, unavailable embedding service, or failed
live-provider smoke is a no-go. P1 items may be scheduled after launch only
when their residual risk is recorded and accepted by the named POC owner.

## Legacy Plan Reconciliation

`[[deployment-plan-digitalocean]]` is retained as a March 2026 historical
implementation reference. Its design direction remains broadly valid (one
Droplet, mounted Volume, Caddy, external chat and local embeddings), but its
completion checkboxes, BDD gate, exact model names/prices and claim that the
production assets implement the target topology are not current truth. This
plan and its linked audit are authoritative for September/October 2026.
