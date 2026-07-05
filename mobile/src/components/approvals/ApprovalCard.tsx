// UC-A7/B6: Approval card with countdown, approve/reject, reason dialog
// FR-071: approval request display

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from 'react-native-paper';
import type { ApprovalRequest, ApprovalStatus } from '../../services/api';
import { ApprovalCardPanel, getPolicyExplanation, resolveApprovalDisplayState } from './ApprovalCardPanel';
import { ApprovalDecisionDialogs } from './ApprovalDialogs';

export interface ApprovalDecisionFeedback {
  kind: 'success' | 'conflict';
  title: string;
  body: string;
  visibleStatus?: ApprovalStatus;
  comment?: string;
  traceId?: string;
}

interface ApprovalCardProps {
  approval: ApprovalRequest;
  onApprove: (id: string, comment?: string) => void;
  onReject: (id: string, reason: string) => void;
  testIDPrefix?: string;
  disabled?: boolean;
  decisionFeedback?: ApprovalDecisionFeedback | null;
}

function formatCountdown(expiresAt: string): string {
  const diff = new Date(expiresAt).getTime() - Date.now();
  if (diff <= 0) return 'Expired';
  const totalMinutes = Math.floor(diff / 60_000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function ApprovalCard({
  approval,
  onApprove,
  onReject,
  testIDPrefix = 'approval-card',
  disabled = false,
  decisionFeedback = null,
}: ApprovalCardProps) {
  const theme = useTheme();
  const [countdown, setCountdown] = useState(() => formatCountdown(approval.expiresAt));
  const [approveDialogVisible, setApproveDialogVisible] = useState(false);
  const [approveComment, setApproveComment] = useState('');
  const [rejectDialogVisible, setRejectDialogVisible] = useState(false);
  const [reason, setReason] = useState('');
  const isExpired = countdown === 'Expired';
  const policyExplanation = getPolicyExplanation(approval.payload);
  const { visibleStatus, statusColor, countdownLabel } = resolveApprovalDisplayState(
    approval.status,
    decisionFeedback?.visibleStatus,
    isExpired,
    countdown,
    theme.colors.error,
    theme.colors.primary,
  );
  const isTerminal = visibleStatus !== 'pending';

  useEffect(() => {
    const interval = setInterval(() => setCountdown(formatCountdown(approval.expiresAt)), 60_000);
    return () => clearInterval(interval);
  }, [approval.expiresAt]);

  const handleApproveSubmit = useCallback(() => {
    const trimmedComment = approveComment.trim();
    setApproveDialogVisible(false);
    onApprove(approval.id, trimmedComment || undefined);
    setApproveComment('');
  }, [approval.id, approveComment, onApprove]);

  const handleRejectSubmit = useCallback(() => {
    if (!reason.trim()) return;
    setRejectDialogVisible(false);
    onReject(approval.id, reason.trim());
    setReason('');
  }, [approval.id, onReject, reason]);

  return (
    <>
      <ApprovalCardPanel
        approval={approval}
        theme={theme}
        testIDPrefix={testIDPrefix}
        disabled={disabled}
        visibleStatus={visibleStatus}
        statusColor={statusColor}
        countdownLabel={countdownLabel}
        isTerminal={isTerminal}
        isExpired={isExpired}
        policyExplanation={policyExplanation}
        decisionFeedback={decisionFeedback}
        onApprove={() => setApproveDialogVisible(true)}
        onReject={() => setRejectDialogVisible(true)}
      />
      <ApprovalDecisionDialogs
        approveVisible={approveDialogVisible}
        approveComment={approveComment}
        rejectVisible={rejectDialogVisible}
        rejectReason={reason}
        testIDPrefix={testIDPrefix}
        onApproveCommentChange={setApproveComment}
        onRejectReasonChange={setReason}
        onApproveCancel={() => { setApproveDialogVisible(false); setApproveComment(''); }}
        onRejectCancel={() => { setRejectDialogVisible(false); setReason(''); }}
        onApproveSubmit={handleApproveSubmit}
        onRejectSubmit={handleRejectSubmit}
      />
    </>
  );
}
