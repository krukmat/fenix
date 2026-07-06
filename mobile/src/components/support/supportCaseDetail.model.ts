import type { TimelineEvent } from '../crm';

export interface SupportCaseDetailData {
  id: string;
  subject?: string;
  status: string;
  priority: 'low' | 'medium' | 'high';
  description?: string;
  accountId?: string;
  accountName?: string;
  contactId?: string;
  contactName?: string;
  slaDeadline?: string;
  handoffStatus?: string;
  assignee?: string;
  activeSignalCount?: number;
}

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

function parseCaseCore(
  source: RecordLike,
  handoff: RecordLike | undefined,
): Omit<SupportCaseDetailData, 'accountName' | 'contactName' | 'activeSignalCount'> {
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

function parseListRecords(data: unknown): RecordLike[] {
  if (Array.isArray(data)) {
    return data as RecordLike[];
  }
  if (!data || typeof data !== 'object') {
    return [];
  }
  const payload = data as { data?: unknown };
  return Array.isArray(payload.data) ? (payload.data as RecordLike[]) : [];
}

function toTimelineType(value?: string): TimelineEvent['type'] {
  if (value === 'note') return 'note';
  if (value === 'activity') return 'activity';
  if (value === 'created') return 'created';
  if (value === 'updated') return 'updated';
  if (value === 'status_change') return 'status_change';
  return 'updated';
}

function sortByNewestFirst(a: TimelineEvent, b: TimelineEvent): number {
  const aTime = Date.parse(a.timestamp);
  const bTime = Date.parse(b.timestamp);
  if (Number.isNaN(aTime) && Number.isNaN(bTime)) return 0;
  if (Number.isNaN(aTime)) return 1;
  if (Number.isNaN(bTime)) return -1;
  return bTime - aTime;
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

export function buildCaseTimeline(
  timelineData: unknown,
  activitiesData: unknown,
  notesData: unknown,
): TimelineEvent[] {
  const timelineEvents = parseListRecords(timelineData).map((event, index) => ({
    id: `timeline-${String(event.id ?? index)}`,
    type: toTimelineType(readString(event, 'eventType', 'event_type', 'type', 'action')),
    title: readString(event, 'title', 'eventType', 'event_type', 'type', 'action') ?? 'Timeline event',
    description: readString(event, 'description'),
    timestamp: readString(event, 'timestamp', 'createdAt', 'created_at') ?? '',
    userName: readString(event, 'actorId', 'actor_id'),
  }));

  const activities = parseListRecords(activitiesData).map((activity, index) => ({
    id: `activity-${String(activity.id ?? index)}`,
    type: 'activity' as const,
    title: readString(activity, 'subject', 'activityType', 'activity_type', 'type') ?? 'Activity',
    description: readString(activity, 'description', 'body', 'status'),
    timestamp:
      readString(activity, 'completedAt', 'completed_at', 'dueAt', 'due_at', 'updatedAt', 'updated_at', 'createdAt', 'created_at') ?? '',
    userName: readString(activity, 'assignedTo', 'assigned_to', 'ownerId', 'owner_id'),
  }));

  const notes = parseListRecords(notesData).map((note, index) => ({
    id: `note-${String(note.id ?? index)}`,
    type: 'note' as const,
    title: note.is_internal === true || note.isInternal === true ? 'Internal note' : 'Note',
    description: readString(note, 'content'),
    timestamp: readString(note, 'createdAt', 'created_at', 'updatedAt', 'updated_at') ?? '',
    userName: readString(note, 'authorId', 'author_id'),
  }));

  return [...timelineEvents, ...activities, ...notes].sort(sortByNewestFirst);
}
