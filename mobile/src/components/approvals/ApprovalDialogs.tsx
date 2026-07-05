import React from 'react';
import { Button, Dialog, Portal, TextInput } from 'react-native-paper';

interface WithTestIDPrefix {
  testIDPrefix: string;
}

interface RejectDialogProps extends WithTestIDPrefix {
  visible: boolean;
  reason: string;
  onChangeReason: (v: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
}

interface ApproveDialogProps extends WithTestIDPrefix {
  visible: boolean;
  comment: string;
  onChangeComment: (v: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
}

interface ApprovalDecisionDialogsProps extends WithTestIDPrefix {
  approveVisible: boolean;
  approveComment: string;
  rejectVisible: boolean;
  rejectReason: string;
  onApproveCommentChange: (value: string) => void;
  onRejectReasonChange: (value: string) => void;
  onApproveCancel: () => void;
  onRejectCancel: () => void;
  onApproveSubmit: () => void;
  onRejectSubmit: () => void;
}

export function RejectDialog({
  visible, reason, testIDPrefix, onChangeReason, onCancel, onSubmit,
}: RejectDialogProps) {
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
}: ApproveDialogProps) {
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
}: ApprovalDecisionDialogsProps) {
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
