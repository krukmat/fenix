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

export { getPolicyExplanation } from './ApprovalPolicyExplanation';

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
