import React from 'react';
import { StyleSheet, View } from 'react-native';
import { useRouter } from 'expo-router';
import { Button, Text } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import type { ApprovalRequest } from '../../services/api';
import { brandColors, semanticColors } from '../../theme/colors';
import { wedgeHrefObject } from '../../utils/navigation';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { ApprovalDecisionFeedback } from './ApprovalCard';

export interface PolicyExplanation {
  title: string;
  lines: string[];
}

interface WithTestIDPrefix {
  testIDPrefix: string;
}

export function PolicyExplanationBlock({
  explanation,
  testIDPrefix,
}: WithTestIDPrefix & { explanation: PolicyExplanation }) {
  return (
    <View style={styles.policyBlock} testID={`${testIDPrefix}-policy`}>
      <Text style={styles.policyEyebrow}>{explanation.title}</Text>
      {explanation.lines.map((line) => (
        <Text
          key={line}
          variant="bodySmall"
          style={{ color: brandColors.onSurface }}
          testID={`${testIDPrefix}-policy-line`}
        >
          {line}
        </Text>
      ))}
    </View>
  );
}

export function DecisionFeedbackBlock({
  feedback,
  testIDPrefix,
}: WithTestIDPrefix & { feedback: ApprovalDecisionFeedback }) {
  const router = useRouter();

  return (
    <View
      style={[
        styles.feedbackBlock,
        feedback.kind === 'success' ? styles.feedbackBlockSuccess : styles.feedbackBlockConflict,
      ]}
      testID={`${testIDPrefix}-feedback`}
    >
      <Text style={styles.feedbackTitle} testID={`${testIDPrefix}-feedback-title`}>
        {feedback.title}
      </Text>
      <Text variant="bodySmall" style={styles.feedbackBody} testID={`${testIDPrefix}-feedback-body`}>
        {feedback.body}
      </Text>
      {feedback.comment ? (
        <Text variant="bodySmall" style={styles.feedbackMeta} testID={`${testIDPrefix}-feedback-comment`}>
          {`Comment: ${feedback.comment}`}
        </Text>
      ) : null}
      {feedback.traceId ? (
        <>
          <Text variant="bodySmall" style={styles.feedbackTrace} testID={`${testIDPrefix}-feedback-trace`}>
            {`Trace: ${feedback.traceId}`}
          </Text>
          <Button
            compact
            mode="text"
            onPress={() => router.push(wedgeHrefObject('/governance/audit', { trace_id: feedback.traceId as string }))}
            testID={`${testIDPrefix}-audit-link`}
          >
            View audit trail
          </Button>
        </>
      ) : null}
    </View>
  );
}

export function ApprovalActions({
  disabled,
  testIDPrefix,
  onApprove,
  onReject,
}: WithTestIDPrefix & { disabled: boolean; onApprove: () => void; onReject: () => void }) {
  return (
    <View style={styles.actions}>
      <Button
        mode="contained"
        onPress={onApprove}
        style={styles.approveBtn}
        disabled={disabled}
        testID={`${testIDPrefix}-approve`}
      >
        Approve
      </Button>
      <Button mode="outlined" onPress={onReject} disabled={disabled} testID={`${testIDPrefix}-reject`}>
        Reject
      </Button>
    </View>
  );
}

export function ApprovalMetadata({
  approval,
  testIDPrefix,
  theme,
}: WithTestIDPrefix & { approval: ApprovalRequest; theme: MD3Theme }) {
  return (
    <>
      {approval.resourceType ? (
        <Text
          variant="bodySmall"
          style={{ color: theme.colors.onSurfaceVariant }}
          testID={`${testIDPrefix}-resource`}
        >
          {`${approval.resourceType}${approval.resourceId ? ` · ${approval.resourceId}` : ''}`}
        </Text>
      ) : null}
      {approval.reason ? (
        <Text variant="bodyMedium" style={styles.reason} testID={`${testIDPrefix}-reason`}>
          {approval.reason}
        </Text>
      ) : null}
    </>
  );
}

const styles = StyleSheet.create({
  reason: { marginTop: 8, marginBottom: 4 },
  actions: { flexDirection: 'row', gap: 8, marginTop: 12 },
  approveBtn: { flex: 1 },
  policyBlock: {
    marginTop: spacing.sm,
    padding: spacing.sm,
    borderRadius: radius.sm,
    backgroundColor: brandColors.surfaceVariant,
    borderWidth: 1,
    borderColor: brandColors.outline,
    gap: spacing.xs,
  },
  policyEyebrow: {
    ...typography.eyebrow,
    color: semanticColors.info,
  },
  feedbackBlock: {
    marginTop: spacing.sm,
    padding: spacing.sm,
    borderRadius: radius.sm,
    borderWidth: 1,
  },
  feedbackBlockSuccess: {
    backgroundColor: brandColors.surfaceVariant,
    borderColor: semanticColors.success,
  },
  feedbackBlockConflict: {
    backgroundColor: brandColors.surfaceVariant,
    borderColor: semanticColors.warning,
  },
  feedbackTitle: {
    ...typography.labelMD,
    color: brandColors.onSurface,
    marginBottom: spacing.xs,
  },
  feedbackBody: {
    color: brandColors.onSurface,
  },
  feedbackMeta: {
    marginTop: spacing.xs,
    color: brandColors.onSurfaceVariant,
  },
  feedbackTrace: {
    marginTop: spacing.sm,
    color: brandColors.onSurfaceVariant,
    fontFamily: typography.monoSM.fontFamily,
  },
});
