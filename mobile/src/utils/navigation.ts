// W4-T1 (mobile_wedge_harmonization_plan): navigation helper for wedge dynamic routes
// Expo Router's typed-routes registry only covers statically known paths.
// Dynamic wedge paths (/sales/[id], /support/[id]/copilot, etc.) are not in the
// generated registry yet. This helper provides a type-safe escape hatch without
// scattering `as any` casts across screens.
import type { Href } from 'expo-router';

/** Cast a string path to Href for Expo Router push/replace calls on dynamic routes. */
export function wedgeHref(path: string): Href {
  return path as Href;
}

/** Cast a pathname object to an Href-compatible object for Expo Router push. */
export function wedgeHrefObject(pathname: string, params?: Record<string, string>): Href {
  return { pathname, params } as Href;
}

/** Resolve the wedge destination for a human handoff package. */
export function resolveWedgeHandoffDestination(
  entityType: string | undefined,
  entityId: string | undefined,
  runId: string,
): string {
  if (!entityType || !entityId) {
    return `/activity/${runId}`;
  }
  if (entityType === 'case') {
    return `/support/${entityId}`;
  }
  if (entityType === 'account') {
    return `/sales/${entityId}`;
  }
  if (entityType === 'deal') {
    return `/sales/deals/${entityId}`;
  }
  return `/activity/${runId}`;
}
