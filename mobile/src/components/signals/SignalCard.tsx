// UC-A5/B4: Signal card for list display

import React, { useState } from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, IconButton, Dialog, Portal, Button, useTheme } from 'react-native-paper';
import type { Signal } from '../../services/api';
import { brandColors } from '../../theme/colors';
import { confidenceGlowStyle, getConfidenceColor, getConfidenceLabel } from '../../theme/semantic';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

interface SignalCardProps {
  signal: Signal;
  onDismiss: (id: string) => void;
  onPress?: (signal: Signal) => void;
  testIDPrefix?: string;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function DismissDialog({
  visible,
  testIDPrefix,
  onCancel,
  onConfirm,
}: {
  visible: boolean;
  testIDPrefix: string;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <Portal>
      <Dialog visible={visible} onDismiss={onCancel} testID={`${testIDPrefix}-dismiss-dialog`}>
        <Dialog.Title>Dismiss signal?</Dialog.Title>
        <Dialog.Content>
          <Text>This signal will be marked as dismissed.</Text>
        </Dialog.Content>
        <Dialog.Actions>
          <Button onPress={onCancel} testID={`${testIDPrefix}-dismiss-cancel`}>Cancel</Button>
          <Button onPress={onConfirm} testID={`${testIDPrefix}-dismiss-confirm`}>Dismiss</Button>
        </Dialog.Actions>
      </Dialog>
    </Portal>
  );
}

export function SignalCard({ signal, onDismiss, onPress, testIDPrefix = 'signal-card' }: SignalCardProps) {
  const [confirmVisible, setConfirmVisible] = useState(false);
  const theme = useTheme();
  const color = getConfidenceColor(signal.confidence);

  const handleDismissConfirm = () => {
    setConfirmVisible(false);
    onDismiss(signal.id);
  };

  return (
    <>
      <Card
        testID={testIDPrefix}
        style={[styles.card, confidenceGlowStyle(signal.confidence)]}
        onPress={onPress ? () => onPress(signal) : undefined}
      >
        <Card.Content>
          <View style={styles.header}>
            <View style={styles.titleRow}>
              <Text variant="labelLarge" testID={`${testIDPrefix}-type`}>{signal.signal_type}</Text>
              <View testID={`${testIDPrefix}-confidence`} style={[styles.confidenceBadge, { backgroundColor: color }]}>
                <Text style={styles.confidenceText}>
                  {`${getConfidenceLabel(signal.confidence)} ${(signal.confidence * 100).toFixed(0)}%`}
                </Text>
              </View>
            </View>
            <IconButton icon="close" size={18} testID={`${testIDPrefix}-dismiss-btn`}
              onPress={() => setConfirmVisible(true)} />
          </View>
          <Text variant="bodySmall" style={[styles.entity, { color: theme.colors.onSurfaceVariant }]}
            testID={`${testIDPrefix}-entity`}>
            {`${signal.entity_type} · ${signal.entity_id}`}
          </Text>
          <Text variant="bodyMedium" numberOfLines={2} style={styles.snippet} testID={`${testIDPrefix}-snippet`}>
            {typeof signal.metadata?.['summary'] === 'string' ? signal.metadata['summary'] : signal.signal_type}
          </Text>
          <Text variant="labelSmall" style={{ color: theme.colors.onSurfaceVariant }}
            testID={`${testIDPrefix}-timestamp`}>
            {formatTimestamp(signal.created_at)}
          </Text>
        </Card.Content>
      </Card>
      <DismissDialog visible={confirmVisible} testIDPrefix={testIDPrefix}
        onCancel={() => setConfirmVisible(false)} onConfirm={handleDismissConfirm} />
    </>
  );
}

const styles = StyleSheet.create({
  card: { marginBottom: spacing.sm, marginHorizontal: spacing.base },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-start' },
  titleRow: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: spacing.sm, flexWrap: 'wrap' },
  confidenceBadge: { borderRadius: radius.full, paddingHorizontal: spacing.sm, paddingVertical: spacing.xs },
  confidenceText: { color: brandColors.onError, ...typography.labelMD },
  entity: { marginTop: 2, marginBottom: spacing.xs },
  snippet: { marginBottom: spacing.sm },
});
