// Task 4.3 — Reusable CRM Detail Header Component

import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from 'react-native-paper';
import { elevation, radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
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
    padding: spacing.base,
    borderRadius: radius.md,
    margin: spacing.base,
    ...elevation.card,
  },
  titleContainer: {
    marginBottom: spacing.md,
  },
  title: {
    ...typography.headingLG,
    marginBottom: spacing.xs,
  },
  subtitle: {
    fontSize: 14,
  },
  metadataContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.base,
  },
  metadataItem: {
    minWidth: 100,
  },
  metadataLabel: {
    ...typography.eyebrow,
    marginBottom: 2,
  },
  metadataValue: {
    fontSize: 14,
    fontWeight: '500',
  },
});

export default CRMDetailHeader;
