---
doc_type: adr
id: ADR-027
title: "DESIGN.md as agent visual context contract"
date: 2026-04-26
status: accepted
deciders: [matias]
tags: [adr, design-system, governance, agents, mobile, documentation]
related_frs: [FR-300]
related_tasks: [design-md-adoption-plan]
---

# ADR-027 — DESIGN.md as agent visual context contract

## Status

`accepted`

## Context

FenixCRM has an implemented mobile Command Center dark theme in `mobile/src/theme/*`.
Those runtime tokens define the current visual system: dark operational surfaces,
operator blue primary actions, amber AI signal accents, semantic status colors,
Roboto interface typography, monospace data fields, and border-based cards.

Future UI work needs persistent design context that agents can read before changing
mobile screens, reusable components, design-sensitive documentation, or other
frontend surfaces. Without a compact design contract, agents may infer visual intent
from isolated screenshots, old scaffold files, or one-off component styles.

The root `DESIGN.md` adoption documents the current implemented theme. It is not a
redesign and does not replace the runtime source of truth in `mobile/src/theme/*`.

## Decision

Adopt root `DESIGN.md` as the preferred persistent visual-context contract for agents
and humans working on FenixCRM UI surfaces.

Agents must consult `DESIGN.md` before making visual changes to mobile screens,
frontend surfaces, reusable UI components, design-system tokens, brand-sensitive
documentation, or other user-facing presentation layers.

`DESIGN.md` is not required for backend-only changes, CLIs, data migrations,
infrastructure scripts, service internals, tests with no UI impact, or libraries with
no user-facing presentation surface.

The initial adoption is documentation-only. It does not change mobile runtime tokens,
screen styling, navigation, API behavior, or product workflows.

## Consequences

Future UI work has a stable first-read design contract. Agents can inspect
`DESIGN.md` for colors, typography, spacing, radii, component conventions, and
practical constraints before editing frontend or mobile surfaces.

`mobile/src/theme/*` remains the runtime source of truth. If `DESIGN.md` conflicts
with implemented tokens in `mobile/src/theme/colors.ts`, `typography.ts`,
`spacing.ts`, or `semantic.ts`, the runtime token wins and `DESIGN.md` must be
corrected unless the task explicitly changes the runtime theme first.

Intentional runtime design-token changes must update `DESIGN.md` in the same turn so
future agents do not inherit stale visual guidance.

This ADR does not make the alpha public `DESIGN.md` convention a hard dependency for
backend-only work. It applies where visual context matters.

## Conflict Policy

Resolve visual-context conflicts in this order:

1. Implemented runtime tokens and helpers in `mobile/src/theme/*`.
2. Root `DESIGN.md`.
3. Existing component usage patterns in mobile screens and reusable UI components.
4. Screenshots, mockups, and older plan prose.

When a conflict affects design governance, update this ADR or create a follow-up ADR
instead of leaving the rule implicit.
