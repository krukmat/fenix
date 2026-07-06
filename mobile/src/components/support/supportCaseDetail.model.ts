import type { SupportCaseDetailData } from './supportCaseDetail.types';

type RecordLike = Record<string, unknown>;

function readString(record: RecordLike | null | undefined, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'string' && value.length > 0) {
      return value;
    }
  }
  return undefined;
}

function parseCaseCore(source: RecordLike, handoff: RecordLike | undefined): Omit<SupportCaseDetailData, 'accountName' | 'contactName' | 'activeSignalCount'> {
  return {
    id: String(source.id ?? ''),
    subject: readString(source, 'subject'),
    status: readString(source, 'status') ?? 'open',
    priority: (readString(source, 'priority') as SupportCaseDetailData['priority'] | undefined) ?? 'medium',
    description: readString(source, 'description'),
    accountId: readString(source, 'accountId', 'account_id'),
    contactId: readString(source, 'contactId', 'contact_id'),
    slaDeadline: readString(source, 'slaDeadline', 'sla_deadline'),
    handoffStatus: readString(handoff, 'status') ?? readString(source, 'handoffStatus'),
    assignee: readString(source, 'assignee'),
  };
}

function parseContactName(contact: RecordLike | undefined): string | undefined {
  const composed = [readString(contact, 'firstName'), readString(contact, 'lastName')].filter(Boolean).join(' ');
  return composed || readString(contact, 'email');
}

function resolveCaseSource(payload: RecordLike | null): RecordLike | undefined {
  return (payload?.case as RecordLike | undefined) ?? payload ?? undefined;
}

function parseSignalCount(payload: RecordLike | null): number {
  return typeof payload?.active_signal_count === 'number' ? payload.active_signal_count : 0;
}

export function parseCasePayload(data: unknown): SupportCaseDetailData | undefined {
  const payload = (data ?? null) as RecordLike | null;
  const source = resolveCaseSource(payload);
  if (!source) {
    return undefined;
  }

  const account = payload?.account as RecordLike | undefined;
  const contact = payload?.contact as RecordLike | undefined;
  const handoff = payload?.handoff as RecordLike | undefined;

  return {
    ...parseCaseCore(source, handoff),
    accountName: readString(account, 'name'),
    contactName: parseContactName(contact),
    activeSignalCount: parseSignalCount(payload),
  };
}

export function formatSignalSummary(activeSignalCount?: number): string | null {
  if (!activeSignalCount || activeSignalCount <= 0) {
    return null;
  }

  return activeSignalCount === 1 ? '1 active signal' : `${activeSignalCount} active signals`;
}
