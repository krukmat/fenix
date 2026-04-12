import type { HandoffPackage, InboxHandoffItem, InboxResponse } from './api.types';

type UnknownRecord = Record<string, unknown>;
type HandoffEntityRef = { entity_type?: string; entity_id?: string };

function asRecord(value: unknown): UnknownRecord | null {
  return value !== null && typeof value === 'object' ? (value as UnknownRecord) : null;
}

function readString(record: UnknownRecord | null, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value;
    }
  }
  return undefined;
}

function readNumber(record: UnknownRecord | null, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

function normalizeEntityRef(record: UnknownRecord | null): { entity_type?: string; entity_id?: string } | undefined {
  if (!record) return undefined;
  const entityType = readString(record, 'entity_type', 'entityType');
  const entityId = readString(record, 'entity_id', 'entityId');
  if (!entityType && !entityId) return undefined;
  return { entity_type: entityType, entity_id: entityId };
}

function unwrapHandoffPayload(raw: unknown): UnknownRecord | null {
  const outer = asRecord(raw);
  return asRecord(outer?.data) ?? outer;
}

function pickFirstString(...values: (string | undefined)[]): string | undefined {
  return values.find((value) => typeof value === 'string' && value.trim() !== '');
}

function resolveNormalizedEntity(
  payload: UnknownRecord | null,
  triggerContext: HandoffEntityRef | undefined,
  finalOutput: HandoffEntityRef | undefined,
  caseId: string | undefined,
): HandoffEntityRef {
  return {
    entity_type: pickFirstString(
      readString(payload, 'entity_type', 'entityType'),
      triggerContext?.entity_type,
      finalOutput?.entity_type,
      caseId ? 'case' : undefined,
    ),
    entity_id: pickFirstString(
      readString(payload, 'entity_id', 'entityId'),
      triggerContext?.entity_id,
      finalOutput?.entity_id,
      caseId,
    ),
  };
}

function resolveEvidenceCount(payload: UnknownRecord | null, evidencePack: UnknownRecord | null): number {
  return (
    readNumber(payload, 'evidence_count', 'evidenceCount') ??
    readNumber(evidencePack, 'source_count') ??
    0
  );
}

function extractContext(payload: UnknownRecord | null, primaryKey: string, fallbackKey: string) {
  return normalizeEntityRef(asRecord(payload?.[primaryKey] ?? payload?.[fallbackKey]));
}

function buildNormalizedHandoff(
  payload: UnknownRecord | null,
  fallbackRunId: string | undefined,
  caseId: string | undefined,
  triggerContext: HandoffEntityRef | undefined,
  finalOutput: HandoffEntityRef | undefined,
  evidencePack: UnknownRecord | null,
): HandoffPackage {
  const entity = resolveNormalizedEntity(payload, triggerContext, finalOutput, caseId);

  return {
    run_id: readString(payload, 'run_id', 'runId') ?? fallbackRunId ?? '',
    reason:
      readString(payload, 'reason', 'abstentionReason', 'abstention_reason') ??
      'Human handoff available',
    conversation_context:
      readString(payload, 'conversation_context', 'conversationContext', 'caseSubject', 'case_subject') ?? '',
    evidence_count: resolveEvidenceCount(payload, evidencePack),
    entity_type: entity.entity_type,
    entity_id: entity.entity_id,
    created_at:
      readString(payload, 'created_at', 'createdAt', 'completedAt', 'startedAt') ?? '',
    caseId,
    triggerContext,
    finalOutput,
  };
}

export function normalizeHandoffPackage(raw: unknown, fallbackRunId?: string): HandoffPackage {
  const payload = unwrapHandoffPayload(raw);
  const triggerContext = extractContext(payload, 'triggerContext', 'trigger_context');
  const finalOutput = extractContext(payload, 'finalOutput', 'final_output');
  const evidencePack = asRecord(payload?.evidencePack ?? payload?.evidence_pack);
  const caseId = readString(payload, 'caseId', 'case_id');
  return buildNormalizedHandoff(payload, fallbackRunId, caseId, triggerContext, finalOutput, evidencePack);
}

export function normalizeInboxResponse(raw: unknown): InboxResponse {
  const payload = asRecord(raw);
  const approvals = Array.isArray(payload?.approvals) ? payload.approvals : [];
  const signals = Array.isArray(payload?.signals) ? payload.signals : [];
  const rejected = Array.isArray(payload?.rejected) ? payload.rejected : [];
  const rawHandoffs = Array.isArray(payload?.handoffs) ? payload.handoffs : [];

  const handoffs = rawHandoffs.map((item): InboxHandoffItem => {
    const handoffItem = asRecord(item);
    const runId = readString(handoffItem, 'run_id', 'runId') ?? '';
    return {
      type: 'handoff',
      run_id: runId,
      handoff: normalizeHandoffPackage(handoffItem?.handoff, runId),
    };
  });

  return {
    approvals: approvals as InboxResponse['approvals'],
    handoffs,
    signals: signals as InboxResponse['signals'],
    rejected: rejected as InboxResponse['rejected'],
  };
}
