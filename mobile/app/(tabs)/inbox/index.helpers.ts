import type { ApprovalDecisionFeedback } from '../../../src/components/approvals/ApprovalCard';
import type { InboxFilter, InboxGroup, InboxGroupKey } from '../../../src/components/inbox/InboxFeed';
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

const GROUP_LABELS: Record<InboxGroupKey, string> = {
  approval: 'Awaiting your approval',
  'approval-decided': 'Recently decided',
  handoff: 'Handoffs waiting',
  signal: 'Signals to review',
  rejected: 'Denied runs',
};

function isApprovalExpired(approval: ApprovalRequest, now: number): boolean {
  return toTimestamp(approval.expiresAt) <= now;
}

function isApprovalDecided(approval: ApprovalRequest, feedbackById: ApprovalFeedbackById, now: number): boolean {
  return approval.status !== 'pending' || Boolean(feedbackById[approval.id]) || isApprovalExpired(approval, now);
}

function decisionTimestamp(approval: ApprovalRequest, now: number): number {
  if (approval.decidedAt) return toTimestamp(approval.decidedAt);
  if (approval.status === 'pending' && isApprovalExpired(approval, now)) return toTimestamp(approval.expiresAt);
  return toTimestamp(approval.updatedAt);
}

function sortByDecisionRecency(approvals: ApprovalRequest[], now: number): ApprovalRequest[] {
  return [...approvals].sort((left, right) => decisionTimestamp(right, now) - decisionTimestamp(left, now));
}

function partitionByDecided(
  approvals: ApprovalRequest[],
  feedbackById: ApprovalFeedbackById,
  now: number,
): { pending: ApprovalRequest[]; decided: ApprovalRequest[] } {
  const pending: ApprovalRequest[] = [];
  const decided: ApprovalRequest[] = [];

  for (const approval of approvals) {
    if (isApprovalDecided(approval, feedbackById, now)) {
      decided.push(approval);
    } else {
      pending.push(approval);
    }
  }

  return { pending, decided };
}

function groupInboxItems(
  approvals: ApprovalRequest[],
  feedbackById: ApprovalFeedbackById,
  handoffs: { run_id: string; handoff: HandoffPackage }[],
  signals: Signal[],
  rejected: AgentRun[],
): InboxGroup[] {
  const now = Date.now();
  const { pending: pendingApprovals, decided: decidedApprovals } = partitionByDecided(
    approvals,
    feedbackById,
    now,
  );
  const pendingApprovalItems = sortApprovals(pendingApprovals).map((approval) => ({
    type: 'approval' as const,
    id: approval.id,
    approval,
  }));
  const decidedApprovalItems = sortByDecisionRecency(decidedApprovals, now).map((approval) => ({
    type: 'approval' as const,
    id: approval.id,
    approval,
  }));
  const handoffItems = sortHandoffs(handoffs).map(({ run_id: runId, handoff }) => ({
    type: 'handoff' as const,
    id: runId,
    runId,
    handoff,
  }));
  const signalItems = sortSignals(signals).map((signal) => ({
    type: 'signal' as const,
    id: signal.id,
    signal,
  }));
  const rejectedItems = sortRejected(rejected).map((run) => ({
    type: 'rejected' as const,
    id: run.id,
    run,
  }));

  return [
    { key: 'approval', filterKey: 'approval', label: GROUP_LABELS.approval, items: pendingApprovalItems },
    {
      key: 'approval-decided',
      filterKey: 'approval',
      label: GROUP_LABELS['approval-decided'],
      items: decidedApprovalItems,
    },
    { key: 'handoff', filterKey: 'handoff', label: GROUP_LABELS.handoff, items: handoffItems },
    { key: 'signal', filterKey: 'signal', label: GROUP_LABELS.signal, items: signalItems },
    { key: 'rejected', filterKey: 'rejected', label: GROUP_LABELS.rejected, items: rejectedItems },
  ];
}

function filterGroups(groups: InboxGroup[], filter: InboxFilter): InboxGroup[] {
  if (filter === 'all') return groups.filter((group) => group.items.length > 0);
  return groups.filter((group) => group.filterKey === filter && group.items.length > 0);
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
    const resolvedAt = new Date().toISOString();
    return {
      approval: { ...approval, decidedAt: resolvedAt, updatedAt: resolvedAt },
      kind: 'conflict',
      title: 'Approval already decided',
      body: 'Another operator or process already recorded a final decision for this request. Refresh context from the audit trail before acting again.',
      traceId: getApprovalTraceId(approval),
    };
  }

  return null;
}

// Reconcile client-synthesized feedback against fresh server truth on every refetch:
// - A known outcome — either our own successful decision (`kind: 'success'`) or a
//   deterministically-verifiable terminal state we computed ourselves (`visibleStatus`
//   set, e.g. the "expired" conflict, derived from `expiresAt` vs. wall clock) — is
//   trustworthy and persists even after the server stops listing that approval at all
//   (e.g. it dropped out of the pending list).
// - An unresolved guess (`kind: 'conflict'` with no `visibleStatus`, e.g. "already
//   decided" — we don't know the actual resulting state) has zero remaining certainty
//   once the server no longer reports it as 'pending' anywhere — including when the
//   server drops it from the payload entirely — so it must be dropped rather than kept
//   indefinitely.
function reconcileFeedbackWithServer(
  approvals: ApprovalRequest[],
  feedbackById: ApprovalFeedbackById,
): ApprovalFeedbackById {
  const approvalsById = new Map(approvals.map((approval) => [approval.id, approval]));
  const reconciled: ApprovalFeedbackById = {};

  for (const [approvalId, feedback] of Object.entries(feedbackById)) {
    const serverApproval = approvalsById.get(approvalId);
    const isKnownOutcome = feedback.kind === 'success' || Boolean(feedback.visibleStatus);
    const serverStillPending = serverApproval?.status === 'pending';
    if (isKnownOutcome || serverStillPending) {
      reconciled[approvalId] = feedback;
    }
  }

  return reconciled;
}

function mergeApprovalFeedback(
  approvals: ApprovalRequest[],
  feedbackById: ApprovalFeedbackById,
): ApprovalRequest[] {
  const itemsById = new Map<string, ApprovalRequest>();

  for (const approval of approvals) {
    itemsById.set(approval.id, approval);
  }

  for (const feedback of Object.values(feedbackById)) {
    itemsById.set(feedback.approval.id, feedback.approval);
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
  const reconciledFeedbackById = reconcileFeedbackWithServer(items.approvals, approvalFeedbackById);
  const allGroups = groupInboxItems(
    mergeApprovalFeedback(items.approvals, reconciledFeedbackById),
    reconciledFeedbackById,
    items.handoffs,
    items.signals,
    items.rejected,
  );
  const visibleGroups = filterGroups(allGroups, filter);
  const totalItems = allGroups.reduce((sum, group) => sum + group.items.length, 0);
  const visibleItems = visibleGroups.reduce((sum, group) => sum + group.items.length, 0);

  return {
    screenState: resolveScreenState(isLoading, Boolean(error), totalItems),
    errorMessage: error?.message ?? 'Inbox unavailable',
    approvalFeedbackById: reconciledFeedbackById,
    visibleGroups,
    totalItems,
    visibleItemCount: visibleItems,
  };
}
