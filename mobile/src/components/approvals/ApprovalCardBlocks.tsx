import React from 'react';
import { StyleSheet, View } from 'react-native';
import { useRouter } from 'expo-router';
import { Button, Dialog, Portal, Text, TextInput } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import type { ApprovalRequest } from '../../services/api';
import { brandColors, semanticColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { ApprovalDecisionFeedback } from './ApprovalCard';

export interface PolicyExplanation {
  title: string;
  lines: string[];
}

export function RejectDialog({
  visible, reason, testIDPrefix, onChangeReason, onCancel, onSubmit,
}: {
  visible: boolean;
  reason: string;
  testIDPrefix: string;
  onChangeReason: (v: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
}) {
  return (
    <Portal>
      <Dialog visible={visible} onDismiss={onCancel} testID={`${testIDPrefix}-reject-dialog`}>
        <Dialog.Title>Reason for rejection</Dialog.Title>
        <Dialog.Content>
          <TextInput
            label="Reason (required)"
            value={reason}
            onChangeText={onChangeReason}
            multiline
            numberOfLines={3}
            testID={`${testIDPrefix}-reject-reason-input`}
          />
        </Dialog.Content>
        <Dialog.Actions>
          <Button onPress={onCancel} testID={`${testIDPrefix}-reject-cancel`}>Cancel</Button>
          <Button onPress={onSubmit} disabled={!reason.trim()} testID={`${testIDPrefix}-reject-submit`}>Reject</Button>
        </Dialog.Actions>
      </Dialog>
    </Portal>
  );
}

export function ApproveDialog({
  visible, comment, testIDPrefix, onChangeComment, onCancel, onSubmit,
}: {
  visible: boolean;
  comment: string;
  testIDPrefix: string;
  onChangeComment: (v: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
}) {
  return (
    <Portal>
      <Dialog visible={visible} onDismiss={onCancel} testID={`${testIDPrefix}-approve-dialog`}>
        <Dialog.Title>Add approval comment</Dialog.Title>
        <Dialog.Content>
          <TextInput
            label="Comment (optional)"
            value={comment}
            onChangeText={onChangeComment}
            multiline
            numberOfLines={3}
            testID={`${testIDPrefix}-approve-comment-input`}
          />
        </Dialog.Content>
        <Dialog.Actions>
          <Button onPress={onCancel} testID={`${testIDPrefix}-approve-cancel`}>Cancel</Button>
          <Button onPress={onSubmit} testID={`${testIDPrefix}-approve-submit`}>Approve</Button>
        </Dialog.Actions>
      </Dialog>
    </Portal>
  );
}

export function PolicyExplanationBlock({
  explanation,
  testIDPrefix,
}: {
  explanation: PolicyExplanation;
  testIDPrefix: string;
}) {
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
}: {
  feedback: ApprovalDecisionFeedback;
  testIDPrefix: string;
}) {
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
            onPress={() => router.push('/governance/audit')}
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
}: {
  disabled: boolean;
  testIDPrefix: string;
  onApprove: () => void;
  onReject: () => void;
}) {
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
}: {
  approval: ApprovalRequest;
  testIDPrefix: string;
  theme: MD3Theme;
}) {
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

export function ApprovalDecisionDialogs({
  approveVisible,
  approveComment,
  rejectVisible,
  rejectReason,
  testIDPrefix,
  onApproveCommentChange,
  onRejectReasonChange,
  onApproveCancel,
  onRejectCancel,
  onApproveSubmit,
  onRejectSubmit,
}: {
  approveVisible: boolean;
  approveComment: string;
  rejectVisible: boolean;
  rejectReason: string;
  testIDPrefix: string;
  onApproveCommentChange: (value: string) => void;
  onRejectReasonChange: (value: string) => void;
  onApproveCancel: () => void;
  onRejectCancel: () => void;
  onApproveSubmit: () => void;
  onRejectSubmit: () => void;
}) {
  return (
    <>
      <ApproveDialog
        visible={approveVisible}
        comment={approveComment}
        testIDPrefix={testIDPrefix}
        onChangeComment={onApproveCommentChange}
        onCancel={onApproveCancel}
        onSubmit={onApproveSubmit}
      />
      <RejectDialog
        visible={rejectVisible}
        reason={rejectReason}
        testIDPrefix={testIDPrefix}
        onChangeReason={onRejectReasonChange}
        onCancel={onRejectCancel}
        onSubmit={onRejectSubmit}
      />
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
