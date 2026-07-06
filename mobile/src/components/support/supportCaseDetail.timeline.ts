import type { TimelineEvent } from '../crm';
import { readString, type RecordLike } from './supportCaseDetail.shared';

function parseListRecords(data: unknown): RecordLike[] {
  if (Array.isArray(data)) return data as RecordLike[];
  if (!data || typeof data !== 'object') return [];
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

export function buildCaseTimeline(timelineData: unknown, activitiesData: unknown, notesData: unknown): TimelineEvent[] {
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
    timestamp: readString(activity, 'completedAt', 'completed_at', 'dueAt', 'due_at', 'updatedAt', 'updated_at', 'createdAt', 'created_at') ?? '',
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
