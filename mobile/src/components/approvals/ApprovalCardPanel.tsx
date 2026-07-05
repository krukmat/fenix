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

function resolveCountdownLabel(
  visibleStatus: ApprovalStatus | null,
  hasUnresolvedConflict: boolean,
  isExpired: boolean,
  countdown: string,
): string {
  if (hasUnresolvedConflict) return 'Closed';
  if (visibleStatus !== null && visibleStatus !== 'pending') {
    return `${visibleStatus.charAt(0).toUpperCase()}${visibleStatus.slice(1)}`;
  }
  return isExpired ? 'Expired' : `Expires in ${countdown}`;
}

export function resolveApprovalDisplayState(
  approvalStatus: ApprovalStatus,
  feedbackStatus: ApprovalStatus | undefined,
  hasUnresolvedConflict: boolean,
  isExpired: boolean,
  countdown: string,
  errorColor: string,
  primaryColor: string,
): { visibleStatus: ApprovalStatus | null; statusColor: string; countdownLabel: string } {
  // `hasUnresolvedConflict` covers feedback (e.g. "already decided") that has no known
  // terminal ApprovalStatus to report — we must not guess one (no fabricated state).
  const visibleStatus = hasUnresolvedConflict ? null : feedbackStatus ?? (isExpired ? 'expired' : approvalStatus);
  const statusColor =
    visibleStatus === 'rejected' || visibleStatus === 'expired' || hasUnresolvedConflict
      ? errorColor
      : primaryColor;
  const countdownLabel = resolveCountdownLabel(visibleStatus, hasUnresolvedConflict, isExpired, countdown);
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
  visibleStatus: ApprovalStatus | null;
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
        {visibleStatus !== null ? (
          <ApprovalPath status={visibleStatus} testIDPrefix={`${testIDPrefix}-path`} />
        ) : null}
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
