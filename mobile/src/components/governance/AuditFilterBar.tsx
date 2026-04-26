import React from 'react';
import { ScrollView, StyleSheet, TouchableOpacity, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import type { AuditFilters, AuditOutcome } from '../../services/api.types';

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
              style={[styles.chip, active && { backgroundColor: colors.primary }]}
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
    paddingTop: 12,
    paddingHorizontal: 16,
  },
  scrollContent: {
    paddingBottom: 4,
  },
  chip: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 16,
    marginRight: 8,
    backgroundColor: '#E5E7EB',
  },
  chipText: {
    fontSize: 13,
    fontWeight: '500',
  },
});
