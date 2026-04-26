---
doc_type: adr
id: ADR-027
title: "DESIGN.md as agent visual context contract"
date: 2026-04-26
status: proposed
deciders: [matias]
tags: [adr, design-system, governance, agents, mobile, documentation]
related_frs: [FR-300]
related_tasks: [design-md-adoption-plan]
---

# ADR-027 — DESIGN.md as agent visual context contract

## Status

`proposed`

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
