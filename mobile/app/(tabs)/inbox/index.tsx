// W2-T1 (mobile_wedge_harmonization_plan): Inbox tab — unified approvals, handoffs, signals
import React, { useEffect, useState } from 'react';
import { useApproveApproval, useInbox, useRejectApproval } from '../../../src/hooks/useWedge';
import { InboxBody, InboxEmpty, InboxError, InboxLoading } from '../../../src/components/inbox/InboxFeed';
import type { InboxFilter } from '../../../src/components/inbox/InboxFeed';
import type { ApprovalRequest } from '../../../src/services/api';
import {
  type ApprovalFeedbackById,
  buildApprovalConflictFeedback,
  buildApprovalFeedback,
  buildInboxModel,
  getErrorMessage,
} from './index.helpers';

function useApprovalActions(
  approveApproval: ReturnType<typeof useApproveApproval>,
  rejectApproval: ReturnType<typeof useRejectApproval>,
  findApprovalById: (id: string) => ApprovalRequest | undefined,
) {
  const [actionError, setActionError] = useState<string | null>(null);
  const [approvalFeedbackById, setApprovalFeedbackById] = useState<ApprovalFeedbackById>({});

  const handleApprove = (id: string, comment?: string) => {
    setActionError(null);
    const approval = findApprovalById(id);
    approveApproval.mutate(
      { id, reason: comment },
      {
        onSuccess: () => {
          if (!approval) return;
          setApprovalFeedbackById((current) => ({
            ...current,
            [id]: buildApprovalFeedback(approval, 'approved', comment),
          }));
        },
        onError: (error: unknown) => {
          if (!approval) {
            setActionError(getErrorMessage(error));
            return;
          }

          const conflict = buildApprovalConflictFeedback(approval, error);
          if (conflict) {
            setApprovalFeedbackById((current) => ({ ...current, [id]: conflict }));
            return;
          }

          setActionError(getErrorMessage(error));
        },
      },
    );
  };

  const handleReject = (id: string, reason: string) => {
    setActionError(null);
    const approval = findApprovalById(id);
    rejectApproval.mutate(
      { id, reason },
      {
        onSuccess: () => {
          if (!approval) return;
          setApprovalFeedbackById((current) => ({
            ...current,
            [id]: buildApprovalFeedback(approval, 'rejected', reason),
          }));
        },
        onError: (error: unknown) => {
          if (!approval) {
            setActionError(getErrorMessage(error));
            return;
          }

          const conflict = buildApprovalConflictFeedback(approval, error);
          if (conflict) {
            setApprovalFeedbackById((current) => ({ ...current, [id]: conflict }));
            return;
          }

          setActionError(getErrorMessage(error));
        },
      },
    );
  };

  return { actionError, approvalFeedbackById, handleApprove, handleReject };
}

function useExpiryTick(): number {
  const [tick, setTick] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => setTick((current) => current + 1), 60_000);
    return () => clearInterval(interval);
  }, []);

  return tick;
}

function useInboxScreenModel() {
  const inbox = useInbox();
  const approveApproval = useApproveApproval();
  const rejectApproval = useRejectApproval();
  const [filter, setFilter] = useState<InboxFilter>('all');
  const currentApprovals = inbox.data?.approvals ?? [];
  const { actionError, approvalFeedbackById, handleApprove, handleReject } = useApprovalActions(
    approveApproval,
    rejectApproval,
    (id) => currentApprovals.find((approval) => approval.id === id),
  );
  // Forces periodic re-grouping so a pending approval that crosses its expiry while this
  // screen stays open moves out of "Awaiting your approval" without requiring a refetch or
  // filter change — mirrors ApprovalCard's own 60s countdown refresh.
  useExpiryTick();

  const model = buildInboxModel(inbox.data, inbox.isLoading, inbox.error, filter, approvalFeedbackById);

  return {
    screenState: model.screenState,
    errorMessage: model.errorMessage,
    refetch: inbox.refetch,
    filter,
    setFilter,
    visibleGroups: model.visibleGroups,
    totalItems: model.totalItems,
    visibleItemCount: model.visibleItemCount,
    actionError,
    approvalFeedbackById: model.approvalFeedbackById,
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
      groups={model.visibleGroups}
      totalItems={model.totalItems}
      visibleItemCount={model.visibleItemCount}
      filter={model.filter}
      onFilterChange={model.setFilter}
      actionError={model.actionError}
      approvalFeedbackById={model.approvalFeedbackById}
      onApprove={model.handleApprove}
      onReject={model.handleReject}
      approvalsPending={model.approvalsPending}
    />
  );
}
