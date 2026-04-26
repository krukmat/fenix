import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

interface SignalCountBadgeProps {
  count?: number;
  testID?: string;
}

export function SignalCountBadge({ count, testID = 'signal-count-badge' }: SignalCountBadgeProps) {
  const theme = useTheme();
  if (!count || count <= 0) {
    return null;
  }

  return (
    <View testID={testID} style={[styles.badge, { backgroundColor: theme.colors.error }]}>
      <Text style={styles.text}>{count}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radius.full,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: spacing.sm,
  },
  text: {
    color: brandColors.onError,
    ...typography.labelMD,
  },
});
