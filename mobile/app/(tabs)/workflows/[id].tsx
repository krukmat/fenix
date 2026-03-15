// FR-302/UC-A4: Workflow detail screen — metadata + DSL viewer + actions

import React, { useCallback } from 'react';
import { View, ScrollView, ActivityIndicator, StyleSheet } from 'react-native';
import { Text, Button, Chip, Divider, useTheme } from 'react-native-paper';
import { Stack, useLocalSearchParams } from 'expo-router';
import { DSLViewer } from '../../../src/components/workflows/DSLViewer';
import { useWorkflow, useActivateWorkflow, useExecuteWorkflow } from '../../../src/hooks/useAgentSpec';
import { workflowApi } from '../../../src/services/api';
import type { WorkflowStatus, Workflow } from '../../../src/services/api';

const STATUS_COLORS: Record<WorkflowStatus, string> = {
  draft: '#616161',
  testing: '#1565c0',
  active: '#2e7d32',
  archived: '#795548',
};

function WorkflowActions({
  workflow, onActivate, onExecute, onVerify, activatePending, executePending,
}: {
  workflow: Workflow;
  onActivate: () => void;
  onExecute: () => void;
  onVerify: () => void;
  activatePending: boolean;
  executePending: boolean;
}) {
  const isDraftOrTesting = workflow.status === 'draft' || workflow.status === 'testing';
  return (
    <View style={styles.actions} testID="workflow-detail-actions">
      {isDraftOrTesting && (
        <Button mode="contained" onPress={onActivate} loading={activatePending}
          disabled={activatePending} style={styles.actionBtn} testID="workflow-activate-btn">
          Activate
        </Button>
      )}
      {workflow.status === 'active' && (
        <Button mode="contained" onPress={onExecute} loading={executePending}
          disabled={executePending} style={styles.actionBtn} testID="workflow-execute-btn">
          Execute
        </Button>
      )}
      {isDraftOrTesting && (
        <Button mode="outlined" onPress={onVerify} style={styles.actionBtn} testID="workflow-verify-btn">
          Verify
        </Button>
      )}
    </View>
  );
}

export default function WorkflowDetailScreen() {
  const theme = useTheme();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;

  const { data: workflow, isLoading, error, refetch } = useWorkflow(id);
  const activateMutation = useActivateWorkflow();
  const executeMutation = useExecuteWorkflow();

  const handleActivate = useCallback(() => {
    activateMutation.mutate(id, { onSuccess: () => refetch() });
  }, [activateMutation, id, refetch]);

  const handleExecute = useCallback(() => executeMutation.mutate(id), [executeMutation, id]);
  const handleVerify = useCallback(() => workflowApi.verifyWorkflow(id), [id]);

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: theme.colors.background }]}>
        <ActivityIndicator size="large" color={theme.colors.primary} />
      </View>
    );
  }

  if (error || !workflow) {
    return (
      <View style={[styles.centered, { backgroundColor: theme.colors.background }]}>
        <Text style={{ color: theme.colors.error }}>{error?.message ?? 'Workflow not found'}</Text>
      </View>
    );
  }

  const statusColor = STATUS_COLORS[workflow.status] ?? '#616161';

  return (
    <>
      <Stack.Screen options={{ title: workflow.name }} />
      <ScrollView style={[styles.container, { backgroundColor: theme.colors.background }]}
        contentContainerStyle={styles.content} testID="workflow-detail">
        <View style={styles.headerRow}>
          <Text variant="titleLarge" style={styles.name} testID="workflow-detail-name">{workflow.name}</Text>
          <Chip compact testID="workflow-detail-status"
            style={[styles.statusChip, { backgroundColor: statusColor }]} textStyle={styles.statusText}>
            {workflow.status}
          </Chip>
        </View>
        <Text variant="labelSmall" style={{ color: theme.colors.onSurfaceVariant }}
          testID="workflow-detail-version">{`v${workflow.version}`}</Text>
        {workflow.description && (
          <Text variant="bodyMedium" style={styles.description} testID="workflow-detail-description">
            {workflow.description}
          </Text>
        )}
        <Divider style={styles.divider} />
        <Text variant="labelMedium"
          style={[styles.sectionLabel, { color: theme.colors.onSurfaceVariant }]}>DSL Source</Text>
        <DSLViewer dsl={workflow.dsl_source} testIDPrefix="workflow-detail-dsl" />
        <Divider style={styles.divider} />
        <WorkflowActions workflow={workflow} onActivate={handleActivate} onExecute={handleExecute}
          onVerify={handleVerify} activatePending={activateMutation.isPending}
          executePending={executeMutation.isPending} />
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: 16, paddingBottom: 32 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  headerRow: { flexDirection: 'row', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginBottom: 4 },
  name: { flex: 1 },
  statusChip: { height: 24 },
  statusText: { color: '#ffffff', fontSize: 11 },
  description: { marginTop: 8 },
  divider: { marginVertical: 16 },
  sectionLabel: { marginBottom: 8, textTransform: 'uppercase', letterSpacing: 0.5 },
  actions: { gap: 10, marginTop: 4 },
  actionBtn: { width: '100%' },
});
