import React from 'react';
import { ScrollView, StyleSheet, TouchableOpacity, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import type { AuditFilters, AuditOutcome } from '../../services/api.types';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

interface AuditFilterBarProps {
  filters: AuditFilters;
  onChange: (filters: AuditFilters) => void;
}

const OUTCOME_FILTERS: { label: string; value?: AuditOutcome }[] = [
  { label: 'All' },
  { label: 'Success', value: 'success' },
  { label: 'Denied', value: 'denied' },
  { label: 'Error', value: 'error' },
];

export function AuditFilterBar({ filters, onChange }: AuditFilterBarProps) {
  const { colors } = useTheme();

  return (
    <View style={styles.container} testID="audit-filter-bar">
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.scrollContent}>
        {OUTCOME_FILTERS.map((filter) => {
          const active = (filters.outcome ?? undefined) === filter.value;
          const testValue = filter.value ?? 'all';
          return (
            <TouchableOpacity
              key={testValue}
              testID={`audit-filter-outcome-${testValue}`}
              style={[
                styles.chip,
                { backgroundColor: active ? colors.primary : colors.surfaceVariant },
              ]}
              onPress={() => onChange({ ...filters, outcome: filter.value })}
            >
              <Text style={[styles.chipText, { color: active ? colors.onPrimary : colors.onSurfaceVariant }]}>
                {filter.label}
              </Text>
            </TouchableOpacity>
          );
        })}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingTop: spacing.md,
    paddingHorizontal: spacing.base,
  },
  scrollContent: {
    paddingBottom: spacing.xs,
  },
  chip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radius.full,
    marginRight: spacing.sm,
  },
  chipText: {
    ...typography.labelMD,
  },
});
