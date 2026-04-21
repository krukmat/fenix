---
doc_type: task
id: ui-redesign-command-center
title: "Professional UI Redesign: Command Center Dark Theme"
status: completed
phase: mobile-polish
week: ""
tags: [mobile, ui, redesign, theme, react-native-paper]
fr_refs: [FR-300]
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - mobile/src/theme/colors.ts
  - mobile/src/theme/index.ts
  - mobile/src/theme/types.ts
  - mobile/src/theme/typography.ts
  - mobile/src/theme/spacing.ts
  - mobile/src/theme/semantic.ts
  - mobile/app/_layout.tsx
  - mobile/app/(tabs)/_layout.tsx
  - mobile/src/components/ui/AuthFormLayout.tsx
  - mobile/src/components/inbox/InboxFeed.tsx
  - mobile/src/components/approvals/ApprovalCard.tsx
  - mobile/src/components/signals/SignalCard.tsx
  - mobile/src/components/governance/AuditEventCard.tsx
  - mobile/src/components/governance/UsageDetailCard.tsx
  - mobile/src/components/crm/CoreCRMReadOnly.tsx
  - mobile/app/(tabs)/sales/index.tsx
  - mobile/app/(tabs)/activity/index.tsx
  - mobile/app/(tabs)/governance/index.tsx
  - mobile/app/(tabs)/crm/index.tsx
created: 2026-04-21
completed: 2026-04-21
---

# FenixCRM — Professional UI Redesign: "Command Center" Dark Theme

## Context

The current app uses MD3LightTheme with `#FEFBFF` white backgrounds, generic blue `#1565C0`, default system fonts, and basic `elevation: 2` shadow cards. The aesthetic feels like a scaffold — not a tool trusted by sales ops and governance officers. The redesign switches to a dark "Command Center" theme: deep navy/charcoal surfaces, precision color semantics, monospace data fields, and left-accent stripe patterns that telegraph urgency and navigability. No structural changes to components — this is a palette + style token upgrade.

**Language throughout: agnostic / English (already in place).**

---

## Gap Review — 2026-04-21

Status: **requires plan hardening before implementation**.

Detected gaps and fixes applied in this revision:

| Gap | Risk | Fix in this plan |
|-----|------|------------------|
| Missing FR traceability | The task affects the FR-300 mobile surface but had empty `fr_refs`. | Set `fr_refs: [FR-300]`. |
| No per-task reasoning complexity | Implementing agents could treat broad visual sweeps as high-reasoning work. | Added a task reasoning matrix with every task capped at `low` or `medium`. |
| Two tasks could drift into high reasoning | T13 and T16 touched local mappings, imports, and scattered style targets. | Constrained them to read-first, replace-one-concern-at-a-time implementation steps. |
| Import path ambiguity in tab screens | `mobile/app/(tabs)/sales/index.tsx` and `activity/index.tsx` import app-level dependencies from `../../../src/...`, not `../../src/...`. | Corrected task instructions for T16 and T17. |
| Verification command mismatch | The plan mentioned a generic Detox command while current visual audit plans use the mobile screenshot script / Maestro path. | Replaced verification with the repo's mobile QA gate plus `cd mobile && npm run screenshots` when visual infrastructure is available. |
| Mobile push policy not explicit | This plan touches `mobile/`, so local gates are required before any push. | Added `bash scripts/qa-mobile-prepush.sh` as the required pre-push gate. |

No functional-scope gap was found that requires adding new components, routes, hooks, services, or API behavior. This remains a style-token and visual-system task.

---

## Design System

### Color Palette (MD3DarkTheme base)

```
background:            #0A0D12   ← near-black canvas
surface:               #111620   ← card layer
surfaceVariant:        #1A2030   ← chip backgrounds, secondary surfaces
surfaceContainerHigh:  #1F2840   ← modals / popovers
outline:               #2E3A50   ← borders
outlineVariant:        #1E2B3E   ← hairline separators

primary:               #3B82F6   ← operator blue
onPrimary:             #FFFFFF
primaryContainer:      #1E3A5F   ← badge / highlight bg
onPrimaryContainer:    #93C5FD

secondary:             #F59E0B   ← amber for AI signals
onSecondary:           #0A0D12
secondaryContainer:    #3D2C00
onSecondaryContainer:  #FDE68A

error:                 #EF4444
onError:               #FFFFFF
errorContainer:        #3B0F0F
onErrorContainer:      #FCA5A5

onBackground:          #F0F4FF   ← primary text
onSurface:             #E2E8F0   ← card text
onSurfaceVariant:      #8899AA   ← metadata / tertiary text

inverseSurface:        #E2E8F0
inverseOnSurface:      #0A0D12
inversePrimary:        #3B82F6
```

**Semantic (exported separately as `semanticColors`):**
```
success / successContainer / onSuccessContainer: #10B981 / #052E1C / #6EE7B7
warning / warningContainer / onWarningContainer: #F59E0B / #3D2C00 / #FDE68A
info:              #60A5FA
confidenceHigh:    #10B981
confidenceMed:     #F59E0B
confidenceLow:     #6B7280
```

**Agent run status → color map:**
```
completed              → #10B981
completed_with_warnings→ #F59E0B
awaiting_approval      → #3B82F6
handed_off             → #A78BFA
denied_by_policy       → #EF4444
abstained              → #6B7280
failed                 → #DC2626
```

### Typography (no new deps — Roboto ships on Android)

```typescript
export const typography = {
  headingLG:  { fontFamily: 'Roboto', fontSize: 22, fontWeight: '700', letterSpacing: -0.3 },
  headingMD:  { fontFamily: 'Roboto', fontSize: 18, fontWeight: '600' },
  eyebrow:    { fontFamily: 'Roboto', fontSize: 11, fontWeight: '700', letterSpacing: 1.2, textTransform: 'uppercase' },
  labelMD:    { fontFamily: 'Roboto', fontSize: 11, fontWeight: '600', letterSpacing: 0.3 },
  mono:       { fontFamily: Platform.OS === 'android' ? 'monospace' : 'Courier New', fontSize: 12 },
  monoLG:     { fontFamily: Platform.OS === 'android' ? 'monospace' : 'Courier New', fontSize: 14, fontWeight: '700' },
  monoSM:     { fontFamily: Platform.OS === 'android' ? 'monospace' : 'Courier New', fontSize: 11 },
};
```

### Border Radius
```
xs: 4  sm: 6  md: 10  lg: 14  full: 999
```

### Card Shadow (dark theme — tinted, not black)
```typescript
// Replace elevation: 2 with:
{ borderWidth: 1, borderColor: '#1E2B3E', elevation: 0 }
// For elevated cards (e.g. ApprovalCard):
{ shadowColor: '#3B82F6', shadowOpacity: 0.08, shadowOffset: { width:0, height:2 }, shadowRadius: 8, elevation: 3 }
```

### Confidence Glow (SignalCard)
```typescript
// In src/theme/semantic.ts
export function confidenceGlowStyle(confidence: number) {
  if (confidence >= 0.8) return {
    borderWidth: 1, borderColor: 'rgba(16,185,129,0.6)',
    shadowColor: '#10B981', shadowOpacity: 0.3, shadowRadius: 6, elevation: 4,
  };
  if (confidence >= 0.5) return {
    borderWidth: 1, borderColor: 'rgba(245,158,11,0.5)',
    shadowColor: '#F59E0B', shadowOpacity: 0.2, shadowRadius: 4, elevation: 3,
  };
  return { borderWidth: 1, borderColor: 'rgba(107,114,128,0.3)' };
}
```

---

## New Files to Create

| File | Purpose |
|------|---------|
| `mobile/src/theme/typography.ts` | All type scale constants |
| `mobile/src/theme/spacing.ts` | `spacing`, `radius`, `elevation` tokens |
| `mobile/src/theme/semantic.ts` | `getAgentStatusColor`, `getConfidenceColor`, `confidenceGlowStyle` |

---

## Implementation Order & Tasks

> **Instructions for the implementing agent**
> - Read each task completely before starting. Every task includes the exact file path, what to read first, and the exact changes to make.
> - Execute phases in order — each phase depends on the previous.
> - Do NOT restructure JSX, rename components, or move logic. Style changes only (unless explicitly stated).
> - Do NOT remove `testID` props from any element.
> - After completing each task, mark it `[x]` in this document.
> - Run `cd mobile && npx tsc --noEmit` after Phase 1 to catch type errors early.
> - If a task starts requiring `high` reasoning, stop and split it into read-only discovery plus one or more small edit tasks before implementation.

### Reasoning Complexity Matrix

All implementation tasks are capped at **medium reasoning**. Tasks that could become high reasoning have been narrowed by explicit boundaries and dependency order.

| Task | Reasoning | Why it stays at this level |
|------|-----------|----------------------------|
| T1 | Low | Full-file token replacement with fixed palette. |
| T2 | Low | Switches MD3 base theme only. |
| T3 | Low | Adds optional interface fields without behavior changes. |
| T4 | Low | Creates isolated typography constants. |
| T5 | Low | Creates isolated spacing/radius/elevation constants. |
| T6 | Medium | Adds shared helper functions, but mapping and thresholds are fully specified. |
| T7 | Low | Two literal style value changes. |
| T8 | Medium | Replaces one navigation options constant; no component tree edits. |
| T9 | Medium | StyleSheet-only auth surface update. |
| T10 | Medium | Many literal style replacements in one file; no logic changes. |
| T11 | Medium | Adds expiry-driven border style using existing data. |
| T12 | Medium | Removes local helper and delegates to shared semantic helpers. |
| T13 | Medium | Split mentally into color delegation first, monospace text second, card border last. No data-shape changes. |
| T14 | Medium | Applies typography to known data fields only. |
| T15 | Low | Replaces elevation with border-based separation. |
| T16 | Medium | Limit to import correction, status-color delegation, amount typography, and active tab color. |
| T17 | Medium | Limit to chip colors, one Insights card style, and data typography. |
| T18 | Medium | Limit to section headers, quota fill logic, and audit row accent. |
| T19 | Medium | Swaps emoji render for already-installed icon component and updates card styles. |

### High-Reasoning Downgrade Rules

- **T13 downgrade:** Do not audit all governance rendering. Only touch outcome/status color calls, `trace_id` / detail JSON / timestamp text, and the `Card` border accent in `AuditEventCard.tsx`.
- **T16 downgrade:** Do not redesign the Sales screen. Only add imports from `../../../src/theme/...`, replace the local status color helper usage, apply `typography.monoLG` to deal amounts, and set the active segmented-tab border to `#3B82F6`.
- **T18 downgrade:** Do not refactor quota components. Only introduce a local `fillColor` constant in `QuotaItem` and use the specified tri-tier logic.

---

### Phase 1 — Design System Foundation
> **Why first:** Every other phase imports from these files. Complete all 6 tasks before moving on.

- [x] **T1** — `mobile/src/theme/colors.ts`
  - Read the current file first.
  - Replace the **entire file** with the following content:
  ```typescript
  // ui-redesign-command-center: dark Command Center palette
  export const brandColors = {
    primary:              '#3B82F6',
    onPrimary:            '#FFFFFF',
    primaryContainer:     '#1E3A5F',
    onPrimaryContainer:   '#93C5FD',
    secondary:            '#F59E0B',
    onSecondary:          '#0A0D12',
    secondaryContainer:   '#3D2C00',
    onSecondaryContainer: '#FDE68A',
    error:                '#EF4444',
    onError:              '#FFFFFF',
    errorContainer:       '#3B0F0F',
    onErrorContainer:     '#FCA5A5',
    background:           '#0A0D12',
    onBackground:         '#F0F4FF',
    surface:              '#111620',
    onSurface:            '#E2E8F0',
    surfaceVariant:       '#1A2030',
    onSurfaceVariant:     '#8899AA',
    outline:              '#2E3A50',
    outlineVariant:       '#1E2B3E',
    inverseSurface:       '#E2E8F0',
    inverseOnSurface:     '#0A0D12',
    inversePrimary:       '#3B82F6',
  } as const;

  export type BrandColors = typeof brandColors;

  export const semanticColors = {
    success:              '#10B981',
    successContainer:     '#052E1C',
    onSuccessContainer:   '#6EE7B7',
    warning:              '#F59E0B',
    warningContainer:     '#3D2C00',
    onWarningContainer:   '#FDE68A',
    info:                 '#60A5FA',
    confidenceHigh:       '#10B981',
    confidenceMed:        '#F59E0B',
    confidenceLow:        '#6B7280',
  } as const;
  ```

- [x] **T2** — `mobile/src/theme/index.ts`
  - Read the current file first.
  - Replace **one line only**: change `MD3LightTheme` to `MD3DarkTheme` in both the import and the spread.
  - Final file must look like:
  ```typescript
  // ui-redesign-command-center: switch to dark base theme
  import { MD3DarkTheme } from 'react-native-paper';
  import type { MD3Theme } from 'react-native-paper';
  import { brandColors } from './colors';

  export const fenixTheme: MD3Theme = {
    ...MD3DarkTheme,
    colors: {
      ...MD3DarkTheme.colors,
      ...brandColors,
    },
  };

  export { brandColors };
  ```

- [x] **T3** — `mobile/src/theme/types.ts`
  - Read the current file first.
  - Add these optional fields to the `ThemeColors` interface (do not remove existing fields):
  ```typescript
  success?: string;
  warning?: string;
  info?: string;
  surfaceContainerHigh?: string;
  ```

- [x] **T4** — `mobile/src/theme/typography.ts` (**NEW FILE — create it**)
  ```typescript
  // ui-redesign-command-center: type scale tokens
  import { Platform } from 'react-native';

  const monoFont = Platform.OS === 'android' ? 'monospace' : 'Courier New';

  export const typography = {
    headingLG: { fontFamily: 'Roboto', fontSize: 22, fontWeight: '700' as const, letterSpacing: -0.3 },
    headingMD: { fontFamily: 'Roboto', fontSize: 18, fontWeight: '600' as const },
    eyebrow:   { fontFamily: 'Roboto', fontSize: 11, fontWeight: '700' as const, letterSpacing: 1.2, textTransform: 'uppercase' as const },
    labelMD:   { fontFamily: 'Roboto', fontSize: 11, fontWeight: '600' as const, letterSpacing: 0.3 },
    mono:      { fontFamily: monoFont, fontSize: 12, fontWeight: '400' as const },
    monoLG:    { fontFamily: monoFont, fontSize: 14, fontWeight: '700' as const },
    monoSM:    { fontFamily: monoFont, fontSize: 11, fontWeight: '400' as const },
  } as const;
  ```

- [x] **T5** — `mobile/src/theme/spacing.ts` (**NEW FILE — create it**)
  ```typescript
  // ui-redesign-command-center: spacing, radius, elevation tokens
  export const spacing = { xs: 4, sm: 8, md: 12, base: 16, lg: 20, xl: 24, xxl: 32 } as const;

  export const radius = { xs: 4, sm: 6, md: 10, lg: 14, full: 999 } as const;

  export const elevation = {
    card:   { borderWidth: 1, borderColor: '#1E2B3E', elevation: 0 },
    raised: { shadowColor: '#3B82F6', shadowOpacity: 0.08, shadowOffset: { width: 0, height: 2 }, shadowRadius: 8, elevation: 3 },
    tabBar: { shadowColor: '#000000', shadowOpacity: 0.4, shadowOffset: { width: 0, height: -2 }, shadowRadius: 12, elevation: 12 },
  } as const;
  ```

- [x] **T6** — `mobile/src/theme/semantic.ts` (**NEW FILE — create it**)
  ```typescript
  // ui-redesign-command-center: shared color helpers
  import { semanticColors } from './colors';

  export function getAgentStatusColor(status: string): string {
    const map: Record<string, string> = {
      completed:               semanticColors.success,
      completed_with_warnings: semanticColors.warning,
      awaiting_approval:       '#3B82F6',
      handed_off:              '#A78BFA',
      denied_by_policy:        '#EF4444',
      abstained:               semanticColors.confidenceLow,
      failed:                  '#DC2626',
      won:                     semanticColors.success,
      lost:                    '#EF4444',
      open:                    '#3B82F6',
      high:                    '#EF4444',
      medium:                  semanticColors.warning,
      low:                     semanticColors.success,
      success:                 semanticColors.success,
      denied:                  '#EF4444',
      error:                   '#DC2626',
    };
    return map[status] ?? semanticColors.confidenceLow;
  }

  export function getConfidenceColor(confidence: number): string {
    if (confidence >= 0.8) return semanticColors.confidenceHigh;
    if (confidence >= 0.5) return semanticColors.confidenceMed;
    return semanticColors.confidenceLow;
  }

  export function confidenceGlowStyle(confidence: number): object {
    if (confidence >= 0.8) return {
      borderWidth: 1, borderColor: 'rgba(16,185,129,0.6)',
      shadowColor: '#10B981', shadowOpacity: 0.3, shadowRadius: 6, elevation: 4,
    };
    if (confidence >= 0.5) return {
      borderWidth: 1, borderColor: 'rgba(245,158,11,0.5)',
      shadowColor: '#F59E0B', shadowOpacity: 0.2, shadowRadius: 4, elevation: 3,
    };
    return { borderWidth: 1, borderColor: 'rgba(107,114,128,0.3)' };
  }
  ```

---

### Phase 2 — Navigation Shell
> **Why second:** Tab bar and header colors frame every screen. Fix these before touching individual screens.

- [x] **T7** — `mobile/app/_layout.tsx`
  - Read the file. Find `<StatusBar` and change `style="auto"` to `style="light"`.
  - Find any view with `backgroundColor: '#1565C0'` (splash loader) and change it to `backgroundColor: '#0A0D12'`.
  - No other changes.

- [x] **T8** — `mobile/app/(tabs)/_layout.tsx`
  - Read the file. Find the `TAB_SCREEN_OPTIONS` constant (around line 32).
  - Replace only the values inside it — do not touch the component tree below. New values:
  ```typescript
  const TAB_SCREEN_OPTIONS = {
    headerShown: true,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: '#111620' },
    headerTintColor: '#F0F4FF',
    headerTitleAlign: 'left' as const,
    headerTitleStyle: {
      color: '#F0F4FF',
      fontSize: 18,
      fontWeight: '700' as const,
      letterSpacing: -0.3,
    },
    tabBarActiveTintColor: '#3B82F6',
    tabBarInactiveTintColor: '#8899AA',
    tabBarShowLabel: true,
    tabBarHideOnKeyboard: true,
    tabBarStyle: {
      backgroundColor: '#111620',
      borderTopColor: '#1E2B3E',
      borderTopWidth: 1,
      height: 68,
      paddingTop: 6,
      paddingBottom: 8,
      shadowColor: '#000000',
      shadowOpacity: 0.4,
      shadowOffset: { width: 0, height: -2 },
      shadowRadius: 12,
      elevation: 12,
    },
    tabBarLabelStyle: {
      fontSize: 11,
      fontWeight: '600' as const,
      marginTop: 2,
      letterSpacing: 0.2,
    },
    tabBarItemStyle: { paddingVertical: 2 },
    tabBarBadgeStyle: {
      backgroundColor: '#EF4444',
      color: '#FFFFFF',
      fontSize: 10,
      fontWeight: '700' as const,
      minWidth: 18,
      height: 18,
      borderRadius: 9,
    },
  };
  ```

---

### Phase 3 — Auth Screen

- [x] **T9** — `mobile/src/components/ui/AuthFormLayout.tsx`
  - Read the file. Find the `StyleSheet.create({...})` call at the bottom.
  - Replace only the style values listed below. Do not change JSX structure:
  ```
  container.backgroundColor: (whatever it is) → '#0A0D12'
  scrollContent: keep flex/alignment, no color changes needed
  form (or card wrapper):
    backgroundColor:  → '#111620'
    borderRadius:     → 14
    borderWidth:      → 1
    borderColor:      → '#1E2B3E'
    borderLeftWidth:  → 3
    borderLeftColor:  → '#3B82F6'
    shadowColor:      → '#3B82F6'
    shadowOpacity:    → 0.12
    shadowOffset:     → { width: 0, height: 4 }
    shadowRadius:     → 16
    elevation:        → 6
  title:
    textAlign:    → 'left'
    color:        → '#F0F4FF'
    fontSize:     → 26
    fontWeight:   → '700'
    letterSpacing:→ -0.5
  subtitle:
    textAlign:    → 'left'
    color:        → '#8899AA'
    fontSize:     → 13
  ```
  - `TextInput` and `Button` do NOT need changes — they inherit the dark theme from T2.

---

### Phase 4 — Inbox

- [x] **T10** — `mobile/src/components/inbox/InboxFeed.tsx`
  - Read the file. The `StyleSheet.create({...})` block starts around line 214. Make these exact replacements:
  ```
  styles.container.backgroundColor:      '#FFFFFF'   → '#0A0D12'
  styles.title.color:                    '#111827'   → '#F0F4FF'
  styles.subtitle.color:                 '#6B7280'   → '#8899AA'
  styles.count.color:                    '#1F2937'   → '#E2E8F0'
  styles.visibleCount.color:             '#6B7280'   → '#8899AA'
  styles.chipSelected.backgroundColor:   '#111827'   → '#3B82F6'
  styles.chipSelected.borderColor:       '#111827'   → '#3B82F6'
  styles.chipIdle.backgroundColor:       '#FFFFFF'   → '#1A2030'
  styles.chipIdle.borderColor:           '#D1D5DB'   → '#2E3A50'
  styles.chipTextIdle.color:             '#111827'   → '#E2E8F0'
  styles.stateTitle.color:               '#111827'   → '#F0F4FF'
  styles.stateBody.color:                '#6B7280'   → '#8899AA'
  styles.retryButton.backgroundColor:    '#111827'   → '#3B82F6'
  styles.inlineError.backgroundColor:    '#FEF2F2'   → '#3B0F0F'
  styles.inlineError.borderColor:        '#FCA5A5'   → '#EF4444'
  styles.inlineErrorText.color:          '#991B1B'   → '#FCA5A5'
  ```
  - Replace `styles.handoffCard` block entirely:
  ```typescript
  handoffCard: {
    marginHorizontal: 16, marginBottom: 8, padding: 16, borderRadius: 12,
    backgroundColor: '#3D2C00',
    borderWidth: 1, borderColor: '#F59E0B', borderLeftWidth: 3,
  },
  handoffEyebrow: { fontSize: 12, fontWeight: '700', color: '#FDE68A', textTransform: 'uppercase', marginBottom: 6 },
  handoffReason:  { fontSize: 16, fontWeight: '600', color: '#F0F4FF', marginBottom: 6 },
  handoffMeta:    { fontSize: 13, color: '#FDE68A' },
  ```
  - Replace `styles.rejectedCard` block entirely:
  ```typescript
  rejectedCard: {
    marginHorizontal: 16, marginBottom: 8, padding: 16, borderRadius: 12,
    backgroundColor: '#3B0F0F',
    borderWidth: 1, borderColor: '#EF4444', borderLeftWidth: 3,
  },
  rejectedEyebrow: { fontSize: 12, fontWeight: '700', color: '#FCA5A5', textTransform: 'uppercase', marginBottom: 6 },
  rejectedReason:  { fontSize: 16, fontWeight: '600', color: '#F0F4FF', marginBottom: 6 },
  rejectedMeta:    { fontSize: 13, color: '#FCA5A5' },
  ```

---

### Phase 5 — Shared Components

- [x] **T11** — `mobile/src/components/approvals/ApprovalCard.tsx`
  - Read the file. Find the `Card` component's `style` prop (or `styles.card` in StyleSheet).
  - Add a left border that changes color based on expiry. If the card style is in `StyleSheet.create`, move it to an inline style on `<Card>`:
  ```typescript
  // Find the Card render and add/modify its style prop:
  <Card
    style={[
      styles.card,
      { borderLeftWidth: 3, borderLeftColor: isExpired ? theme.colors.error : theme.colors.primary }
    ]}
    ...
  >
  ```
  - If `isExpired` is not already a variable in scope, derive it from the approval's `expires_at` field: `const isExpired = approval.expires_at ? new Date(approval.expires_at) < new Date() : false;`

- [x] **T12** — `mobile/src/components/signals/SignalCard.tsx`
  - Read the file. Lines 15–19 contain a local `confidenceColor()` function. Delete it.
  - Add this import at the top (after existing imports):
  ```typescript
  import { getConfidenceColor, confidenceGlowStyle } from '../../theme/semantic';
  ```
  - In the `SignalCard` component body, find `const color = confidenceColor(signal.confidence)` and change to:
  ```typescript
  const color = getConfidenceColor(signal.confidence);
  ```
  - Find `<Card ... style={styles.card}` and change to:
  ```typescript
  <Card ... style={[styles.card, confidenceGlowStyle(signal.confidence)]}
  ```
  - No other changes.

- [x] **T13** — `mobile/src/components/governance/AuditEventCard.tsx`
  - Read the file. Find any local function that maps outcome/status strings to hex colors (e.g. `getOutcomeColor()`). Add this import:
  ```typescript
  import { getAgentStatusColor } from '../../theme/semantic';
  import { typography } from '../../theme/typography';
  ```
  - Replace the body of that local color function to delegate to `getAgentStatusColor(outcome)`, or replace its call sites directly.
  - Find any `Text` displaying `trace_id`, detail JSON, or timestamp. Add `style={typography.monoSM}` to those `Text` elements.
  - On the `Card` component, add a left border in the outcome color:
  ```typescript
  style={[styles.card, { borderLeftWidth: 2, borderLeftColor: getAgentStatusColor(event.outcome) }]}
  ```

- [x] **T14** — `mobile/src/components/governance/UsageDetailCard.tsx`
  - Read the file. Add these imports:
  ```typescript
  import { typography } from '../../theme/typography';
  import { semanticColors } from '../../theme/colors';
  ```
  - Find the `Text` element(s) displaying cost values (e.g. `€0.05000`). Add:
  ```typescript
  style={[typography.monoLG, { color: semanticColors.success }]}
  ```
  - Find `Text` elements displaying latency (e.g. `1200 ms`). Add: `style={typography.monoSM}`
  - Find `Text` elements displaying timestamps. Add: `style={typography.monoSM}`
  - Badge colors auto-update via `useTheme()` — no changes needed there.

- [x] **T15** — `mobile/src/components/crm/CoreCRMReadOnly.tsx`
  - Read the file. Find any row/card style with `elevation: 1` or `elevation: 2`.
  - Replace those elevation values with border-based separation:
  ```typescript
  // Remove: elevation: 1 (or 2)
  // Add:
  borderWidth: 1,
  borderColor: '#1E2B3E',
  elevation: 0,
  ```

---

### Phase 6 — Screens

- [x] **T16** — `mobile/app/(tabs)/sales/index.tsx`
  - Read the file. Find the internal segmented tab bar (not the Expo Router tab bar — it's a `View` with tab buttons rendered inside the screen).
  - On the active tab indicator, change `borderBottomColor` to `'#3B82F6'`.
  - Find `Text` elements displaying deal amounts (values like `$50,000`). Add `style={typography.monoLG}` — import `typography` from `../../../src/theme/typography`.
  - Find any local `getDealStatusColor()` or `getPublicStatusColor()` function. Import `getAgentStatusColor` from `../../../src/theme/semantic` and replace calls to those local functions with `getAgentStatusColor(status)`.
  - Medium-reasoning boundary: do not alter tab membership, list data fetching, routing, or the Sales screen layout.

- [x] **T17** — `mobile/app/(tabs)/activity/index.tsx`
  - Read the file. Find filter chips (similar to Inbox chips — pill-shaped `Pressable` elements).
  - Apply the same chip pattern as T10:
    - Active: `backgroundColor: '#1E3A5F', borderColor: '#3B82F6'`
    - Idle: `backgroundColor: '#1A2030', borderColor: '#2E3A50'`
  - Find the "Insights" entry card (the card that opens the insights query form). Change its container style to:
  ```typescript
  { backgroundColor: '#1E3A5F', borderWidth: 1, borderColor: '#3B82F6', borderLeftWidth: 3, borderRadius: 12, padding: 18 }
  ```
  - Find `Text` elements in run rows displaying cost (e.g. `€0.05`) and latency (e.g. `1200ms`). Import `typography` from `../../../src/theme/typography` and add `style={typography.monoSM}`.
  - Medium-reasoning boundary: do not change filter semantics, status mapping, routing, or run sorting.

- [x] **T18** — `mobile/app/(tabs)/governance/index.tsx`
  - Read the file. Find section header `Text` elements (e.g. "RECENT USAGE", "QUOTA STATES").
  - Replace their style with:
  ```typescript
  { fontSize: 11, fontWeight: '700', letterSpacing: 1.2, textTransform: 'uppercase', color: '#8899AA', marginBottom: 12 }
  ```
  - Find the quota progress bar `View`. Change its height from current value to `4`. Update the fill color logic to tri-tier:
  ```typescript
  const fillColor = pct >= 90 ? '#EF4444' : pct >= 70 ? '#F59E0B' : '#3B82F6';
  ```
  - Find the Audit Trail row (a tappable row navigating to audit detail). Add:
  ```typescript
  { borderLeftWidth: 3, borderLeftColor: '#3B82F6' }
  ```
  - Medium-reasoning boundary: do not change governance summary fetching, quota percentage calculation, or navigation targets.

- [x] **T19** — `mobile/app/(tabs)/crm/index.tsx`
  - Read the file. Find the `ENTITIES` array (or equivalent) that defines each CRM entity tile.
  - It currently uses emoji strings (e.g. `'🏢'`). Replace each emoji with a `MaterialCommunityIcons` icon name string and update the render to use `<MaterialCommunityIcons name={entity.iconName} size={28} color="#3B82F6" />`:
  ```typescript
  // Before (emoji): icon: '🏢'
  // After (icon name):
  { label: 'Accounts', iconName: 'domain' }
  { label: 'Contacts', iconName: 'account-group' }
  { label: 'Leads',    iconName: 'target' }
  { label: 'Deals',    iconName: 'handshake' }
  { label: 'Cases',    iconName: 'ticket-confirmation' }
  ```
  - `MaterialCommunityIcons` is already imported in the codebase (`@expo/vector-icons`) — check and import if not already in this file.
  - Update each grid card's style:
  ```typescript
  { backgroundColor: '#111620', borderWidth: 1, borderColor: '#1E2B3E', borderRadius: 12 }
  ```
  - Find the "Select entity" heading `Text`. Replace its style with:
  ```typescript
  { fontSize: 11, fontWeight: '700', letterSpacing: 1.2, textTransform: 'uppercase', color: '#8899AA', marginBottom: 16 }
  ```

---

## Files Affected

| File | Change Type |
|------|-------------|
| `mobile/src/theme/colors.ts` | Rewrite |
| `mobile/src/theme/index.ts` | Rewrite |
| `mobile/src/theme/types.ts` | Extend |
| `mobile/src/theme/typography.ts` | **NEW** |
| `mobile/src/theme/spacing.ts` | **NEW** |
| `mobile/src/theme/semantic.ts` | **NEW** |
| `mobile/app/_layout.tsx` | Minor edit |
| `mobile/app/(tabs)/_layout.tsx` | Rewrite `TAB_SCREEN_OPTIONS` |
| `mobile/src/components/ui/AuthFormLayout.tsx` | Style rewrite |
| `mobile/src/components/inbox/InboxFeed.tsx` | Style rewrite (StyleSheet only) |
| `mobile/src/components/approvals/ApprovalCard.tsx` | Minor style addition |
| `mobile/src/components/signals/SignalCard.tsx` | Replace `confidenceColor()` + card glow |
| `mobile/src/components/governance/AuditEventCard.tsx` | Status colors + monospace |
| `mobile/src/components/governance/UsageDetailCard.tsx` | Monospace data fields |
| `mobile/src/components/crm/CoreCRMReadOnly.tsx` | Row border style |
| `mobile/app/(tabs)/sales/index.tsx` | Tab bar + chip colors + monospace |
| `mobile/app/(tabs)/activity/index.tsx` | Filter chips + InsightsCard + RunRow |
| `mobile/app/(tabs)/governance/index.tsx` | Section headers + quota bar + audit row |
| `mobile/app/(tabs)/crm/index.tsx` | Icon swap + card dark style |

---

## What NOT to Change

- Component tree structure — no JSX restructuring
- All `testID` props — E2E tests must continue passing
- Routing configuration in `_layout.tsx`
- All hooks, stores, services, API layer
- Any animation/gesture handler logic

---

## Verification

Required local gates before any push:

1. Run `bash scripts/qa-mobile-prepush.sh` from the repo root.
2. If the shortcut cannot run, run the required mobile gates individually:
   - `bash scripts/check-no-inline-eslint-disable.sh`
   - `cd mobile && npm run typecheck`
   - `cd mobile && npm run lint`
   - `cd mobile && npm run quality:arch`
   - `cd mobile && npm run test:coverage`

Visual verification:

1. Run `cd mobile && npx expo start --android` and visually verify login, all 5 tabs, Inbox, Sales, Activity, Governance, and CRM hub.
2. Run `cd mobile && npm run screenshots` when Android/Maestro screenshot infrastructure is available.
3. Inspect generated screenshots in `mobile/artifacts/screenshots/` for:
   - dark background on every touched screen;
   - visible tab/header contrast;
   - no white card islands left from the old light theme;
   - `SignalCard` confidence glow visible on green/amber borders;
   - CRM hub icons rendered as `MaterialCommunityIcons`, not emoji;
   - no text overlap or clipped tab labels.
4. Verify MD3DarkTheme propagates correctly — Paper `Card`, `TextInput`, and `Button` should use dark surfaces unless a task explicitly sets a style override.
5. Check Android `StatusBar` shows white icons on the dark header.

If a required local gate cannot execute because local infrastructure is missing, stop and report the exact command, failure reason, and remaining risk before pushing.

---

## Complejidad: Media
## Tokens: ~4300
