// W4-T1 (mobile_wedge_harmonization_plan): navigation helper for wedge dynamic routes
// Expo Router's typed-routes registry only covers statically known paths.
// Dynamic wedge paths (/sales/[id], /support/[id]/copilot, etc.) are not in the
// generated registry yet. This helper provides a type-safe escape hatch without
// scattering `as any` casts across screens.
import type { Href } from 'expo-router';

type HandoffEntitySource = {
  caseId?: string;
  entity_type?: string;
  entity_id?: string;
  triggerContext?: {
    entity_type?: string;
    entity_id?: string;
  };
  finalOutput?: {
    entity_type?: string;
    entity_id?: string;
  };
};

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

export function resolveHandoffEntityContext(handoff: HandoffEntitySource): {
  entityType?: string;
  entityId?: string;
} {
  const directEntity = { entityType: handoff.entity_type, entityId: handoff.entity_id };
  const triggerEntity = toHandoffEntityContext(handoff.triggerContext);
  const finalEntity = toHandoffEntityContext(handoff.finalOutput);
  const fallbackEntity = handoff.caseId ? { entityType: 'case', entityId: handoff.caseId } : {};
  const entityType = directEntity.entityType ?? triggerEntity.entityType ?? finalEntity.entityType ?? fallbackEntity.entityType;
  const entityId = directEntity.entityId ?? triggerEntity.entityId ?? finalEntity.entityId ?? fallbackEntity.entityId;

  return { entityType, entityId };
}

function toHandoffEntityContext(source?: { entity_type?: string; entity_id?: string }) {
  return {
    entityType: source?.entity_type,
    entityId: source?.entity_id,
  };
}

export function resolveWedgeHandoffPackageDestination(
  handoff: HandoffEntitySource,
  runId: string,
): string {
  const { entityType, entityId } = resolveHandoffEntityContext(handoff);
  return resolveWedgeHandoffDestination(entityType, entityId, runId);
}
