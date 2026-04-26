// FR-302/UC-A4: Workflow detail screen — metadata + DSL viewer + actions

import React, { useCallback } from 'react';
import { View, ScrollView, ActivityIndicator, StyleSheet } from 'react-native';
import { Text, Button, Chip, Divider, useTheme } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { DSLViewer } from '../../../src/components/workflows/DSLViewer';
import {
  useActivateWorkflow,
  useExecuteWorkflow,
  useNewVersion,
  useRollback,
  useWorkflow,
  useWorkflowVersions,
} from '../../../src/hooks/useAgentSpec';
import { workflowApi } from '../../../src/services/api';
import type { WorkflowStatus, Workflow } from '../../../src/services/api';
import { brandColors, semanticColors } from '../../../src/theme/colors';
import { radius, spacing } from '../../../src/theme/spacing';
import { typography } from '../../../src/theme/typography';

const STATUS_COLORS: Record<WorkflowStatus, string> = {
  draft: brandColors.onSurfaceVariant,
  testing: brandColors.primary,
  active: semanticColors.success,
  archived: brandColors.onSurfaceVariant,
};

function WorkflowActions({
  workflow,
  onActivate,
  onExecute,
  onVerify,
  onEdit,
  onNewVersion,
  activatePending,
  executePending,
  newVersionPending,
}: {
  workflow: Workflow;
  onActivate: () => void;
  onExecute: () => void;
  onVerify: () => void;
  onEdit: () => void;
  onNewVersion: () => void;
  activatePending: boolean;
  executePending: boolean;
  newVersionPending: boolean;
}) {
  const isDraftOrTesting = workflow.status === 'draft' || workflow.status === 'testing';
  const isDraft = workflow.status === 'draft';
  return (
    <View style={styles.actions} testID="workflow-detail-actions">
      {isDraft && (
        <Button mode="outlined" onPress={onEdit} style={styles.actionBtn} testID="workflow-edit-btn">
          Edit Draft
        </Button>
      )}
      {isDraftOrTesting && (
        <Button mode="contained" onPress={onActivate} loading={activatePending}
          disabled={activatePending} style={styles.actionBtn} testID="workflow-activate-btn">
          Activate
        </Button>
      )}
      {workflow.status === 'active' && (
        <>
          <Button
            mode="outlined"
            onPress={onNewVersion}
            loading={newVersionPending}
            disabled={newVersionPending}
            style={styles.actionBtn}
            testID="workflow-new-version-btn"
          >
            New Version
          </Button>
        <Button mode="contained" onPress={onExecute} loading={executePending}
          disabled={executePending} style={styles.actionBtn} testID="workflow-execute-btn">
          Execute
        </Button>
        </>
      )}
      {isDraftOrTesting && (
        <Button mode="outlined" onPress={onVerify} style={styles.actionBtn} testID="workflow-verify-btn">
          Verify
        </Button>
      )}
    </View>
  );
}

function VersionHistory({
  workflow,
  versions,
  rollbackPending,
  onRollback,
}: {
  workflow: Workflow;
  versions: Workflow[];
  rollbackPending: boolean;
  onRollback: (workflowId: string) => void;
}) {
  return (
    <View testID="workflow-version-history">
      <Text variant="labelMedium" style={styles.sectionLabel}>
        Version History
      </Text>
      {versions.map((version) => {
        const statusColor = STATUS_COLORS[version.status];
        const canRollback = version.status === 'archived' && workflow.id === version.id;

        return (
          <View key={version.id} style={styles.versionRow} testID={`workflow-version-${version.id}`}>
            <View style={styles.versionMeta}>
              <Text style={styles.versionTitle}>{`${version.name} v${version.version}`}</Text>
              <Text style={styles.versionTimestamp}>{new Date(version.updated_at).toLocaleString()}</Text>
            </View>
            <View style={styles.versionActions}>
              <Chip compact style={[styles.statusChip, { backgroundColor: statusColor }]} textStyle={styles.statusText}>
                {version.status}
              </Chip>
              {canRollback ? (
                <Button
                  mode="outlined"
                  compact
                  loading={rollbackPending}
                  disabled={rollbackPending}
                  onPress={() => onRollback(version.id)}
                  testID={`workflow-rollback-btn-${version.id}`}
                >
                  Rollback
                </Button>
              ) : null}
            </View>
          </View>
        );
      })}
    </View>
  );
}

function WorkflowDetailLoading({ backgroundColor, primaryColor }: { backgroundColor: string; primaryColor: string }) {
  return (
    <View style={[styles.centered, { backgroundColor }]}>
      <ActivityIndicator size="large" color={primaryColor} />
    </View>
  );
}

function WorkflowDetailError({ backgroundColor, color, message }: { backgroundColor: string; color: string; message: string }) {
  return (
    <View style={[styles.centered, { backgroundColor }]}>
      <Text style={{ color }}>{message}</Text>
    </View>
  );
}

function WorkflowDetailBody({
  workflow,
  versions,
  statusColor,
  theme,
  activatePending,
  executePending,
  newVersionPending,
  rollbackPending,
  onActivate,
  onExecute,
  onVerify,
  onEdit,
  onNewVersion,
  onRollback,
}: {
  workflow: Workflow;
  versions: Workflow[];
  statusColor: string;
  theme: MD3Theme;
  activatePending: boolean;
  executePending: boolean;
  newVersionPending: boolean;
  rollbackPending: boolean;
  onActivate: () => void;
  onExecute: () => void;
  onVerify: () => void;
  onEdit: () => void;
  onNewVersion: () => void;
  onRollback: (workflowId: string) => void;
}) {
  return (
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
      {workflow.description ? (
        <Text variant="bodyMedium" style={styles.description} testID="workflow-detail-description">
          {workflow.description}
        </Text>
      ) : null}
      <Divider style={styles.divider} />
      <Text variant="labelMedium"
        style={[styles.sectionLabel, { color: theme.colors.onSurfaceVariant }]}>DSL Source</Text>
      <DSLViewer dsl={workflow.dsl_source} testIDPrefix="workflow-detail-dsl" />
      <Divider style={styles.divider} />
      <VersionHistory
        workflow={workflow}
        versions={versions}
        rollbackPending={rollbackPending}
        onRollback={onRollback}
      />
      <Divider style={styles.divider} />
      <WorkflowActions workflow={workflow} onActivate={onActivate} onExecute={onExecute}
        onVerify={onVerify} onEdit={onEdit} onNewVersion={onNewVersion}
        activatePending={activatePending}
        executePending={executePending}
        newVersionPending={newVersionPending} />
    </ScrollView>
  );
}

export default function WorkflowDetailScreen() {
  const theme = useTheme();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;

  const { data: workflow, isLoading, error, refetch } = useWorkflow(id);
  const { data: versions = [] } = useWorkflowVersions(id);
  const activateMutation = useActivateWorkflow();
  const executeMutation = useExecuteWorkflow();
  const newVersionMutation = useNewVersion();
  const rollbackMutation = useRollback();

  const handleActivate = useCallback(() => {
    activateMutation.mutate(id, { onSuccess: () => refetch() });
  }, [activateMutation, id, refetch]);

  const handleExecute = useCallback(() => executeMutation.mutate(id), [executeMutation, id]);
  const handleVerify = useCallback(() => workflowApi.verifyWorkflow(id), [id]);
  const handleEdit = useCallback(() => router.push(`/workflows/edit/${id}`), [id, router]);
  const handleNewVersion = useCallback(() => {
    newVersionMutation.mutate(id, {
      onSuccess: (result) => {
        refetch();
        router.push(`/workflows/${result.id}`);
      },
    });
  }, [id, newVersionMutation, refetch, router]);
  const handleRollback = useCallback((workflowId: string) => {
    rollbackMutation.mutate(workflowId, { onSuccess: () => refetch() });
  }, [refetch, rollbackMutation]);

  if (isLoading) return <WorkflowDetailLoading backgroundColor={theme.colors.background} primaryColor={theme.colors.primary} />;
  if (error || !workflow) return <WorkflowDetailError backgroundColor={theme.colors.background} color={theme.colors.error} message={error?.message ?? 'Workflow not found'} />;

  const statusColor = STATUS_COLORS[workflow.status];

  return (
    <>
      <Stack.Screen options={{ title: workflow.name }} />
      <WorkflowDetailBody
        workflow={workflow}
        versions={versions}
        statusColor={statusColor}
        theme={theme}
        activatePending={activateMutation.isPending}
        executePending={executeMutation.isPending}
        newVersionPending={newVersionMutation.isPending}
        rollbackPending={rollbackMutation.isPending}
        onActivate={handleActivate}
        onExecute={handleExecute}
        onVerify={handleVerify}
        onEdit={handleEdit}
        onNewVersion={handleNewVersion}
        onRollback={handleRollback}
      />
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: spacing.base, paddingBottom: spacing.xxl },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  headerRow: { flexDirection: 'row', alignItems: 'center', gap: radius.md, flexWrap: 'wrap', marginBottom: spacing.xs },
  name: { flex: 1 },
  statusChip: { height: 24 },
  statusText: { ...typography.labelMD, color: brandColors.onError },
  description: { marginTop: spacing.sm },
  divider: { marginVertical: spacing.base },
  sectionLabel: { ...typography.eyebrow, marginBottom: spacing.sm },
  actions: { gap: radius.md, marginTop: spacing.xs },
  actionBtn: { width: '100%' },
  versionRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: spacing.md,
    marginBottom: spacing.md,
  },
  versionMeta: { flex: 1 },
  versionTitle: { fontSize: 14, fontWeight: '600' },
  versionTimestamp: { ...typography.monoSM, color: brandColors.onSurfaceVariant, marginTop: 2 },
  versionActions: { alignItems: 'flex-end', gap: spacing.sm },
});
