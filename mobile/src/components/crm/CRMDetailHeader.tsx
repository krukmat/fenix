// Task 4.3 — Reusable CRM Detail Header Component

import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from 'react-native-paper';
import type { ThemeColors } from '../../theme/types';

export interface CRMDetailHeaderProps {
  title: string;
  subtitle?: string;
  metadata?: { label: string; value: string }[];
  testIDPrefix?: string;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

export function CRMDetailHeader({ title, subtitle, metadata, testIDPrefix = 'crm-detail' }: CRMDetailHeaderProps) {
  const colors = useThemeColors();

  return (
    <View style={[styles.container, { backgroundColor: colors.surface }]} testID={`${testIDPrefix}-header`}>
      <View style={styles.titleContainer}>
        <Text style={[styles.title, { color: colors.onSurface }]}>{title}</Text>
        {subtitle && (
          <Text style={[styles.subtitle, { color: colors.onSurfaceVariant }]}>{subtitle}</Text>
        )}
      </View>
      {metadata && metadata.length > 0 && (
        <View style={styles.metadataContainer}>
          {metadata.map((item) => (
            <View key={item.label} style={styles.metadataItem}>
              <Text style={[styles.metadataLabel, { color: colors.onSurfaceVariant }]}>{item.label}</Text>
              <Text style={[styles.metadataValue, { color: colors.onSurface }]}>{item.value}</Text>
            </View>
          ))}
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
    borderRadius: 12,
    margin: 16,
    elevation: 2,
  },
  titleContainer: {
    marginBottom: 12,
  },
  title: {
    fontSize: 24,
    fontWeight: '600',
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 14,
  },
  metadataContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 16,
  },
  metadataItem: {
    minWidth: 100,
  },
  metadataLabel: {
    fontSize: 12,
    marginBottom: 2,
    textTransform: 'uppercase',
  },
  metadataValue: {
    fontSize: 14,
    fontWeight: '500',
  },
});

export default CRMDetailHeader;
