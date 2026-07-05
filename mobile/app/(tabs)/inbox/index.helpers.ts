import type { ApprovalDecisionFeedback } from '../../../src/components/approvals/ApprovalCard';
import type { InboxFilter, InboxRenderableItem } from '../../../src/components/inbox/InboxFeed';
import type { AgentRun, ApprovalRequest, HandoffPackage, Signal, InboxResponse } from '../../../src/services/api';

export type ApprovalFeedbackById = Record<string, ApprovalDecisionFeedback & { approval: ApprovalRequest }>;
export type ScreenState = 'loading' | 'error' | 'empty' | 'ready';

function toTimestamp(value: string | undefined): number {
  if (!value) return 0;
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

function sortApprovals(approvals: ApprovalRequest[]): ApprovalRequest[] {
  return [...approvals].sort((left, right) => {
    const expiresDiff = toTimestamp(left.expiresAt) - toTimestamp(right.expiresAt);
    if (expiresDiff !== 0) return expiresDiff;
    return toTimestamp(left.createdAt) - toTimestamp(right.createdAt);
  });
}

function sortHandoffs(handoffs: { run_id: string; handoff: HandoffPackage }[]) {
  return [...handoffs].sort(
    (left, right) => toTimestamp(right.handoff.created_at) - toTimestamp(left.handoff.created_at),
  );
}

function sortSignals(signals: Signal[]): Signal[] {
  return [...signals].sort((left, right) => {
    const confidenceDiff = right.confidence - left.confidence;
    if (confidenceDiff !== 0) return confidenceDiff;
    return toTimestamp(right.created_at) - toTimestamp(left.created_at);
  });
}

function sortRejected(runs: AgentRun[]): AgentRun[] {
  return [...runs].sort((left, right) => {
    const completedDiff = toTimestamp(right.completedAt) - toTimestamp(left.completedAt);
    if (completedDiff !== 0) return completedDiff;
    return toTimestamp(right.createdAt) - toTimestamp(left.createdAt);
  });
}

function interleaveItems(groups: InboxRenderableItem[][]): InboxRenderableItem[] {
  const ordered: InboxRenderableItem[] = [];
  const maxLength = Math.max(0, ...groups.map((group) => group.length));

  for (let index = 0; index < maxLength; index += 1) {
    for (const group of groups) {
      const item = group[index];
      if (item) {
        ordered.push(item);
      }
    }
  }

  return ordered;
}

function normalizeItems(
  approvals: ApprovalRequest[],
  handoffs: { run_id: string; handoff: HandoffPackage }[],
  signals: Signal[],
  rejected: AgentRun[],
): InboxRenderableItem[] {
  return interleaveItems([
    sortApprovals(approvals).map((approval) => ({ type: 'approval' as const, id: approval.id, approval })),
    sortHandoffs(handoffs).map(({ run_id: runId, handoff }) => ({
      type: 'handoff' as const,
      id: runId,
      runId,
      handoff,
    })),
    sortSignals(signals).map((signal) => ({ type: 'signal' as const, id: signal.id, signal })),
    sortRejected(rejected).map((run) => ({ type: 'rejected' as const, id: run.id, run })),
  ]);
}

function filterItems(items: InboxRenderableItem[], filter: InboxFilter): InboxRenderableItem[] {
  if (filter === 'all') return items;
  return items.filter((item) => item.type === filter);
}

function readString(record: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string' && value.trim().length > 0) {
      return value.trim();
    }
  }
  return undefined;
}

function getResponseErrorMessage(data: unknown): string | undefined {
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return undefined;
  }

  return readString(data as Record<string, unknown>, 'error', 'message');
}

function getApprovalTraceId(approval: ApprovalRequest): string | undefined {
  if (!approval.payload || typeof approval.payload !== 'object' || Array.isArray(approval.payload)) {
    return undefined;
  }

  return readString(
    approval.payload as Record<string, unknown>,
    'trace_id',
    'traceId',
    'source_trace_id',
    'sourceTraceId',
  );
}

export function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }

  if (typeof error === 'object' && error !== null) {
    const objectRecord = error as Record<string, unknown>;
    const message = readString(objectRecord, 'message');
    if (message) {
      return message;
    }

    const responseData = (error as { response?: { data?: unknown } }).response?.data;
    const responseMessage = getResponseErrorMessage(responseData);
    if (responseMessage) {
      return responseMessage;
    }
  }

  return 'Approval decision failed';
}

export function buildApprovalFeedback(
  approval: ApprovalRequest,
  outcome: 'approved' | 'rejected',
  comment?: string,
): ApprovalDecisionFeedback & { approval: ApprovalRequest } {
  const updatedAt = new Date().toISOString();

  return {
    approval: {
      ...approval,
      status: outcome,
      decidedAt: updatedAt,
      updatedAt,
    },
    kind: 'success',
    title: outcome === 'approved' ? 'Approval recorded' : 'Rejection recorded',
    body: outcome === 'approved'
      ? 'The server accepted this approval decision and the flow is now complete.'
      : 'The server accepted this rejection and the flow is now complete.',
    visibleStatus: outcome,
    comment,
    traceId: getApprovalTraceId(approval),
  };
}

export function buildApprovalConflictFeedback(
  approval: ApprovalRequest,
  error: unknown,
): (ApprovalDecisionFeedback & { approval: ApprovalRequest }) | null {
  const message = getErrorMessage(error).toLowerCase();

  if (message.includes('expired')) {
    return {
      approval: { ...approval, status: 'expired', updatedAt: new Date().toISOString() },
      kind: 'conflict',
      title: 'Approval expired',
      body: 'This approval expired before your decision could be recorded. Review the audit trail for the final governed state.',
      visibleStatus: 'expired',
      traceId: getApprovalTraceId(approval),
    };
  }

  if (message.includes('already decided') || message.includes('already closed')) {
    return {
      approval,
      kind: 'conflict',
      title: 'Approval already decided',
      body: 'Another operator or process already recorded a final decision for this request. Refresh context from the audit trail before acting again.',
      traceId: getApprovalTraceId(approval),
    };
  }

  return null;
}

function mergeApprovalFeedback(
  approvals: ApprovalRequest[],
  feedbackById: ApprovalFeedbackById,
): ApprovalRequest[] {
  const itemsById = new Map<string, ApprovalRequest>();

  for (const approval of approvals) {
    itemsById.set(approval.id, approval);
  }

  for (const [approvalId, feedback] of Object.entries(feedbackById)) {
    itemsById.set(approvalId, feedback.approval);
  }

  return Array.from(itemsById.values());
}

function extractInboxItems(data: InboxResponse | undefined) {
  return {
    approvals: data?.approvals ?? [],
    handoffs: data?.handoffs ?? [],
    signals: data?.signals ?? [],
    rejected: data?.rejected ?? [],
  };
}

function resolveScreenState(isLoading: boolean, hasError: boolean, totalItems: number): ScreenState {
  if (isLoading) return 'loading';
  if (hasError) return 'error';
  if (totalItems === 0) return 'empty';
  return 'ready';
}

export function buildInboxModel(
  data: InboxResponse | undefined,
  isLoading: boolean,
  error: Error | null,
  filter: InboxFilter,
  approvalFeedbackById: ApprovalFeedbackById,
) {
  const items = extractInboxItems(data);
  const allItems = normalizeItems(
    mergeApprovalFeedback(items.approvals, approvalFeedbackById),
    items.handoffs,
    items.signals,
    items.rejected,
  );
  const visibleItems = filterItems(allItems, filter);
  const totalItems = allItems.length;

  return {
    screenState: resolveScreenState(isLoading, Boolean(error), totalItems),
    errorMessage: error?.message ?? 'Inbox unavailable',
    visibleItems,
    totalItems,
  };
}
