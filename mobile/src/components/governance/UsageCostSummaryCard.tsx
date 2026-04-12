import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import type { UsageCostSummary } from '../../services/api.types';

interface UsageCostSummaryCardProps {
  summary: UsageCostSummary;
  testIDPrefix?: string;
}

function formatCost(value: number): string {
  return `€${value.toFixed(5)}`;
}

function StatCell({
  label,
  value,
  testID,
}: {
  label: string;
  value: string | number;
  testID: string;
}) {
  const { colors } = useTheme();

  return (
    <View style={styles.statCell}>
      <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>
        {label}
      </Text>
      <Text testID={testID} variant="titleSmall" style={{ color: colors.onSurface, marginTop: 4 }}>
        {value}
      </Text>
    </View>
  );
}

export function UsageCostSummaryCard({
  summary,
  testIDPrefix = 'usage-summary',
}: UsageCostSummaryCardProps) {
  return (
    <Card testID={`${testIDPrefix}-card`} style={styles.card}>
      <Card.Content>
        <View style={styles.row}>
          <StatCell label="Total Cost" value={formatCost(summary.totalCost)} testID={`${testIDPrefix}-total-cost`} />
          <StatCell label="Events" value={summary.eventCount} testID={`${testIDPrefix}-event-count`} />
        </View>
        <View style={styles.row}>
          <StatCell label="Input Units" value={summary.totalInputUnits} testID={`${testIDPrefix}-input-units`} />
          <StatCell label="Output Units" value={summary.totalOutputUnits} testID={`${testIDPrefix}-output-units`} />
        </View>
      </Card.Content>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: {
    marginHorizontal: 16,
    marginTop: 16,
    marginBottom: 8,
  },
  row: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 8,
  },
  statCell: {
    flex: 1,
  },
});
