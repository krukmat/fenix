---
colors:
  brandPrimary: "#3B82F6"
  brandOnPrimary: "#FFFFFF"
  brandPrimaryContainer: "#1E3A5F"
  brandOnPrimaryContainer: "#93C5FD"
  brandSecondary: "#F59E0B"
  brandOnSecondary: "#0A0D12"
  brandSecondaryContainer: "#3D2C00"
  brandOnSecondaryContainer: "#FDE68A"
  brandError: "#EF4444"
  brandOnError: "#FFFFFF"
  brandErrorContainer: "#3B0F0F"
  brandOnErrorContainer: "#FCA5A5"
  brandBackground: "#0A0D12"
  brandOnBackground: "#F0F4FF"
  brandSurface: "#111620"
  brandOnSurface: "#E2E8F0"
  brandSurfaceVariant: "#1A2030"
  brandOnSurfaceVariant: "#8899AA"
  brandOutline: "#2E3A50"
  brandOutlineVariant: "#1E2B3E"
  brandInverseSurface: "#E2E8F0"
  brandInverseOnSurface: "#0A0D12"
  brandInversePrimary: "#3B82F6"
  semanticSuccess: "#10B981"
  semanticSuccessContainer: "#052E1C"
  semanticOnSuccessContainer: "#6EE7B7"
  semanticWarning: "#F59E0B"
  semanticWarningContainer: "#3D2C00"
  semanticOnWarningContainer: "#FDE68A"
  semanticInfo: "#60A5FA"
  semanticConfidenceHigh: "#10B981"
  semanticConfidenceMed: "#F59E0B"
  semanticConfidenceLow: "#6B7280"
typography:
  headingLG:
    fontFamily: "Roboto"
    fontSize: 22
    fontWeight: "700"
    letterSpacing: -0.3
  headingMD:
    fontFamily: "Roboto"
    fontSize: 18
    fontWeight: "600"
  eyebrow:
    fontFamily: "Roboto"
    fontSize: 11
    fontWeight: "700"
    letterSpacing: 1.2
    textTransform: "uppercase"
  labelMD:
    fontFamily: "Roboto"
    fontSize: 11
    fontWeight: "600"
    letterSpacing: 0.3
  mono:
    fontFamily:
      android: "monospace"
      ios: "Courier New"
    fontSize: 12
    fontWeight: "400"
  monoLG:
    fontFamily:
      android: "monospace"
      ios: "Courier New"
    fontSize: 14
    fontWeight: "700"
  monoSM:
    fontFamily:
      android: "monospace"
      ios: "Courier New"
    fontSize: 11
    fontWeight: "400"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  base: "16px"
  lg: "20px"
  xl: "24px"
  xxl: "32px"
rounded:
  xs: "4px"
  sm: "6px"
  md: "10px"
  lg: "14px"
  full: "999px"
components:
  screen:
    backgroundColor: "{colors.brandBackground}"
    textColor: "{colors.brandOnBackground}"
    padding: "{spacing.base}"
  card:
    backgroundColor: "{colors.brandSurface}"
    textColor: "{colors.brandOnSurface}"
    rounded: "{rounded.md}"
    padding: "{spacing.base}"
  button-primary:
    backgroundColor: "{colors.brandPrimary}"
    textColor: "{colors.brandOnPrimary}"
    rounded: "{rounded.sm}"
    padding: "{spacing.md}"
  button-secondary:
    backgroundColor: "{colors.brandSurfaceVariant}"
    textColor: "{colors.brandOnSurface}"
    rounded: "{rounded.sm}"
    padding: "{spacing.md}"
  status-chip:
    backgroundColor: "{colors.brandSurfaceVariant}"
    textColor: "{colors.brandOnSurfaceVariant}"
    rounded: "{rounded.full}"
    padding: "{spacing.xs}"
  data-code:
    backgroundColor: "{colors.brandSurfaceVariant}"
    textColor: "{colors.brandOnSurface}"
    typography: "{typography.mono}"
    rounded: "{rounded.xs}"
    padding: "{spacing.sm}"
  tab-bar:
    backgroundColor: "{colors.brandSurface}"
    textColor: "{colors.brandOnSurfaceVariant}"
    rounded: "{rounded.xs}"
    padding: "{spacing.sm}"
---

# FenixCRM Design Contract

## Overview

`DESIGN.md` is the agent-facing visual contract for FenixCRM. It documents the implemented mobile Command Center dark theme so future UI work starts from the current system instead of reinterpreting the brand from screenshots or isolated components.

The runtime source of truth remains `mobile/src/theme/*`. When this document conflicts with `mobile/src/theme/colors.ts`, `mobile/src/theme/typography.ts`, `mobile/src/theme/spacing.ts`, or `mobile/src/theme/semantic.ts`, update `DESIGN.md` to match the runtime tokens unless a task explicitly changes the runtime design system first.

The current visual direction is operational and dense: dark command surfaces, border-based separation, operator blue primary actions, amber AI signal accents, semantic status colors, Roboto interface type, and monospace data fields.

## Colors

Use `brandBackground` for the app canvas, `brandSurface` for cards and primary panels, and `brandSurfaceVariant` for chips, code fields, and secondary surfaces. These map to `brandColors.background`, `brandColors.surface`, and `brandColors.surfaceVariant` in `mobile/src/theme/colors.ts`. Use `brandOutlineVariant` for hairline card borders and `brandOutline` when a control needs a stronger boundary.

`brandPrimary` is the operator blue for primary actions, active navigation, selected tabs, and approval-ready states. `brandSecondary` is the amber signal color for AI or warning-adjacent emphasis; it should not replace semantic warning states when `semanticWarning` is the clearer token.

Use `semanticSuccess`, `semanticWarning`, `semanticInfo`, and the confidence tokens for status-driven meaning. These map to `semanticColors.*` in `mobile/src/theme/colors.ts`. Do not infer new status colors from screenshots when `mobile/src/theme/semantic.ts` already provides a helper or mapping.

## Typography

Use Roboto for interface hierarchy, labels, actions, and compact metadata. `headingLG` is for screen-level headings, `headingMD` is for section and card headings, `eyebrow` is for uppercase operational labels, and `labelMD` is for compact labels and control text.

Use the monospace tokens for identifiers, numeric facts, timestamps, trace IDs, confidence values, JSON snippets, quota values, and other data that benefits from stable glyph widths. The runtime monospace family is platform-aware: Android uses `monospace`; iOS uses `Courier New`.

Keep text dense and scannable. Do not introduce oversized marketing-style headings inside dashboards, operational cards, tab screens, or repeated mobile components.

## Layout

Use the spacing scale from `spacing` for all padding and gaps. Prefer `base` for standard screen and card padding, `md` for compact internal groups, `sm` for tight control spacing, and `lg` to `xxl` only when separating major screen regions.

Use dark surfaces with border-based separation instead of raised light-theme shadows. Cards should read as operational panels on the command canvas: `surface` background, `outlineVariant` border, and predictable internal spacing.

Layouts should remain compact enough for repeated sales, governance, workflow, and activity scanning. Avoid decorative page sections, nested cards, and large empty hero-style areas in app surfaces.

## Shapes

Use `rounded.xs` for small data fields and inline code containers, `rounded.sm` for buttons and compact controls, `rounded.md` for cards and standard panels, and `rounded.lg` only for larger surfaces that need more softness. Use `rounded.full` for pills, chips, badges, and circular affordances.

The default card shape is a modest radius with a 1px border. Do not increase radii to make the product feel playful; FenixCRM should remain precise, operational, and work-focused.

## Components

Screens use the dark canvas and standard `base` padding. Cards use `surface`, `onSurface`, `outlineVariant`, and `rounded.md`; they should frame one repeated entity or one coherent operational panel.

Primary buttons use operator blue for the decisive action in a local workflow. Secondary buttons use dark surfaces and borders when the action is available but not dominant.

Status chips should stay compact and semantic. Use helper-driven status colors from `mobile/src/theme/semantic.ts` when state-specific meaning matters; use the neutral chip component token for non-semantic filters, labels, or metadata.

Data code surfaces use monospace type, dark secondary surface, and a border. They are appropriate for IDs, trace details, timestamps, JSON, quota numbers, and values that operators compare visually.

The tab bar uses dark surface separation, muted inactive labels, and operator blue for the active destination. It should support scanning and orientation, not become a decorative brand band.

## Do's and Don'ts

Do read `DESIGN.md` before frontend or mobile visual changes. Do verify token values against `mobile/src/theme/*` when a task touches design primitives. Do update this document in the same change when runtime design tokens intentionally change.

Do keep app surfaces dense, structured, and useful for repeated CRM operations. Do use semantic helpers for statuses, confidence, approvals, errors, and warning states.

Don't redesign screens while updating this contract. Don't introduce a new palette, type scale, radius scale, or decorative visual language without first changing the runtime theme and documenting the decision.

Don't use screenshots as the source of truth when a token or helper exists. Don't add nested cards, large marketing-style hero sections, decorative gradients, or one-off colors to operational app screens.
