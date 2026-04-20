import type {
  CRMAccount,
  CRMActivity,
  CRMAttachment,
  CRMCase,
  CRMContact,
  CRMDeal,
  CRMLead,
  CRMListResponse,
  CRMNote,
  CRMPageMeta,
  CRMPipeline,
  CRMPipelineStage,
  CRMTimelineEvent,
} from './api.crm.types';

type UnknownRecord = Record<string, unknown>;

function asRecord(value: unknown): UnknownRecord | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as UnknownRecord)
    : null;
}

function readString(record: UnknownRecord | null, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'string' && value.trim() !== '') return value;
  }
  return undefined;
}

function readNumber(record: UnknownRecord | null, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'number' && Number.isFinite(value)) return value;
  }
  return undefined;
}

function readBoolean(record: UnknownRecord | null, ...keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'boolean') return value;
    if (typeof value === 'number' && (value === 0 || value === 1)) return value === 1;
  }
  return undefined;
}

function readMetadata(record: UnknownRecord | null): Record<string, unknown> {
  const raw = record?.metadata;
  const rawRecord = asRecord(raw);
  if (rawRecord) return rawRecord;
  if (typeof raw === 'string' && raw.trim() !== '') {
    try {
      const parsed = JSON.parse(raw) as unknown;
      return asRecord(parsed) ?? {};
    } catch {
      return {};
    }
  }
  return {};
}

function base(record: UnknownRecord | null) {
  return {
    id: readString(record, 'id') ?? '',
    workspaceId: readString(record, 'workspaceId', 'workspace_id'),
    createdAt: readString(record, 'createdAt', 'created_at'),
    updatedAt: readString(record, 'updatedAt', 'updated_at'),
  };
}

function owner(record: UnknownRecord | null) {
  return { ...base(record), ownerId: readString(record, 'ownerId', 'owner_id') };
}

function signalCount(record: UnknownRecord | null): number {
  return readNumber(record, 'activeSignalCount', 'active_signal_count') ?? 0;
}

function listItems(raw: unknown, payload: UnknownRecord | null): unknown[] {
  if (Array.isArray(raw)) return raw;
  return Array.isArray(payload?.data) ? payload.data : [];
}

function metaNumber(
  metaRecord: UnknownRecord | null,
  payload: UnknownRecord | null,
  key: keyof CRMPageMeta,
  fallback: number,
): number {
  return readNumber(metaRecord, key) ?? readNumber(payload, key) ?? fallback;
}

export function normalizeCRMList<T>(
  raw: unknown,
  normalizeItem: (item: unknown) => T,
): CRMListResponse<T> {
  const payload = asRecord(raw);
  const items = listItems(raw, payload);
  const metaRecord = asRecord(payload?.meta);
  const data = items.map(normalizeItem);
  const meta: CRMPageMeta = {
    total: metaNumber(metaRecord, payload, 'total', data.length),
    limit: metaNumber(metaRecord, payload, 'limit', data.length),
    offset: metaNumber(metaRecord, payload, 'offset', 0),
  };
  return { data, meta };
}

export function normalizeCRMAccount(raw: unknown): CRMAccount {
  const r = asRecord(raw);
  return {
    ...owner(r),
    name: readString(r, 'name') ?? 'Unnamed Account',
    industry: readString(r, 'industry'),
    website: readString(r, 'website'),
    phone: readString(r, 'phone'),
    email: readString(r, 'email'),
    description: readString(r, 'description'),
    activeSignalCount: signalCount(r),
  };
}

export function normalizeCRMContact(raw: unknown): CRMContact {
  const r = asRecord(raw);
  return {
    ...owner(r),
    accountId: readString(r, 'accountId', 'account_id'),
    firstName: readString(r, 'firstName', 'first_name'),
    lastName: readString(r, 'lastName', 'last_name'),
    email: readString(r, 'email'),
    phone: readString(r, 'phone'),
    title: readString(r, 'title'),
    activeSignalCount: signalCount(r),
  };
}

export function normalizeCRMLead(raw: unknown): CRMLead {
  const r = asRecord(raw);
  return {
    ...owner(r),
    accountId: readString(r, 'accountId', 'account_id'),
    contactId: readString(r, 'contactId', 'contact_id'),
    source: readString(r, 'source'),
    status: readString(r, 'status'),
    score: readNumber(r, 'score'),
    metadata: readMetadata(r),
  };
}

export function normalizeCRMPipeline(raw: unknown): CRMPipeline {
  const r = asRecord(raw);
  return {
    ...base(r),
    name: readString(r, 'name') ?? 'Unnamed Pipeline',
    entityType: readString(r, 'entityType', 'entity_type'),
    isDefault: readBoolean(r, 'isDefault', 'is_default') ?? false,
  };
}

export function normalizeCRMPipelineStage(raw: unknown): CRMPipelineStage {
  const r = asRecord(raw);
  return {
    ...base(r),
    pipelineId: readString(r, 'pipelineId', 'pipeline_id') ?? '',
    name: readString(r, 'name') ?? 'Unnamed Stage',
    position: readNumber(r, 'position'),
    probability: readNumber(r, 'probability'),
  };
}

export function normalizeCRMDeal(raw: unknown): CRMDeal {
  const r = asRecord(raw);
  return {
    ...owner(r),
    accountId: readString(r, 'accountId', 'account_id') ?? '',
    contactId: readString(r, 'contactId', 'contact_id'),
    pipelineId: readString(r, 'pipelineId', 'pipeline_id') ?? '',
    stageId: readString(r, 'stageId', 'stage_id') ?? '',
    title: readString(r, 'title', 'name') ?? 'Unnamed Deal',
    amount: readNumber(r, 'amount'),
    currency: readString(r, 'currency'),
    expectedClose: readString(r, 'expectedClose', 'expected_close'),
    status: readString(r, 'status'),
    metadata: readMetadata(r),
    accountName: readString(r, 'accountName', 'account_name'),
    activeSignalCount: signalCount(r),
  };
}

export function normalizeCRMCase(raw: unknown): CRMCase {
  const r = asRecord(raw);
  return {
    ...owner(r),
    accountId: readString(r, 'accountId', 'account_id'),
    contactId: readString(r, 'contactId', 'contact_id'),
    pipelineId: readString(r, 'pipelineId', 'pipeline_id'),
    stageId: readString(r, 'stageId', 'stage_id'),
    subject: readString(r, 'subject') ?? 'No Subject',
    description: readString(r, 'description'),
    priority: readString(r, 'priority'),
    status: readString(r, 'status'),
    channel: readString(r, 'channel'),
    slaConfig: readString(r, 'slaConfig', 'sla_config'),
    slaDeadline: readString(r, 'slaDeadline', 'sla_deadline'),
    metadata: readMetadata(r),
    accountName: readString(r, 'accountName', 'account_name'),
    activeSignalCount: signalCount(r),
  };
}

export function normalizeCRMActivity(raw: unknown): CRMActivity {
  const r = asRecord(raw);
  return {
    ...base(r),
    entityType: readString(r, 'entityType', 'entity_type') ?? '',
    entityId: readString(r, 'entityId', 'entity_id') ?? '',
    ownerId: readString(r, 'ownerId', 'owner_id'),
    assignedTo: readString(r, 'assignedTo', 'assigned_to'),
    type: readString(r, 'type', 'activity_type'),
    subject: readString(r, 'subject'),
    description: readString(r, 'description'),
    status: readString(r, 'status'),
    dueAt: readString(r, 'dueAt', 'due_at'),
    completedAt: readString(r, 'completedAt', 'completed_at'),
    metadata: readMetadata(r),
  };
}

export function normalizeCRMNote(raw: unknown): CRMNote {
  const r = asRecord(raw);
  return {
    ...base(r),
    entityType: readString(r, 'entityType', 'entity_type') ?? '',
    entityId: readString(r, 'entityId', 'entity_id') ?? '',
    authorId: readString(r, 'authorId', 'author_id'),
    content: readString(r, 'content') ?? '',
    isInternal: readBoolean(r, 'isInternal', 'is_internal') ?? false,
    metadata: readMetadata(r),
  };
}

export function normalizeCRMAttachment(raw: unknown): CRMAttachment {
  const r = asRecord(raw);
  return {
    ...base(r),
    entityType: readString(r, 'entityType', 'entity_type') ?? '',
    entityId: readString(r, 'entityId', 'entity_id') ?? '',
    uploaderId: readString(r, 'uploaderId', 'uploader_id'),
    fileName: readString(r, 'fileName', 'file_name'),
    storagePath: readString(r, 'storagePath', 'storage_path'),
    contentType: readString(r, 'contentType', 'content_type'),
    sizeBytes: readNumber(r, 'sizeBytes', 'size_bytes'),
    metadata: readMetadata(r),
  };
}

export function normalizeCRMTimelineEvent(raw: unknown): CRMTimelineEvent {
  const r = asRecord(raw);
  const eventType = readString(r, 'eventType', 'event_type', 'type', 'action');
  return {
    ...base(r),
    entityType: readString(r, 'entityType', 'entity_type') ?? '',
    entityId: readString(r, 'entityId', 'entity_id') ?? '',
    actorId: readString(r, 'actorId', 'actor_id'),
    eventType,
    action: readString(r, 'action'),
    title: readString(r, 'title') ?? eventType ?? 'Timeline event',
    description: readString(r, 'description'),
    timestamp: readString(r, 'timestamp', 'createdAt', 'created_at') ?? '',
    metadata: readMetadata(r),
  };
}
