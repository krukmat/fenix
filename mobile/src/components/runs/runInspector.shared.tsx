import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

export function Section({
  title,
  children,
  testID,
}: {
  title: string;
  children: React.ReactNode;
  testID: string;
}) {
  const { colors } = useTheme();
  return (
    <View style={sharedStyles.section} testID={testID}>
      <Text style={[sharedStyles.sectionTitle, { color: colors.onSurface }]}>{title}</Text>
      {children}
    </View>
  );
}

export function EmptyState({ message }: { message: string }) {
  const { colors } = useTheme();
  return <Text style={{ color: colors.onSurfaceVariant }}>{message}</Text>;
}

export function MetaChip({ label, tone = 'neutral' }: { label: string; tone?: 'neutral' | 'status' }) {
  const { colors } = useTheme();
  return (
    <View
      style={[
        sharedStyles.metaChip,
        tone === 'status'
          ? { backgroundColor: brandColors.primaryContainer }
          : { backgroundColor: colors.surfaceVariant },
      ]}
    >
      <Text
        style={[
          sharedStyles.metaChipText,
          tone === 'status' ? { color: brandColors.onPrimaryContainer } : { color: colors.onSurface },
        ]}
      >
        {label}
      </Text>
    </View>
  );
}

export function SummaryMetric({
  label,
  value,
  monospace = false,
}: {
  label: string;
  value: string;
  monospace?: boolean;
}) {
  const { colors } = useTheme();
  return (
    <View style={sharedStyles.summaryMetric}>
      <Text style={[sharedStyles.summaryMetricLabel, { color: colors.onSurfaceVariant }]}>{label}</Text>
      <Text
        style={[
          monospace ? typography.monoLG : sharedStyles.summaryMetricValue,
          { color: colors.onSurface },
        ]}
      >
        {value}
      </Text>
    </View>
  );
}

export const sharedStyles = StyleSheet.create({
  section: {
    paddingHorizontal: spacing.base,
    paddingTop: spacing.base,
  },
  sectionTitle: {
    ...typography.eyebrow,
    marginBottom: spacing.sm,
  },
  panel: {
    borderRadius: radius.lg,
    padding: spacing.base,
    gap: spacing.sm,
    marginBottom: spacing.xs,
  },
  subpanel: {
    borderWidth: 1,
    borderRadius: radius.md,
    padding: spacing.md,
    marginBottom: spacing.sm,
    gap: spacing.sm,
  },
  headerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: spacing.sm,
  },
  metaRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.xs,
    alignItems: 'center',
  },
  metaChip: {
    borderRadius: radius.full,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
  },
  metaChipText: {
    ...typography.labelMD,
  },
  summaryGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.md,
  },
  summaryMetric: {
    minWidth: 132,
    flexGrow: 1,
  },
  summaryMetricLabel: {
    ...typography.labelMD,
    marginBottom: spacing.xs,
  },
  summaryMetricValue: {
    fontSize: 15,
    fontWeight: '600',
  },
  inlineAction: {
    marginTop: spacing.sm,
    alignSelf: 'flex-start',
  },
  inlineActionText: {
    fontSize: 14,
    fontWeight: '600',
  },
});
