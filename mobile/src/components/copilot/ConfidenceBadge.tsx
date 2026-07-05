import * as React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text } from 'react-native-paper';
import { brandColors, semanticColors } from '../../theme/colors';
import { chipShape } from '../../theme/spacing';
import { typography } from '../../theme/typography';

export type ConfidenceTier = 'high' | 'medium' | 'low';

const CONFIDENCE_LABELS: Record<ConfidenceTier, string> = {
  high: 'High',
  medium: 'Medium',
  low: 'Low',
};

const CONFIDENCE_COLORS: Record<ConfidenceTier, string> = {
  high: semanticColors.confidenceHigh,
  medium: semanticColors.confidenceMed,
  low: semanticColors.confidenceLow,
};

export interface ConfidenceBadgeProps {
  confidence?: ConfidenceTier;
  testID?: string;
}

export function ConfidenceBadge({ confidence, testID = 'confidence-badge' }: ConfidenceBadgeProps) {
  if (!confidence) {
    return null;
  }

  const confidenceLabel = CONFIDENCE_LABELS[confidence];
  const confidenceColor = CONFIDENCE_COLORS[confidence];

  return (
    <View style={[styles.confidenceBadge, { backgroundColor: confidenceColor }]} testID={testID}>
      <Text style={styles.confidenceText}>{`${confidenceLabel} confidence`}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  confidenceBadge: {
    alignSelf: 'flex-start',
    ...chipShape,
  },
  confidenceText: {
    color: brandColors.onSurface,
    ...typography.labelMD,
  },
});
