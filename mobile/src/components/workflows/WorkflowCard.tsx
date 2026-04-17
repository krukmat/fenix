// Task Mobile P1.4 — T4: WorkflowCard component
import React from 'react';
import { TouchableOpacity, View, Text, StyleSheet } from 'react-native';
import { useTheme } from 'react-native-paper';
import type { Workflow, WorkflowStatus } from '../../services/api';

const STATUS_COLORS: Record<WorkflowStatus, string> = {
  draft: '#616161',
  testing: '#1565c0',
  active: '#2e7d32',
  archived: '#795548',
};

interface WorkflowCardProps {
  workflow: Workflow;
  onPress: () => void;
  testIDPrefix?: string;
}

export function WorkflowCard({ workflow, onPress, testIDPrefix = 'workflow-card' }: WorkflowCardProps) {
  const theme = useTheme();
  const statusColor = STATUS_COLORS[workflow.status] ?? '#616161';

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
  card: { padding: 16, marginHorizontal: 16, marginBottom: 12, borderRadius: 8, elevation: 2 },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 },
  name: { fontSize: 16, fontWeight: '600', flex: 1 },
  statusBadge: { paddingHorizontal: 8, paddingVertical: 4, borderRadius: 12, marginLeft: 8 },
  statusText: { color: '#ffffff', fontSize: 11, fontWeight: '600' },
  description: { fontSize: 13, marginTop: 4, marginBottom: 4 },
  version: { fontSize: 12, marginTop: 2 },
});
