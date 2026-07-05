import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text } from 'react-native-paper';
import { brandColors } from '../../theme/colors';
import { radius, spacing, elevation } from '../../theme/spacing';
import { typography } from '../../theme/typography';

export interface AbstentionPanelProps {
  eyebrow: string;
  reason: string;
  escalateText?: string;
  testID?: string;
  manualLaneTestID?: string;
}

export function AbstentionPanel({
  eyebrow,
  reason,
  escalateText = 'Escalate or handle manually with the evidence below.',
  testID = 'abstention-panel',
  manualLaneTestID,
}: AbstentionPanelProps) {
  return (
    <View style={styles.abstentionPanel} testID={testID}>
      <Text style={styles.trustEyebrow}>{eyebrow}</Text>
      <Text style={styles.abstentionText}>{reason}</Text>
      <View style={styles.manualLane} testID={manualLaneTestID ?? `${testID}-manual-lane`}>
        <Text style={styles.manualLaneLabel}>Recommended next step</Text>
        <Text style={styles.manualLaneText}>{escalateText}</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  abstentionPanel: {
    borderRadius: radius.md,
    padding: spacing.base,
    gap: spacing.sm,
    backgroundColor: brandColors.surfaceVariant,
    ...elevation.card,
  },
  trustEyebrow: {
    color: brandColors.onSurfaceVariant,
    ...typography.eyebrow,
  },
  abstentionText: {
    color: brandColors.onSurface,
  },
  manualLane: {
    borderRadius: radius.sm,
    padding: spacing.sm,
    gap: spacing.xs,
    borderWidth: 1,
    borderColor: brandColors.outline,
    backgroundColor: brandColors.surface,
  },
  manualLaneLabel: {
    color: brandColors.onSurfaceVariant,
    ...typography.labelMD,
  },
  manualLaneText: {
    color: brandColors.onSurface,
  },
});
