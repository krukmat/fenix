// UC-A7/B6: Approval card with countdown, approve/reject, reason dialog
// FR-071: approval request display

import React, { useState, useEffect, useCallback } from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Button, Dialog, Portal, TextInput, useTheme } from 'react-native-paper';
import type { ApprovalRequest } from '../../services/api';

interface ApprovalCardProps {
  approval: ApprovalRequest;
  onApprove: (id: string) => void;
  onReject: (id: string, reason: string) => void;
  testIDPrefix?: string;
  disabled?: boolean;
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

function RejectDialog({
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
          <TextInput label="Reason (required)" value={reason} onChangeText={onChangeReason}
            multiline numberOfLines={3} testID={`${testIDPrefix}-reject-reason-input`} />
        </Dialog.Content>
        <Dialog.Actions>
          <Button onPress={onCancel} testID={`${testIDPrefix}-reject-cancel`}>Cancel</Button>
          <Button onPress={onSubmit} disabled={!reason.trim()} testID={`${testIDPrefix}-reject-submit`}>Reject</Button>
        </Dialog.Actions>
      </Dialog>
    </Portal>
  );
}

export function ApprovalCard({
  approval,
  onApprove,
  onReject,
  testIDPrefix = 'approval-card',
  disabled = false,
}: ApprovalCardProps) {
  const theme = useTheme();
  const [countdown, setCountdown] = useState(() => formatCountdown(approval.expires_at));
  const [rejectDialogVisible, setRejectDialogVisible] = useState(false);
  const [reason, setReason] = useState('');
  const isExpired = countdown === 'Expired';

  useEffect(() => {
    const interval = setInterval(() => setCountdown(formatCountdown(approval.expires_at)), 60_000);
    return () => clearInterval(interval);
  }, [approval.expires_at]);

  const handleApprove = useCallback(() => onApprove(approval.id), [approval.id, onApprove]);

  const handleRejectSubmit = useCallback(() => {
    if (!reason.trim()) return;
    setRejectDialogVisible(false);
    onReject(approval.id, reason.trim());
    setReason('');
  }, [approval.id, onReject, reason]);

  return (
    <>
      <Card testID={testIDPrefix} style={styles.card}>
        <Card.Content>
          <View style={styles.headerRow}>
            <Text variant="titleSmall" style={styles.action} testID={`${testIDPrefix}-action`}>
              {approval.action}
            </Text>
            <Text variant="labelSmall"
              style={[styles.countdown, { color: isExpired ? theme.colors.error : theme.colors.onSurfaceVariant }]}
              testID={`${testIDPrefix}-countdown`}>
              {isExpired ? 'Expired' : `Expires in ${countdown}`}
            </Text>
          </View>
          {approval.resource_type && (
            <Text variant="bodySmall" style={{ color: theme.colors.onSurfaceVariant }}
              testID={`${testIDPrefix}-resource`}>
              {`${approval.resource_type}${approval.resource_id ? ` · ${approval.resource_id}` : ''}`}
            </Text>
          )}
          {approval.reason && (
            <Text variant="bodyMedium" style={styles.reason} testID={`${testIDPrefix}-reason`}>
              {approval.reason}
            </Text>
          )}
          {!isExpired && (
            <View style={styles.actions}>
              <Button mode="contained" onPress={handleApprove} style={styles.approveBtn} disabled={disabled}
                testID={`${testIDPrefix}-approve`}>Approve</Button>
              <Button mode="outlined" onPress={() => setRejectDialogVisible(true)} disabled={disabled}
                testID={`${testIDPrefix}-reject`}>Reject</Button>
            </View>
          )}
        </Card.Content>
      </Card>
      <RejectDialog visible={rejectDialogVisible} reason={reason} testIDPrefix={testIDPrefix}
        onChangeReason={setReason} onCancel={() => { setRejectDialogVisible(false); setReason(''); }}
        onSubmit={handleRejectSubmit} />
    </>
  );
}

const styles = StyleSheet.create({
  card: { marginBottom: 8, marginHorizontal: 16 },
  headerRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 },
  action: { flex: 1, marginRight: 8 },
  countdown: { fontSize: 12 },
  reason: { marginTop: 8, marginBottom: 4 },
  actions: { flexDirection: 'row', gap: 8, marginTop: 12 },
  approveBtn: { flex: 1 },
});
