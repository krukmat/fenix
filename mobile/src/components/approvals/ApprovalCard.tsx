// UC-A7/B6: Approval card with countdown, approve/reject, reason dialog
// FR-071: approval request display

import React, { useState, useEffect, useCallback } from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import type { ApprovalRequest, ApprovalStatus } from '../../services/api';
import { ApprovalPath } from './ApprovalPath';
import {
  ApprovalActions,
  ApprovalDecisionDialogs,
  ApprovalMetadata,
  DecisionFeedbackBlock,
  type PolicyExplanation,
  PolicyExplanationBlock,
} from './ApprovalCardBlocks';

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

function readString(record: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string' && value.trim()) {
      return value.trim();
    }
  }
  return undefined;
}

function readStringList(record: Record<string, unknown>, ...keys: string[]): string[] {
  for (const key of keys) {
    const value = record[key];
    if (Array.isArray(value)) {
      const items = value.filter((item): item is string => typeof item === 'string' && item.trim().length > 0);
      if (items.length > 0) {
        return items.map((item) => item.trim());
      }
    }
  }
  return [];
}

function getPolicyExplanation(payload: ApprovalRequest['payload']): PolicyExplanation | null {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return null;
  }

  const record = payload as Record<string, unknown>;
  const policyId = readString(record, 'policy_id', 'policyId');
  const policyType = readString(record, 'policy_type', 'policyType');
  const decision = readString(record, 'decision', 'policy_decision', 'policyDecision');
  const reason = readString(record, 'reason', 'policy_reason', 'policyReason', 'policy_explanation', 'policyExplanation');
  const reasonCodes = readStringList(record, 'reason_codes', 'reasonCodes', 'rule_ids', 'ruleIds');

  const lines: string[] = [];
  if (policyType) lines.push(`Type: ${policyType}`);
  if (policyId) lines.push(`Policy: ${policyId}`);
  if (decision) lines.push(`Decision: ${decision}`);
  if (reason) lines.push(reason);
  if (reasonCodes.length > 0) lines.push(`Rules: ${reasonCodes.join(', ')}`);

  return lines.length > 0
    ? { title: 'Policy explanation', lines }
    : null;
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
  const visibleStatus = resolveVisibleStatus(approval.status, decisionFeedback?.visibleStatus, isExpired);
  const isTerminal = visibleStatus !== 'pending';
  const statusColor = resolveStatusColor(visibleStatus, theme.colors.error, theme.colors.primary);
  const countdownLabel = resolveCountdownLabel(visibleStatus, isExpired, countdown);

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

function ApprovalCardPanel({
  approval,
  theme,
  testIDPrefix,
  disabled,
  visibleStatus,
  statusColor,
  countdownLabel,
  isTerminal,
  isExpired,
  policyExplanation,
  decisionFeedback,
  onApprove,
  onReject,
}: {
  approval: ApprovalRequest;
  theme: MD3Theme;
  testIDPrefix: string;
  disabled: boolean;
  visibleStatus: ApprovalStatus;
  statusColor: string;
  countdownLabel: string;
  isTerminal: boolean;
  isExpired: boolean;
  policyExplanation: PolicyExplanation | null;
  decisionFeedback: ApprovalDecisionFeedback | null;
  onApprove: () => void;
  onReject: () => void;
}) {
  return (
    <Card
      testID={testIDPrefix}
      style={[
        styles.card,
        {
          borderLeftWidth: 3,
          borderLeftColor: statusColor,
        },
      ]}
    >
      <Card.Content>
        <View style={styles.headerRow}>
          <Text variant="titleSmall" style={styles.action} testID={`${testIDPrefix}-action`}>
            {approval.action}
          </Text>
          <Text
            variant="labelSmall"
            style={[styles.countdown, { color: statusColor }]}
            testID={`${testIDPrefix}-countdown`}
          >
            {countdownLabel}
          </Text>
        </View>
        <ApprovalMetadata approval={approval} testIDPrefix={testIDPrefix} theme={theme} />
        <ApprovalPath status={visibleStatus} testIDPrefix={`${testIDPrefix}-path`} />
        {policyExplanation ? (
          <PolicyExplanationBlock explanation={policyExplanation} testIDPrefix={testIDPrefix} />
        ) : null}
        {decisionFeedback ? (
          <DecisionFeedbackBlock feedback={decisionFeedback} testIDPrefix={testIDPrefix} />
        ) : null}
        {!isTerminal && !isExpired ? (
          <ApprovalActions
            disabled={disabled}
            testIDPrefix={testIDPrefix}
            onApprove={onApprove}
            onReject={onReject}
          />
        ) : null}
      </Card.Content>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: { marginBottom: 8, marginHorizontal: 16 },
  headerRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 },
  action: { flex: 1, marginRight: 8 },
  countdown: { fontSize: 12 },
});

function resolveVisibleStatus(
  approvalStatus: ApprovalStatus,
  feedbackStatus: ApprovalStatus | undefined,
  isExpired: boolean,
): ApprovalStatus {
  if (feedbackStatus) return feedbackStatus;
  if (isExpired) return 'expired';
  return approvalStatus;
}

function resolveStatusColor(visibleStatus: ApprovalStatus, errorColor: string, primaryColor: string): string {
  if (visibleStatus === 'rejected' || visibleStatus === 'expired') {
    return errorColor;
  }

  if (visibleStatus === 'pending') {
    return primaryColor;
  }

  return primaryColor;
}

function resolveCountdownLabel(visibleStatus: ApprovalStatus, isExpired: boolean, countdown: string): string {
  if (visibleStatus !== 'pending') {
    return `${visibleStatus.charAt(0).toUpperCase()}${visibleStatus.slice(1)}`;
  }

  if (isExpired) {
    return 'Expired';
  }

  return `Expires in ${countdown}`;
}
