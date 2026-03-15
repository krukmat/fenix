// Task Mobile P1.3 — Workflow card for list display

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Chip, useTheme } from 'react-native-paper';
import type { Workflow, WorkflowStatus } from '../../services/api';

interface WorkflowCardProps {
  workflow: Workflow;
  onPress?: (workflow: Workflow) => void;
  testIDPrefix?: string;
}

const STATUS_COLORS: Record<WorkflowStatus, string> = {
  draft: '#616161',
  testing: '#1565c0',
  active: '#2e7d32',
  archived: '#795548',
};

export function WorkflowCard({ workflow, onPress, testIDPrefix = 'workflow-card' }: WorkflowCardProps) {
  const theme = useTheme();
  const statusColor = STATUS_COLORS[workflow.status] ?? '#616161';

  return (
    <Card
      testID={testIDPrefix}
      style={styles.card}
      onPress={onPress ? () => onPress(workflow) : undefined}
    >
      <Card.Content>
        <View style={styles.header}>
          <Text variant="titleSmall" style={styles.name} testID={`${testIDPrefix}-name`}>
            {workflow.name}
          </Text>
          <Chip
            compact
            testID={`${testIDPrefix}-status`}
            style={[styles.statusChip, { backgroundColor: statusColor }]}
            textStyle={styles.statusText}
          >
            {workflow.status}
          </Chip>
        </View>

        <Text
          variant="labelSmall"
          style={{ color: theme.colors.onSurfaceVariant }}
          testID={`${testIDPrefix}-version`}
        >
          {`v${workflow.version}`}
        </Text>

        {workflow.description && (
          <Text
            variant="bodySmall"
            numberOfLines={2}
            style={styles.description}
            testID={`${testIDPrefix}-description`}
          >
            {workflow.description}
          </Text>
        )}
      </Card.Content>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: { marginBottom: 8, marginHorizontal: 16 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 2 },
  name: { flex: 1, marginRight: 8 },
  statusChip: { height: 24 },
  statusText: { color: '#ffffff', fontSize: 11 },
  description: { marginTop: 6 },
});
