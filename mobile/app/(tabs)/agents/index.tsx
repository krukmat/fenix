// Task 4.5 — Agent Runs List Screen
// Task 4.8 — GAP 4: Added TriggerAgentButton for E2E tests

import React, { useMemo, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useAgentRuns } from '../../../src/hooks/useCRM';
import TriggerAgentButton from '../../../src/components/agents/TriggerAgentButton';
import { formatLatency, getStatusColor, getStatusLabel } from '../../../src/screens/agents/agentDetail.helpers';
import type { ThemeColors } from '../../../src/theme/types';

interface AgentRun {
  id: string;
  agent_name: string;
  status:
    | 'running'
    | 'success'
    | 'failed'
    | 'abstained'
    | 'partial'
    | 'escalated'
    | 'accepted'
    | 'rejected'
    | 'delegated';
  started_at: string;
  latency_ms: number;
  cost_euros: number;
  rejection_reason?: string;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function renderLoadingState(colors: ThemeColors) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]}>
      <ActivityIndicator size="large" color={colors.primary} />
      <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>
        Loading agent runs...
      </Text>
    </View>
  );
}

function renderErrorState(colors: ThemeColors, message: string, onRetry: () => void) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]}>
      <Text style={{ color: colors.error, fontSize: 16 }}>{message}</Text>
      <TouchableOpacity
        style={[styles.retryButton, { marginTop: 16, backgroundColor: colors.primary }]}
        onPress={onRetry}
      >
        <Text style={styles.retryButtonText}>Retry</Text>
      </TouchableOpacity>
    </View>
  );
}

function renderEmptyState(colors: ThemeColors) {
  return (
    <View style={styles.emptyState}>
      <Text style={[styles.emptyTitle, { color: colors.onSurfaceVariant }]}>
        No agent runs found
      </Text>
      <Text style={[styles.emptySubtitle, { color: colors.onSurfaceVariant }]}>
        Trigger an agent to get started
      </Text>
    </View>
  );
}

export default function AgentsListScreen() {
  const colors = useThemeColors();
  const router = useRouter();
  const { data, isLoading, error, refetch } = useAgentRuns();

  const allRuns: AgentRun[] = useMemo(() => {
    if (!data?.pages) return [];
    return data.pages.flatMap((p) => (p.data as AgentRun[] | undefined) ?? []);
  }, [data]);

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const renderItem = useCallback(
    ({ item }: { item: AgentRun }) => (
      <TouchableOpacity
        style={[styles.runItem, { backgroundColor: colors.surface }]}
        onPress={() => router.push(`/agents/${item.id}`)}
        testID={`agent-run-item-${item.id}`}
      >
        <View style={styles.header}>
          <Text style={[styles.agentName, { color: colors.onSurface }]}>{item.agent_name}</Text>
          <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) }]}>
            <Text style={styles.statusBadgeText}>{getStatusLabel(item.status)}</Text>
          </View>
        </View>
        <View style={styles.metrics}>
          <Text style={[styles.metric, { color: colors.onSurfaceVariant }]}>
            {formatLatency(item.latency_ms)}
          </Text>
          <Text style={[styles.metric, { color: colors.onSurfaceVariant }]}>
            {item.cost_euros.toFixed(2)} €
          </Text>
        </View>
        <Text style={[styles.timestamp, { color: colors.onSurfaceVariant }]}>
          {new Date(item.started_at).toLocaleString()}
        </Text>
        {item.status === 'rejected' && item.rejection_reason ? (
          <Text testID={`agent-run-rejection-${item.id}`} style={[styles.rejectionReason, { color: colors.error }]}>
            {item.rejection_reason}
          </Text>
        ) : null}
      </TouchableOpacity>
    ),
    [colors, router]
  );

  if (isLoading) return renderLoadingState(colors);
  if (error) return renderErrorState(colors, error.message || 'Failed to load agent runs', handleRefresh);

  return (
    <View testID="agent-runs-list-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <TriggerAgentButton />
      <View style={styles.list} testID="agent-runs-list">
        {allRuns.length === 0
          ? renderEmptyState(colors)
          : allRuns.map((run) => renderItem({ item: run }))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  list: { flex: 1 },
  runItem: {
    padding: 16,
    marginHorizontal: 16,
    marginBottom: 12,
    borderRadius: 8,
    elevation: 2,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  agentName: {
    fontSize: 16,
    fontWeight: '600',
  },
  statusBadge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 4,
  },
  statusBadgeText: {
    color: '#FFF',
    fontSize: 11,
    fontWeight: '600',
  },
  metrics: {
    flexDirection: 'row',
    gap: 16,
    marginBottom: 4,
  },
  metric: { fontSize: 13 },
  timestamp: { fontSize: 12 },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
  },
  emptyTitle: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    textAlign: 'center',
  },
  retryButton: {
    paddingVertical: 8,
    paddingHorizontal: 24,
    borderRadius: 8,
  },
  retryButtonText: {
    color: '#FFF',
    fontSize: 14,
    fontWeight: '600',
  },
  rejectionReason: {
    fontSize: 12,
    marginTop: 6,
  },
});
