// Task Mobile P1.4 — T4: WorkflowCard component
// Task T8.1 - Workflow status colors migrated to semantic tokens
import React from 'react';
import { TouchableOpacity, View, Text, StyleSheet } from 'react-native';
import { useTheme } from 'react-native-paper';
import { brandColors, semanticColors } from '../../theme/colors';
import { elevation, radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { Workflow, WorkflowStatus } from '../../services/api';

// Workflow status colors mapped to Command Center semantic tokens:
// - draft: onSurfaceVariant (neutral, indicates inactive/non-production)
// - testing: primary (blue, indicates active development/testing)
// - active: success (green, indicates production/active status)
// - archived: onSurfaceVariant (neutral, indicates retired status)
// All map to existing theme tokens; no exceptions needed.
const STATUS_COLORS: Record<WorkflowStatus, string> = {
  draft: brandColors.onSurfaceVariant,
  testing: brandColors.primary,
  active: semanticColors.success,
  archived: brandColors.onSurfaceVariant,
};

interface WorkflowCardProps {
  workflow: Workflow;
  onPress: () => void;
  testIDPrefix?: string;
}

export function WorkflowCard({ workflow, onPress, testIDPrefix = 'workflow-card' }: WorkflowCardProps) {
  const theme = useTheme();
  const statusColor = STATUS_COLORS[workflow.status];

  return (
    <TouchableOpacity
      style={[styles.card, { backgroundColor: theme.colors.surface }]}
      onPress={onPress}
      testID={testIDPrefix}
    >
      <View style={styles.row}>
        <Text style={[styles.name, { color: theme.colors.onSurface }]} testID={`${testIDPrefix}-name`}>
          {workflow.name}
        </Text>
        <View style={[styles.statusBadge, { backgroundColor: statusColor }]}>
          <Text style={styles.statusText} testID={`${testIDPrefix}-status`}>
            {workflow.status}
          </Text>
        </View>
      </View>
      {workflow.description ? (
        <Text
          style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
          numberOfLines={2}
          testID={`${testIDPrefix}-description`}
        >
          {workflow.description}
        </Text>
      ) : null}
      <Text style={[styles.version, { color: theme.colors.onSurfaceVariant }]} testID={`${testIDPrefix}-version`}>
        {`v${workflow.version}`}
      </Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  card: { padding: spacing.base, marginHorizontal: spacing.base, marginBottom: spacing.md, borderRadius: radius.md, ...elevation.card },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: spacing.xs },
  name: { ...typography.headingMD, flex: 1 },
  statusBadge: { paddingHorizontal: spacing.sm, paddingVertical: spacing.xs, borderRadius: radius.full, marginLeft: spacing.sm },
  statusText: { ...typography.labelMD, color: brandColors.onError },
  description: { ...typography.monoSM, marginTop: spacing.xs, marginBottom: spacing.xs },
  version: { ...typography.monoSM, marginTop: spacing.xs },
});
