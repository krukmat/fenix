// W2-T1 (mobile_wedge_harmonization_plan): Inbox tab — unified approvals, handoffs, signals
import React, { useState } from 'react';
import { useApproveApproval, useInbox, useRejectApproval } from '../../../src/hooks/useWedge';
import { InboxBody, InboxEmpty, InboxError, InboxLoading } from '../../../src/components/inbox/InboxFeed';
import type { InboxFilter, InboxRenderableItem } from '../../../src/components/inbox/InboxFeed';
import type { ApprovalRequest, HandoffPackage, Signal } from '../../../src/services/api';

function toTimestamp(value: string | undefined): number {
  if (!value) return 0;
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

function sortApprovals(approvals: ApprovalRequest[]): ApprovalRequest[] {
  return [...approvals].sort((left, right) => {
    const expiresDiff = toTimestamp(left.expiresAt) - toTimestamp(right.expiresAt);
    if (expiresDiff !== 0) return expiresDiff;
    return toTimestamp(left.created_at) - toTimestamp(right.created_at);
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

function normalizeItems(
  approvals: ApprovalRequest[],
  handoffs: { run_id: string; handoff: HandoffPackage }[],
  signals: Signal[],
): InboxRenderableItem[] {
  return [
    ...sortApprovals(approvals).map((approval) => ({ type: 'approval' as const, id: approval.id, approval })),
    ...sortHandoffs(handoffs).map(({ run_id: runId, handoff }) => ({
      type: 'handoff' as const,
      id: runId,
      runId,
      handoff,
    })),
    ...sortSignals(signals).map((signal) => ({ type: 'signal' as const, id: signal.id, signal })),
  ];
}

function filterItems(items: InboxRenderableItem[], filter: InboxFilter): InboxRenderableItem[] {
  if (filter === 'all') return items;
  return items.filter((item) => item.type === filter);
}

type ScreenState = 'loading' | 'error' | 'empty' | 'ready';

function resolveScreenState(isLoading: boolean, hasError: boolean, totalItems: number): ScreenState {
  if (isLoading) return 'loading';
  if (hasError) return 'error';
  if (totalItems === 0) return 'empty';
  return 'ready';
}

function useApprovalActions(
  approveApproval: ReturnType<typeof useApproveApproval>,
  rejectApproval: ReturnType<typeof useRejectApproval>,
  refetchInbox: () => void,
) {
  const [actionError, setActionError] = useState<string | null>(null);

  const handleApprove = (id: string) => {
    setActionError(null);
    approveApproval.mutate(
      { id },
      {
        onSuccess: () => {
          refetchInbox();
        },
        onError: (error: Error) => {
          setActionError(error.message || 'Approval decision failed');
        },
      },
    );
  };

  const handleReject = (id: string, reason: string) => {
    setActionError(null);
    rejectApproval.mutate(
      { id, reason },
      {
        onSuccess: () => {
          refetchInbox();
        },
        onError: (error: Error) => {
          setActionError(error.message || 'Approval decision failed');
        },
      },
    );
  };

  return { actionError, handleApprove, handleReject };
}

function useInboxScreenModel() {
  const inbox = useInbox();
  const approveApproval = useApproveApproval();
  const rejectApproval = useRejectApproval();
  const [filter, setFilter] = useState<InboxFilter>('all');
  const { actionError, handleApprove, handleReject } = useApprovalActions(
    approveApproval,
    rejectApproval,
    () => inbox.refetch(),
  );
  const approvals = inbox.data?.approvals ?? [];
  const handoffs = inbox.data?.handoffs ?? [];
  const signals = inbox.data?.signals ?? [];
  const allItems = normalizeItems(approvals, handoffs, signals);
  const visibleItems = filterItems(allItems, filter);
  const totalItems = allItems.length;
  const screenState = resolveScreenState(inbox.isLoading, Boolean(inbox.error), totalItems);

  return {
    screenState,
    errorMessage: inbox.error?.message ?? 'Inbox unavailable',
    refetch: inbox.refetch,
    filter,
    setFilter,
    visibleItems,
    totalItems,
    actionError,
    handleApprove,
    handleReject,
    approvalsPending: approveApproval.isPending || rejectApproval.isPending,
  };
}

export default function InboxScreen() {
  const model = useInboxScreenModel();

  if (model.screenState === 'loading') {
    return <InboxLoading />;
  }

  if (model.screenState === 'error') {
    return <InboxError message={model.errorMessage} onRetry={model.refetch} />;
  }

  if (model.screenState === 'empty') {
    return <InboxEmpty filter={model.filter} onFilterChange={model.setFilter} />;
  }

  return (
    <InboxBody
      items={model.visibleItems}
      totalItems={model.totalItems}
      filter={model.filter}
      onFilterChange={model.setFilter}
      actionError={model.actionError}
      onApprove={model.handleApprove}
      onReject={model.handleReject}
      approvalsPending={model.approvalsPending}
    />
  );
}
