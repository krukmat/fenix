import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import type { ApprovalRequest, ApprovalStatus } from '../../services/api';
import { ApprovalPath } from './ApprovalPath';
import {
  ApprovalActions,
  ApprovalMetadata,
  DecisionFeedbackBlock,
  type PolicyExplanation,
  PolicyExplanationBlock,
} from './ApprovalCardBlocks';
import type { ApprovalDecisionFeedback } from './ApprovalCard';

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

export function getPolicyExplanation(payload: ApprovalRequest['payload']): PolicyExplanation | null {
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

export function resolveApprovalDisplayState(
  approvalStatus: ApprovalStatus,
  feedbackStatus: ApprovalStatus | undefined,
  isExpired: boolean,
  countdown: string,
  errorColor: string,
  primaryColor: string,
): { visibleStatus: ApprovalStatus; statusColor: string; countdownLabel: string } {
  const visibleStatus = feedbackStatus ?? (isExpired ? 'expired' : approvalStatus);
  const statusColor = visibleStatus === 'rejected' || visibleStatus === 'expired' ? errorColor : primaryColor;
  const countdownLabel =
    visibleStatus !== 'pending'
      ? `${visibleStatus.charAt(0).toUpperCase()}${visibleStatus.slice(1)}`
      : isExpired
        ? 'Expired'
        : `Expires in ${countdown}`;
  return { visibleStatus, statusColor, countdownLabel };
}

export function ApprovalCardPanel({
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
