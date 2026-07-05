---
doc_type: plan
id: ui-ux-governed-console-strategy
title: "Mobile UI/UX Strategy: The Verifiable-Trust Operator App"
status: active
owner: product-ui
created: 2026-07-05
accepted: 2026-07-05
tags: [plan, ui, ux, mobile, wedge, governance, verifiable-trust]
primary_refs:
  - docs/plans/fenixcrm_strategic_repositioning_spec.md
  - docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md
  - docs/plans/mobile_wedge_harmonization_plan.md
  - docs/plans/ui_redesign_command_center_dark_theme.md
  - docs/architecture.md
  - DESIGN.md
---

# Mobile UI/UX Strategy Plan — The Verifiable-Trust Operator App

> **Status**: Active (accepted 2026-07-05). Canonical strategy for mobile UI/UX; execution proceeds one task at a time per Section 9.
> **Date**: 2026-07-05
> **Focus**: **Mobile is the primary UI/UX surface.** All primary design and functional investment targets the React Native app (`mobile/`).
> **Design stance**: **Inspired by Salesforce, not a copy of it.** We learn from what mature CRM consoles got right and — more importantly — from what they structurally cannot do, then invert the model where Fenix can be better.
> **Audience**: Product, Architecture, and implementation coding agents (Codex GPT, Claude Sonnet, or equivalent)
> **Precedence rule**: subordinate to `docs/architecture.md`, the strategic repositioning spec, `docs/plans/mobile_wedge_harmonization_plan.md`, and the ADRs in Section 2. On conflict, those win and the drift must be flagged.
> **RRI note (plan authoring)**: scored 10 (Low band) via `python3 scripts/rri.py` — docs-only artifact, executed directly per `docs/policies/HITL_AUTONOMY_POLICY.md`.

---

## 0. The Thesis — Inspired by Salesforce, Inverted for Trust

### 0.1 What we take from Salesforce (inspiration)

Salesforce got real things right that we keep:

- **Orientation speed**: a good console lets an operator understand a situation in seconds (highlights, status at a glance, chronological context).
- **Action in context**: the operator acts without leaving the screen they are reasoning on.
- **Progressive disclosure**: summary first, detail on demand.

### 0.2 What Salesforce structurally cannot fix (the opening)

Salesforce is **record-first**: it is a system of record, so everything orbits the object (Account → Contact → Case). Its AI (Einstein) is a *guest panel* inside a record — it suggests, but the operator cannot cheaply verify *why*, at what confidence, at what cost, or under which policy. The result, and the exact pain our ICP reports (spec §4.3): **"agents and copilots are not trusted due to hallucination risk"** and **"AI initiatives stall because governance is weak."** Salesforce cannot invert this without cannibalizing its own system-of-record business.

### 0.3 The inversion (how Fenix improves on it)

Fenix explicitly is **not** the system of record (spec §9.1 — context may be native or external). That freedom lets us invert the model:

> **The primary unit of the app is not the record. It is the governed decision.**

The operator does not come to *administer objects*. They come to **exercise judgment over what the AI proposed — with the evidence, the confidence, the cost, and the policy state already in front of them.** The record (case/deal) is *supporting context that opens from a decision*, not the starting point. This is the moat made into a UX (CLAUDE.md: "evidence, approval, audit, controlled execution"; ADR-019: CRM is context, not the moat).

### 0.4 The commercial through-line: Verifiable Trust

The single differentiating promise, chosen because it maps directly to the ICP's #1 buying blocker (spec §4.3):

> **Verifiable Trust** — every AI output is trustworthy *because* its evidence, confidence, cost, and governance state travel *inline with the output itself*, and every governed state (abstained, denied, awaiting approval, handed off) is a designed, explainable screen — never an error.

Two design consequences carry this promise (they are *how Verifiable Trust feels*, not separate goals):

- **Speed of judgment**: everything needed to decide fits on one screen; the operator moves from "AI proposed" to "human decided with criteria" in seconds, no navigation.
- **Zero opacity**: no output ever appears without its *why*; where Salesforce/Einstein shows a suggestion, Fenix shows the suggestion **plus its governed reasoning chain**, traceable and replayable.

### 0.5 One-sentence north star

**On every screen, the AI earns the operator's trust in the moment: its evidence, confidence, cost, and policy state are inline with the output, and every governed outcome is a designed decision surface — so the operator judges in seconds instead of second-guessing.**

---

## 1. Governance Caveat (carry it, don't hide it)

ADR-022 and the spec (§6.3, §10.6) classify mobile as an *optional, non-differentiating* delivery surface — not a launch gate. Making mobile the UI/UX focus is a legitimate product-experience decision, but it does **not** silently overturn ADR-022's commercial-priority stance.

- **Recorded here**: mobile is the primary *experience* surface for delivering Verifiable Trust; the backend runtime (retrieval, evidence, policy, approvals, audit, metering) remains the moat and the system of truth. Mobile renders those contracts; it does not become the system of record.
- **Requires a new ADR (UIX-00)**: if the intent is to make mobile a *hard release/commercial gate* (amend/supersede ADR-022), that is a decision record, sequenced first in Section 6. Until UIX-00 is accepted, mobile is the primary experience surface, not a blocking commercial gate.

---

## 2. Governing Constraints (fixed — do not re-open)

| Constraint | Source | Effect on this plan |
|---|---|---|
| Governed AI layer, not CRM suite | ADR-019, spec §6 | No new CRM object breadth; the app serves the support/sales wedge only |
| Mobile surface = exactly 5 tabs (Inbox, Support, Sales, Activity, Governance) | `mobile_wedge_harmonization_plan.md` §2.1 | No new top-level surfaces; legacy screens stay hidden (`href: null`) |
| Mobile commercial priority is behind wedge validation | ADR-022, spec §6.3/§10.6 | See §1 + UIX-00; UX focus ≠ commercial release gate without a new ADR |
| Dark Command Center theme is the as-built visual system | `ui_redesign_command_center_dark_theme.md` (completed), `DESIGN.md` | Reuse `mobile/src/theme/*` tokens; **no new palettes, radii, or type scales** |
| `DESIGN.md` is the agent-facing visual contract | ADR-027 | Every visual task starts from `DESIGN.md` + `mobile/src/theme/*` |
| Evidence Pack v1, approval FSM, agent outcome enum are locked contracts | `docs/architecture.md` §8.2–8.4 | UI renders these verbatim; it never invents states or fields |
| BFF is a thin proxy: zero business logic, zero DB access; aggregation allowed | ADR-009, ADR-025 | Any missing mobile data is a Go API (or BFF aggregation) task — never client-side faking |
| BDD behavior lives in `features/*.feature`; mobile variants exist (`*-mobile.feature`) | ADR-018 | Presentation-only tasks add no scenarios; behavior-adding tasks extend the relevant `*-mobile.feature` |
| Never remove `testID` props; E2E (Detox) + Maestro screenshots must stay green | redesign plan "What NOT to Change" | Structural edits preserve all `testID`s |
| Task file discipline, RRI gating, peer review, one task at a time | `CLAUDE.md`, `AGENTS.md`, `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` | Section 9 is the execution protocol |

---

## 3. Design Principles (Verifiable Trust, made testable)

1. **Inline provenance, never a side panel.** Evidence, confidence, cost, and policy state render *with* the AI output in the same visual unit — not in a separate tab the operator must go find. This is the concrete inversion of Einstein's guest-panel model.
2. **Governed outcomes are designed decision surfaces.** `abstained`, `denied_by_policy`, `awaiting_approval`, `handed_off` each get a dedicated treatment (semantic color from `mobile/src/theme/semantic.ts`, plain-language explanation, and the *next human action* — "Review policy", "Take over", "Escalate"). None is an error toast.
3. **Decide-in-seconds.** Any decision screen must carry everything the operator needs to judge without navigating away: the proposal, its evidence, its risk/confidence, its cost, its policy state, and the decision affordance — one screen.
4. **Trace continuity in thumb's reach.** Every run, approval, and answer shows its `trace_id` (monospace). One tap deep-links into Governance → Audit filtered by that trace: from "the AI said X" to the immutable record of *why* in a single gesture. This is what "replayable" means on a phone.
5. **Work-first home = decision queue.** The home is the Inbox: decisions awaiting judgment (approvals, handoffs, low-confidence outputs, denials), ordered by urgency, badged on the tab bar (already wired via `useInbox`). Records open *from* a decision, never from an object launcher.
6. **Cost is a trust signal, not an afterthought.** Tokens/cost/latency render in monospace on every AI surface (per `DESIGN.md` data-code). Visible cost is part of why the operator trusts the system is governed, and it backs the <€0.10/interaction NFR and per-run commercial attribution.
7. **Dense, operational, dark, thumb-friendly.** Border-separated dark panels, Roboto type, monospace data, bottom-sheet detail, generous tap targets, no hero sections, no decorative gradients — built for an ops user clearing 40 decisions, not for a demo.

---

## 4. Salesforce → Fenix: Inspiration, Then Inversion

For each pattern: what Salesforce does, what it *cannot* do, and how Fenix improves rather than copies.

| Concern | Salesforce (inspiration) | Its structural limit | Fenix inversion (the improvement) | Where |
|---|---|---|---|---|
| **Home** | Object lists / record home | Starts on *data*, not on *what needs a human* | **Decision queue**: home is the judgment backlog, ordered by urgency | Inbox (UIX-21) |
| **AI in context** | Einstein side panel suggests | Suggestion without inline, verifiable *why* | **Trust unit**: answer + evidence + confidence + cost + policy in one inseparable block | CopilotPanel (UIX-11), EvidenceCard (UIX-10) |
| **Record orientation** | Highlights Panel | The record is the center of gravity | Highlights are **supporting context reached from a decision**, not the destination | Record screens (UIX-22, 31) |
| **Process visibility** | Path component (sales stages) | Cosmetic stage bar; no governance semantics | **Governed FSM path**: approval states as a deterministic, audited stepper | ApprovalPath (UIX-12) |
| **Trust in AI actions** | Einstein Trust Layer (backstage) | Not operator-facing at decision time | **Run Inspector**: operator-facing flight recorder of one governed run | Run detail (UIX-14) |
| **Auditability** | Field history / setup logs | Buried, admin-only, not linked to the AI claim | **One-tap trace continuity** from any AI output to its immutable record | Audit deep-link (UIX-15) |

### What we deliberately refuse to copy (breadth traps)

App launcher / N-object nav, custom-object CRUD screens, reports & dashboards suite, Chatter feed, offline record-editing engine. All contradict the wedge (five fixed tabs, spec §6.2) and pull toward "another Salesforce."

### The one screen Salesforce cannot cheaply answer back

The **mobile Run Inspector** (UIX-14): a governed flight recorder — trigger → evidence pack → policy decisions → tool calls → approval gate → outcome → audit → cost — on a phone. Einstein's Trust Layer has no operator-facing equivalent at this granularity. This screen *is* the category argument, and it is the commercial demo centerpiece.

---

## 5. Surface Strategy and Contracts

### 5.1 Investment order (all mobile), driven by the thesis

1. **The trust unit** — evidence card + copilot governed states. Every decision surface reuses these; they *are* Verifiable Trust in component form. (Phase 1)
2. **The decision loop** — approval FSM/detail + decision + run inspector + audit continuity. This is the governed-decision unit made whole. (Phase 2)
3. **Supporting context** — support case record + sales record/brief, with the trust unit embedded. Records serve decisions, so they come last. (Phase 3)

### 5.2 One visual system (already in place — reuse, don't recreate)

Command Center dark theme is as-built (`ui_redesign_command_center_dark_theme.md` completed; tokens in `mobile/src/theme/{colors,typography,spacing,semantic}.ts`; documented in `DESIGN.md`). **This plan adds no new tokens.** It fills gaps in *how existing tokens render governed data*, reusing helpers (`getAgentStatusColor`, `getConfidenceColor`, `confidenceGlowStyle`).

### 5.3 Current mobile state (audited 2026-07-05)

- Five wedge tabs wired (`app/(tabs)/_layout.tsx`); Inbox already **interleaves approvals + handoffs + signals + rejected runs** (`inbox/index.tsx`, `normalizeItems`) and badges the tab (`useInbox`). **The decision-queue infrastructure already exists — it is not yet framed or prioritized as the product thesis.**
- `CopilotPanel` has SSE + `EvidenceCard` list + `ActionButton`s. **Gap:** no confidence tier, no warnings row, no designed abstention state → the "trust unit" is incomplete.
- `EvidenceCard` shows title/snippet/score/timestamp. **Gap:** no retrieval method, provenance, or PII/stale flags from Evidence Pack v1.
- `ApprovalCard` **already ships approve/reject** with a required-reason dialog (UC-A7/B6, FR-071), wired via `useApproveApproval`/`useRejectApproval` → `approvalApi` — the mobile decision write path exists today. **Gap:** no five-state FSM path, no policy-explanation block, no post-decision feedback (terminal state + audit link), and API conflicts (already decided, expired) surface as a generic error line instead of designed states.
- `activity/[id].tsx` (~302 lines) is run detail. **Gap:** not yet the full flight recorder.
- Governance tab has audit + usage + filter bar. **Gap:** no `trace_id` deep-link from runs/approvals into a trace-filtered audit view.

### 5.4 Contracts the UI renders (verbatim — do not invent fields)

- **Evidence Pack v1** (`docs/architecture.md` §8.2): `schema_version, query, source_count, dedup_count, filtered_count, confidence, warnings, retrieval_methods_used, built_at, sources[] {snippet, relevance_score, retrieval_method, pii_redacted, source_timestamp, provenance {source_type, source_system, source_object_id}}`.
- **Approval FSM** (§8.3): `pending → approved | rejected | expired | cancelled` (terminals). Legacy `denied` normalizes to `rejected`.
- **Public agent outcomes** (§8.4): `completed, completed_with_warnings, abstained, awaiting_approval, handed_off, denied_by_policy, failed` (+ transient `running`). Color map already in `mobile/src/theme/semantic.ts` — reuse exactly.
- **Usage event** (§8.5): render `model_name, input_units, output_units, estimated_cost, latency_ms` in monospace.

**If a screen needs a field the API does not expose** (architecture §8.2 notes richer per-source provenance as follow-up): **stop and report** — propose a Go API task; do not fabricate it client-side. Fabricated provenance would be the exact opposite of Verifiable Trust.

---

## 6. Functional Scope by Phase — Ordered Task Graph

| ID | Task | Type | Hard deps | Unblocks | Criticality (advisory) |
|---|---|---|---|---|---|
| `UIX-00` | ADR-034: elevate mobile to primary UX surface (amends ADR-022 stance) | ADR (docs-only) | none | frames the plan | standard |
| `UIX-10` | `EvidenceCard` → truthful evidence trust fields (method, PII, knowledge item id) | dev (mobile) | none | UIX-11, UIX-14, UIX-22 | standard |
| `UIX-10A` | Copilot SSE evidence contract unblock for per-source trust fields | dev (go/mobile contract) | none | UIX-10, UIX-11, UIX-14, UIX-22 | standard |
| `UIX-11A` | Copilot trust SSE contract unblock for confidence, warnings, abstention metadata | dev (go/mobile contract) | none | UIX-11 | standard |
| `UIX-11` | `CopilotPanel` trust unit: confidence badge, warnings row, abstention state | dev (mobile) | UIX-10, UIX-11A | UIX-22, UIX-31 | standard |
| `UIX-12` | Approval FSM path stepper + policy-explanation block | dev (mobile) | none | UIX-13, UIX-21, UIX-31 | standard |
| `UIX-13` | Harden the existing decide flow: post-decision FSM + audit link, optional approve comment, designed conflict states | dev (mobile) | UIX-12 | wedge demo credibility | **critical candidate** (governance write surface) |
| `UIX-14` | Run Inspector: evidence + tools + approval + handoff + usage flight recorder | dev (mobile) | UIX-10 | UIX-15 | standard |
| `UIX-15` | Trace continuity: `trace_id` deep-links from runs/approvals → filtered audit | dev (mobile) | UIX-14 | — | standard |
| `UIX-21` | Inbox as decision queue: grouped by urgency, outcome accents, empty state | dev (mobile) | UIX-12 | — | standard |
| `UIX-22` | Sales record + brief: context reached from a decision; trust unit embedded | dev (mobile) | UIX-10, UIX-11 | — | standard |
| `UIX-31` | Support case record: context reached from a decision; trust unit + approval | dev (mobile) | UIX-10, UIX-11, UIX-12 | — | standard |

**Critical path**: `UIX-00 → UIX-10 → UIX-11A → UIX-11`; in parallel `UIX-12 → UIX-13`; then `UIX-14 → UIX-15`; then `UIX-21`, and record screens `UIX-22`, `UIX-31`. (`UIX-00` frames priority for the whole plan but hard-blocks no other task; it goes first because it is cheap and settles the ADR-022 tension before code lands.)

**Status update (2026-07-05)**: `UIX-10A` completed and unblocked `UIX-10`. `UIX-10` is now completed: `EvidenceCard` renders the currently truthful per-source trust fields (`retrieval_method`, `pii_redacted`, stable mobile `timestamp`, and `knowledge_item_id`) in expanded state without fabricating provenance or stale markers that the backend does not yet assert per source. During `UIX-11` execution we found a second contract gap: the copilot stream did not expose truthful `confidence`/`warnings` metadata to mobile and the mobile SSE parser discarded terminal metadata that already carried `answer_type`/`abstention_reason`. `UIX-11A` is now completed: the Go copilot stream emits truthful evidence-pack trust metadata and preserves done-event outcome metadata, while the mobile SSE parser and message shaping retain those fields for downstream rendering. `UIX-11` is now completed: `CopilotPanel` renders the inline trust unit with truthful confidence tier badge, warnings row, and designed abstention panel while preserving streaming semantics, `testID`s, evidence reuse, and `onSupportTrigger`. `UIX-12` is now completed: approval cards render the governed five-state FSM stepper (`pending`, `approved`, `rejected`, `expired`, `cancelled`), normalize legacy `denied` to `rejected` for display only, and show a compact policy explanation block only when machine-readable policy metadata is already present in `approval.payload`, without changing approval action wiring or existing `testID`s. `UIX-13` is now completed: the mobile approval write surface keeps the existing backend decision authority and transport semantics while adding an optional approval comment dialog, per-approval terminal/conflict feedback in the inbox, terminal `ApprovalPath` rendering after successful decisions, and designed expired/already-decided states with an audit-trail affordance when trace metadata is present. `UIX-14` is now completed: the activity run detail screen has been refactored into a modular Run Inspector that surfaces truthful highlights, evidence-pack metadata when present, tool activity, awaiting-approval linkage, handoff payload retrieval, reasoning trace, outcome payload, and per-run usage, while routing trace visibility to the existing audit screen without fabricating the filtered deep-link reserved for `UIX-15`.

**Explicitly deferred (do not schedule)**: any new top-level tab, CRM object CRUD screens, offline sync, dashboards suite, web/BFF surfaces, builder screens on mobile.

---

## 7. Task Specifications

Each task still requires its own `docs/tasks/task_uix-*.md` (with a `**Plan**:` link here, High-Level Pseudocode for dev tasks, RRI run, peer readiness review) **before any code is written**. These specs seed those files; they do not replace them.

### UIX-00 — ADR-034: mobile as primary UX surface

- **Files**: `docs/decisions/ADR-034-*.md`, `docs/decisions/README.md`
- **Scope**: Record that mobile is the primary UX surface for delivering the wedge experience (Verifiable Trust), amending ADR-022's *experience-priority* aspect while preserving its backend-moat stance. State explicitly whether mobile becomes a hard release gate (if yes, supersede ADR-022's release-gating bullet; if no, only re-scope UX priority). Cross-reference `mobile_wedge_harmonization_plan.md`.
- **BDD impact**: none. **QA**: `python3 scripts/check_okf_frontmatter.py`; update `docs/decisions/README.md`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Low

### UIX-10 — EvidenceCard → Evidence Pack truthful subset (the trust unit's proof)

- **Files**: `mobile/src/components/copilot/EvidenceCard.tsx`; `EvidenceSource` type in `mobile/src/services/sse.ts` if it lacks v1 fields; tests
- **Scope**: In the expanded state, render only the currently truthful per-source trust fields exposed by UIX-10A when present: `retrieval_method` (chip), `pii_redacted` ("PII redacted" flag), and the authoritative `knowledge_item_id` (monospace). Collapsed row unchanged; preserve `testID`s. `provenance.*` and `stale_knowledge_item` remain explicit stop-and-report contract gaps until a backend source asserts them truthfully.
- **Pseudocode**:
  ```
  collapsed: [index] title + score (unchanged)
  expanded: snippet + timestamp(mono)
    + retrieval_method ? <Chip>
    + pii_redacted ? <Flag>PII redacted</Flag>
    + knowledge_item_id ? <Text mono>knowledge item id</Text>
    + provenance/stale absent ? stop and report contract gap
  ```
- **BDD impact**: none. **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium

### UIX-10A — Copilot SSE evidence contract unblock

- **Files**: `internal/domain/copilot/chat.go`; copilot stream tests; `mobile/src/services/sse.ts`; SSE tests
- **Scope**: Unblock UIX-10 by extending the copilot SSE evidence payload with whatever per-source Evidence Pack v1 trust fields the backend can already assert truthfully. Do not fabricate provenance or stale markers that the current backend/architecture note still treats as follow-up work; expose only supported fields and document any residual gap.
- **Pseudocode**:
  ```
  inspect current knowledge evidence fields
  stream authoritative per-source fields in copilot evidence chunks
  update mobile SSE typing/parsing to match
  add focused tests for the richer evidence payload
  ```
- **BDD impact**: none. **QA**: `bash scripts/qa-go-prepush.sh` for Go changes plus `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium

### UIX-11A — Copilot trust SSE contract unblock (confidence + warnings + abstention metadata)

- **Files**: `internal/domain/copilot/chat.go`; `internal/domain/copilot/chat_test.go`; `mobile/src/services/sse.ts`; `mobile/src/hooks/useSSE.helpers.ts`; SSE/hook tests
- **Scope**: Unblock `UIX-11` by extending the truthful copilot SSE contract so the evidence event carries the Evidence Pack trust fields mobile needs (`confidence`, `warnings`, existing retrieval metadata) and the done event preserves answer outcome metadata (`answer_type`, `abstention_reason`). Update the mobile SSE parser and message shaping so those fields survive into `CopilotMessage` without changing token streaming semantics. Do not fabricate fields that the backend still does not assert truthfully.
- **Pseudocode**:
  ```
  backend evidence event:
    emit confidence + warnings from evidence pack meta
  backend done event:
    emit answer_type + abstention_reason when present
  mobile SSE:
    parse evidence meta + done meta into typed message fields
    keep token accumulation behavior unchanged
  ```
- **BDD impact**: none. This is a contract-enablement task; user-visible behavior is covered by `UIX-11`. **QA**: `bash scripts/qa-go-prepush.sh` plus `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium-High

### UIX-11 — CopilotPanel trust unit (answer + confidence + warnings + abstention)

- **Files**: `mobile/src/components/copilot/CopilotPanel.tsx`; copilot panel tests
- **Scope**: The answer footer becomes the inseparable trust block: **confidence tier badge** (`getConfidenceColor`), **warnings row** (from the pack), and a designed **abstention state** — on abstain (zero usable sources / abstain signal) render a panel (abstain reason + "Escalate / handle manually" affordance + the insufficient evidence found), never an empty bubble. Reuse `EvidenceCard` (post UIX-10) and consume only the truthful fields surfaced by `UIX-11A`. Preserve SSE streaming, `testID`s, and `onSupportTrigger`.
- **Pseudocode**:
  ```
  Footer(lastAssistant):
    if abstained: AbstentionPanel(reason, foundEvidence, escalateCTA)
    else: ConfidenceBadge(confidence) + (warnings ? WarningsRow) + EvidenceCard[] + ActionButton[]
  ```
- **BDD impact**: abstention covered by `uc-c1`/`uc-s1` at API level; add a mobile scenario to `uc-s1-sales-copilot-mobile-smoke.feature` only if the panel adds observable behavior. Decide in the task file. **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium-High. If RRI ≥ 56, split: (1) confidence+warnings, (2) abstention panel.

### UIX-12 — Approval FSM path + policy explanation (the governed process, made visible)

- **Files**: new `mobile/src/components/approvals/ApprovalPath.tsx`, `mobile/src/components/approvals/ApprovalCard.tsx`; tests
- **Scope**: Compact horizontal five-state stepper (`pending → approved | rejected | expired | cancelled`), current highlighted, terminals dimmed, colors from `mobile/src/theme/semantic.ts` (map legacy `denied → rejected`). Add a policy-explanation block rendering the machine-readable policy decision when present. No navigation/data-shape changes; preserve `testID`s.
- **Pseudocode**:
  ```
  ApprovalPath(status): nodes=[pending,approved,rejected,expired,cancelled]
    normalize denied->rejected; highlight active; dim unreached terminals
  ApprovalCard: header + expiry border + <ApprovalPath/> + (policyContext ? <PolicyExplanation/>)
  ```
- **BDD impact**: none. **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **System context note**: `ApprovalPath` is a new component → include the ASCII system-context diagram in the task file (upstream: approval payload; downstream: `ApprovalCard`, inbox detail; invariant: legacy `denied` normalized before render).
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium

### UIX-13 — Harden the existing decide flow (the judgment act, completed)

- **Reality check (2026-07-05)**: approve/reject from the app **already exists** — `ApprovalCard` renders both actions with a required-reason dialog on reject, wired via `useApproveApproval`/`useRejectApproval` → `approvalApi` (UC-A7/B6, FR-071). This task does **not** create the write path; it completes the judgment loop around it. ADR-034 (UIX-00) does not gate this task.
- **Files**: `mobile/src/components/approvals/ApprovalCard.tsx`, `mobile/app/(tabs)/inbox/index.tsx`, `mobile/src/hooks/useWedge.ts` (post-decision handling only); tests
- **Scope**: Three hardening moves, no transport changes: (1) **post-decision feedback** — on success, render the reached terminal FSM node (reuse `ApprovalPath` from UIX-12) plus a `trace_id` link toward the audit view (basic navigation now; upgraded to the filtered deep-link by UIX-15); (2) **optional comment on approve** — `approvalApi.approve(id, reason)` already accepts a reason; expose an optional comment field so approvals carry rationale like rejections do; (3) **designed conflict states** — deterministic API errors (already decided, expired) render as governed states with explanation, replacing the generic `actionError` line. **No client-side business logic** — the Go API decides; the app relays and renders.
- **Pseudocode**:
  ```
  onDecide(id, decision, comment?):
    approvalApi.approve|reject(id, comment)   // existing transport, unchanged
    200 -> re-render item: <ApprovalPath status=terminal/> + audit link
    409/410-style conflict -> ConflictState(reason from API error code)
  ```
- **BDD impact**: decision behavior is specified in `uc-a7-human-override-and-approval.feature` (no mobile-variant file exists today); add a mobile scenario only if this task adds observable behavior beyond the API (the approve-comment relay likely qualifies — decide in the task file). **QA**: `bash scripts/qa-mobile-prepush.sh` + manual audit-emission check via `GET /api/v1/audit/events`.
- **Criticality**: `critical` candidate — it modifies an existing governance write surface (`auth_security` RRI penalty). Expect RRI 26+ → task card + explicit human approval before editing.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium

### UIX-14 — Run Inspector (the category-defining flight recorder)

- **Files**: `mobile/app/(tabs)/activity/[id].tsx`, new `mobile/src/components/runs/` set; tests
- **Scope**: Run detail becomes the flight recorder: highlights (agent, trigger type, outcome badge via the exact outcome color map, `trace_id` mono) → evidence pack (v1 fields incl. `confidence`, `warnings`, `retrieval_methods_used`, reusing `EvidenceCard`) → tool calls → approval linkage (if `awaiting_approval`) → handoff payload (if `handed_off`) → usage strip (`input_units / output_units / estimated_cost / latency_ms`, mono). Data from `GET /api/v1/agents/runs/{id}`, `GET /api/v1/usage?run_id=`, handoff endpoint. Client composes for display only.
- **Pseudocode**:
  ```
  useRunDetail(id): {run, usage(run_id), handoff(if handed_off)}
  render Highlights(outcome_badge, trace_id) + EvidenceSection + ToolCalls
       + (awaiting_approval ? ApprovalLink) + (handed_off ? HandoffPayload) + UsageStrip(mono)
  trace_id onPress -> /governance/audit?trace_id=...
  ```
- **BDD impact**: none (read-only; aligns with `uc-c1`/`uc-d1-*-mobile`). **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **System context note**: introduces a `runs/` component set → include the ASCII diagram in the task file (upstream: run/usage/handoff endpoints; downstream: audit deep-link; invariant: no client-side data mutation).
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: High. If RRI ≥ 56, split by section.

### UIX-15 — Trace continuity

- **Files**: `mobile/app/(tabs)/governance/audit.tsx`, `mobile/src/components/governance/{AuditFilterBar,AuditEventCard}.tsx`; deep-link params in runs/approvals screens; tests
- **Scope**: `trace_id` (and actor/action) filter param on the audit screen; `trace_id` becomes a tappable mono link on run detail (UIX-14) and approval detail → navigates to the filtered audit view. Outcome-colored left-border accents on audit rows (reuse the color map; partially present per completed redesign T13).
- **BDD impact**: none. **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4-mini | Anthropic: claude-sonnet-4-6` · Effort: Low-Medium

### UIX-21 — Inbox as decision queue (the thesis, made the home)

- **Files**: `mobile/app/(tabs)/inbox/index.tsx`, `mobile/src/components/inbox/*`; tests
- **Scope**: Reframe the existing interleaved feed as an explicit decision queue, grouped by the **four item types the inbox payload already carries** (`extractInboxItems`: approvals, handoffs, signals, rejected runs): "Awaiting your approval" (expiry-ordered), "Handoffs waiting", "Signals to review" (confidence-ordered), "Denied runs" — each with a count eyebrow; outcome/expiry accent stripes (reuse UIX-12 path colors + completed redesign handoff/rejected styles); designed empty state ("No decisions waiting"). Keep the existing per-group sorting semantics, the `useInbox` badge wiring, and all `testID`s. This is reframing + grouping over the existing payload — no data-model change, no invented categories.
- **BDD impact**: none. **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium

### UIX-22 — Sales record + brief (context reached from a decision)

- **Files**: `mobile/app/(tabs)/sales/[id].tsx`, `mobile/app/(tabs)/sales/deal-[id].tsx`, `mobile/src/components/sales/*`; tests
- **Scope**: Record anatomy as *supporting context*: highlights (account/deal, stage chip, figures via `typography.monoLG`), deal-stage Path, timeline tab. Embed the upgraded `CopilotPanel` (post UIX-11) and render the sales brief (`POST /api/v1/copilot/sales-brief`) via `SalesBriefContent` with per-section evidence citations, confidence, and the designed abstention state when evidence is insufficient. Usage footer as in the run inspector.
- **BDD impact**: none new (`uc-s1`/`uc-s1-sales-copilot-mobile-smoke`). **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium

### UIX-31 — Support case record (context reached from a decision)

- **Files**: `mobile/app/(tabs)/support/[id].tsx`, `mobile/src/components/support/SupportCaseDetailContent.tsx`; tests
- **Scope**: Record anatomy as supporting context: highlights (subject, status chip, account/contact/owner), status Path, timeline tab (`timeline_event` + activities/notes), evidence tab (knowledge linked to the case). Embed the upgraded `CopilotPanel` (support-trigger callback intact) with the trust unit + approval FSM (UIX-12) when a support run enters `awaiting_approval`. Read-only over existing case/timeline endpoints; missing-endpoint rule applies.
- **BDD impact**: none (`uc-c1`/`uc-c1-*`). **QA**: `bash scripts/qa-mobile-prepush.sh`.
- **Model**: `OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6` · Effort: Medium

---

## 8. BDD and Traceability Map

| Task | Feature file(s) | New scenarios? |
|---|---|---|
| UIX-10/12/14/15/21/22/31 | none | No — presentation/composition over locked contracts |
| UIX-11 | `features/uc-s1-sales-copilot-mobile-smoke.feature`, `features/uc-c1-support-agent.feature` (ref) | Only if the panel adds observable behavior; decide in the task file |
| UIX-13 | `features/uc-a7-human-override-and-approval.feature` (no mobile-variant file exists today) | Only if the app adds observable behavior beyond the API (approve-comment relay likely qualifies); decide in the task file |

FR anchors: FR-300/301 (mobile presentation), FR-060/070/071 (governance), FR-200/202 (copilot), FR-230/232 (support agent + handoff). No Go changes planned; if a contract gap forces one, run `cmd/frtrace` traceability on that separate Go task.

---

## 9. Execution Protocol for Coding Agents (Codex GPT / Claude Sonnet / other)

Mandatory for any agent executing a UIX task. Applies `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` to this plan — read that guide first; on conflict, the guide and `CLAUDE.md`/`AGENTS.md` win.

1. **Orient**: `README_AGENT_ORDER.md`, then `CLAUDE.md`/`AGENTS.md`, then this plan (esp. §0 thesis), then `DESIGN.md`, then `mobile_wedge_harmonization_plan.md` and the Section 2 ADRs relevant to your files.
2. **One task only**: next unblocked task from Section 6. Do not batch.
3. **Task file first**: `docs/tasks/task_uix-XX_<slug>.md` with full frontmatter (`doc_type: task`, `id`, `title`, `status`, `phase`, `week`, `tags`, `fr_refs`, `uc_refs`, `blocked_by`, `blocks`, `files_affected`, `created`, `completed`), a `**Plan**:` link here, High-Level Pseudocode (dev tasks), and a System Context diagram if a new component is introduced (UIX-12 `ApprovalPath`, UIX-14 `runs/`).
4. **Score RRI**: `python3 scripts/rri.py --auto-cc --platform rn --touches <files> ...`. UIX-13 includes `auth_security`. Use `criticality_suggested` as advisory input.
5. **Peer readiness review**: `python3 scripts/peer-workflow-review.py task-readiness --caller-kind <codex|claude-code> --task <file> --plan docs/plans/ui_ux_governed_console_strategy_plan.md --task-card <preview>`. Non-pass blocks presentation.
6. **Gate**: RRI 0–25 → execute directly (check `docs/policies/LOCAL_MODEL_POLICY.md`). RRI 26+ → present the task card (English, all required fields) and wait for explicit human approval.
7. **Guardrails**:
   - Thesis: every decision surface keeps the trust block inseparable — never split evidence/confidence/cost/policy from the AI output into a place the operator must go find.
   - Mobile: never remove `testID`; never add a top-level tab; prefer token/composition changes over structural rewrites; reuse `mobile/src/theme/*` and its helpers — no new palettes, radii, or type scales.
   - Contracts: render §5.4 enums/fields verbatim; unknown states render as a neutral "unknown" treatment — never crash, never guess.
   - Data: compose for display only. Missing field/endpoint → **stop, report, propose a separate Go/BFF task** — do not fabricate values client-side (fabrication is the opposite of the thesis).
   - No BFF business logic; new aggregation stays a thin BFF proxy (ADR-009/025).
8. **Verify**: `bash scripts/qa-mobile-prepush.sh`; `cd mobile && npm run screenshots` (Maestro) when available, inspect `mobile/artifacts/screenshots/`. If a required gate cannot run, stop and report before any push.
9. **Peer code review**: `python3 scripts/peer-workflow-review.py post-code-review ...` before closure; `--criticality critical` for UIX-13 if declared.
10. **Sync and close**: update this plan's task-graph status, the task file (`status`, `completed`), and materially affected vault artifacts (`DESIGN.md` only if a token intentionally changes — not expected; `docs/architecture.md` "Mobile visual system as-built" if a component contract changes). Report `Result / Verification / Peer code review approval / Files affected / Effort / Recommended model / Tokens`, then present the next task card and **wait**.
11. **Attribution**: before any commit, `export AI_AGENT="<model id>"` and `git config fenix.ai-agent "<model id>"`. Push only after local gates pass (`make install-hooks` once per environment).

---

## 10. Acceptance Criteria

Complete when, **on the phone**, the thesis is real:

1. Any AI answer shows evidence (method + provenance + PII/stale flags), a confidence tier, warnings when present, and renders abstention as a designed state — inline with the answer, not in a separate panel (UIX-10, UIX-11).
2. An operator reviews a pending approval with its policy explanation and five-state FSM path, decides it (with optional rationale on approve, required on reject), sees the terminal state rendered on the path, and reaches the audit record of that decision (UIX-12/13/15).
3. Any run is inspectable end-to-end on one screen — evidence, tools, approval, handoff, outcome, cost — with `trace_id` tap-through to filtered audit (UIX-14/15).
4. Support cases and sales accounts/deals are reached *from decisions*, each with highlights → path → timeline and an embedded trust unit that cites evidence, shows confidence and cost, and renders abstention as a designed state (UIX-22, UIX-31).
5. The home reads as a decision queue: grouped approvals (expiry-ordered), handoffs, signals, and denied runs, badged, with a designed empty state (UIX-21).
6. No task added a top-level tab, CRM CRUD breadth, BFF business logic, an undocumented token, or a client-fabricated contract field. `qa-mobile-prepush.sh` green; Maestro screenshots reflect the changes.

---

## 11. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Plan drifts back into "smaller Salesforce" | §0 thesis is the acceptance frame; every task ties to the trust unit or the decision loop, not to reproducing a Salesforce widget |
| Mobile focus read as overturning ADR-022 commercial priority | UIX-00 (ADR-034) records the exact scope; §1/§0.4 keep the backend as the moat |
| Scope creep toward CRM parity via record screens | Records are *context reached from decisions*; proxy existing read endpoints; missing-endpoint rule (§9.7) forces re-planning |
| Contract gaps surface as fabricated client fields | Each affected task carries a "stop and report" clause tied to §5.4; fabrication contradicts the thesis |
| Changes to the decide flow (UIX-13) regress an existing governed write surface | Write path already exists (UC-A7/B6) and stays untouched at transport level; UIX-13 is UX hardening only, declared `critical` candidate, peer-reviewed with `--criticality critical`; `uc-a7` BDD remains authoritative |
| Visual drift from one-off styles | Single token source (`mobile/src/theme/*` + `DESIGN.md`); reviewers reject new hex/scale |
| Detox `testID` / Maestro churn | Guardrail: never remove `testID`; UIX tasks own updating Maestro baselines in the same task |
