// Task Mobile P1.3 — UC-A7/B6: Approval card with countdown, approve/deny, reason dialog

import React, { useState, useEffect, useCallback } from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Button, Dialog, Portal, TextInput, useTheme } from 'react-native-paper';
import type { ApprovalRequest } from '../../services/api';

interface ApprovalCardProps {
  approval: ApprovalRequest;
  onApprove: (id: string) => void;
  onDeny: (id: string, reason: string) => void;
  testIDPrefix?: string;
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
  onDeny,
  testIDPrefix = 'approval-card',
}: ApprovalCardProps) {
  const theme = useTheme();
  const [countdown, setCountdown] = useState(() => formatCountdown(approval.expires_at));
  const [denyDialogVisible, setDenyDialogVisible] = useState(false);
  const [reason, setReason] = useState('');
  const isExpired = countdown === 'Expired';

  useEffect(() => {
    const interval = setInterval(() => {
      setCountdown(formatCountdown(approval.expires_at));
    }, 60_000);
    return () => clearInterval(interval);
  }, [approval.expires_at]);

  const handleApprove = useCallback(() => {
    onApprove(approval.id);
  }, [approval.id, onApprove]);

  const handleDenySubmit = useCallback(() => {
    if (!reason.trim()) return;
    setDenyDialogVisible(false);
    onDeny(approval.id, reason.trim());
    setReason('');
  }, [approval.id, onDeny, reason]);

  return (
    <>
      <Card testID={testIDPrefix} style={styles.card}>
        <Card.Content>
          <View style={styles.headerRow}>
            <Text variant="titleSmall" style={styles.action} testID={`${testIDPrefix}-action`}>
              {approval.action}
            </Text>
            <Text
              variant="labelSmall"
              style={[styles.countdown, { color: isExpired ? theme.colors.error : theme.colors.onSurfaceVariant }]}
              testID={`${testIDPrefix}-countdown`}
            >
              {isExpired ? 'Expired' : `Expires in ${countdown}`}
            </Text>
          </View>

          {approval.resource_type && (
            <Text
              variant="bodySmall"
              style={{ color: theme.colors.onSurfaceVariant }}
              testID={`${testIDPrefix}-resource`}
            >
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
              <Button
                mode="contained"
                onPress={handleApprove}
                style={styles.approveBtn}
                testID={`${testIDPrefix}-approve`}
              >
                Approve
              </Button>
              <Button
                mode="outlined"
                onPress={() => setDenyDialogVisible(true)}
                testID={`${testIDPrefix}-deny`}
              >
                Deny
              </Button>
            </View>
          )}
        </Card.Content>
      </Card>

      <Portal>
        <Dialog
          visible={denyDialogVisible}
          onDismiss={() => setDenyDialogVisible(false)}
          testID={`${testIDPrefix}-deny-dialog`}
        >
          <Dialog.Title>Reason for denial</Dialog.Title>
          <Dialog.Content>
            <TextInput
              label="Reason (required)"
              value={reason}
              onChangeText={setReason}
              multiline
              numberOfLines={3}
              testID={`${testIDPrefix}-deny-reason-input`}
            />
          </Dialog.Content>
          <Dialog.Actions>
            <Button
              onPress={() => { setDenyDialogVisible(false); setReason(''); }}
              testID={`${testIDPrefix}-deny-cancel`}
            >
              Cancel
            </Button>
            <Button
              onPress={handleDenySubmit}
              disabled={!reason.trim()}
              testID={`${testIDPrefix}-deny-submit`}
            >
              Deny
            </Button>
          </Dialog.Actions>
        </Dialog>
      </Portal>
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
