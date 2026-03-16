import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';

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
    borderRadius: 11,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 6,
  },
  text: {
    color: '#FFF',
    fontSize: 11,
    fontWeight: '700',
  },
});
