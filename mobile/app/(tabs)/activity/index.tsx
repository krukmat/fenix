// Activity Log — normalized public outcome list with filter chips (W5-T1)
// Replaces shim that re-exported agents/index
import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  FlatList,
  ActivityIndicator,
  ScrollView,
} from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useAgentRuns } from '../../../src/hooks/useWedge';
import { wedgeHref } from '../../../src/utils/navigation';
import type { AgentRunPublicStatus } from '../../../src/services/api.types';
import type { ThemeColors } from '../../../src/theme/types';

// ─── Types ────────────────────────────────────────────────────────────────────

type FilterOption = AgentRunPublicStatus | 'all';

interface RunItem {
  id: string;
  agent_name?: string;
  status: AgentRunPublicStatus;
  started_at?: string;
  latency_ms?: number;
  cost_euros?: number;
}

// ─── Filter config ────────────────────────────────────────────────────────────

const FILTERS: { label: string; value: FilterOption }[] = [
  { label: 'All', value: 'all' },
  { label: 'Completed', value: 'completed' },
  { label: 'Warnings', value: 'completed_with_warnings' },
  { label: 'Approval', value: 'awaiting_approval' },
  { label: 'Handed Off', value: 'handed_off' },
  { label: 'Denied', value: 'denied_by_policy' },
  { label: 'Abstained', value: 'abstained' },
  { label: 'Failed', value: 'failed' },
];

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function getPublicStatusColor(status: AgentRunPublicStatus): string {
  const map: Record<AgentRunPublicStatus, string> = {
    completed: '#10B981',
    completed_with_warnings: '#F59E0B',
    abstained: '#6B7280',
    awaiting_approval: '#3B82F6',
    handed_off: '#8B5CF6',
    denied_by_policy: '#EF4444',
    failed: '#DC2626',
  };
  return map[status] ?? '#999';
}

function formatLatency(ms?: number): string {
  if (!ms) return '';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function FilterChips({ active, onSelect, colors }: {
  active: FilterOption;
  onSelect: (f: FilterOption) => void;
  colors: ThemeColors;
}) {
  return (
    <ScrollView horizontal showsHorizontalScrollIndicator={false} style={[styles.chipRow, { backgroundColor: colors.surface }]}>
      {FILTERS.map((f) => (
        <TouchableOpacity
          key={f.value}
          testID={`filter-${f.value}`}
          style={[styles.chip, active === f.value && { backgroundColor: colors.primary }]}
          onPress={() => onSelect(f.value)}
        >
          <Text style={[styles.chipText, { color: active === f.value ? '#FFF' : colors.onSurfaceVariant }]}>
            {f.label}
          </Text>
        </TouchableOpacity>
      ))}
    </ScrollView>
  );
}

function RunRow({ run, colors, onPress }: { run: RunItem; colors: ThemeColors; onPress: () => void }) {
  return (
    <TouchableOpacity
      testID={`activity-run-item-${run.id}`}
      style={[styles.row, { backgroundColor: colors.surface }]}
      onPress={onPress}
    >
      <View style={styles.rowHeader}>
        <Text style={[styles.rowTitle, { color: colors.onSurface }]} numberOfLines={1}>
          {run.agent_name ?? 'Agent Run'}
        </Text>
        <View testID={`activity-status-${run.id}`} style={[styles.statusChip, { backgroundColor: getPublicStatusColor(run.status) }]}>
          <Text style={styles.statusText}>{run.status.replace(/_/g, ' ')}</Text>
        </View>
      </View>
      <View style={styles.rowMeta}>
        {run.latency_ms ? <Text style={[styles.metaText, { color: colors.onSurfaceVariant }]}>{formatLatency(run.latency_ms)}</Text> : null}
        {run.cost_euros !== undefined ? <Text style={[styles.metaText, { color: colors.onSurfaceVariant }]}>{run.cost_euros.toFixed(3)} €</Text> : null}
        {run.started_at ? <Text style={[styles.metaText, { color: colors.onSurfaceVariant }]}>{new Date(run.started_at).toLocaleString()}</Text> : null}
      </View>
    </TouchableOpacity>
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function ActivityLogScreen() {
  const colors = useColors();
  const router = useRouter();
  const [activeFilter, setActiveFilter] = useState<FilterOption>('all');

  const filters = activeFilter === 'all' ? undefined : { status: activeFilter };
  const { data, isLoading } = useAgentRuns(filters);

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="activity-log-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading activity...</Text>
      </View>
    );
  }

  const runs: RunItem[] = ((data as { data?: unknown[] } | null)?.data ?? []) as RunItem[];

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]} testID="activity-log-screen">
      <FilterChips active={activeFilter} onSelect={setActiveFilter} colors={colors} />
      {runs.length === 0 ? (
        <View style={styles.centered} testID="activity-log-empty">
          <Text style={{ color: colors.onSurfaceVariant }}>No runs found</Text>
        </View>
      ) : (
        <FlatList
          data={runs}
          keyExtractor={(r) => r.id}
          renderItem={({ item }) => (
            <RunRow run={item} colors={colors} onPress={() => router.push(wedgeHref(`/activity/${item.id}`))} />
          )}
          contentContainerStyle={styles.listContent}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  chipRow: { paddingHorizontal: 12, paddingVertical: 8, flexGrow: 0 },
  chip: { paddingHorizontal: 12, paddingVertical: 6, borderRadius: 16, marginRight: 8, backgroundColor: '#E5E7EB' },
  chipText: { fontSize: 13, fontWeight: '500' },
  listContent: { padding: 16 },
  row: { padding: 16, borderRadius: 8, marginBottom: 10, elevation: 1 },
  rowHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 },
  rowTitle: { fontSize: 15, fontWeight: '600', flex: 1, marginRight: 8 },
  statusChip: { paddingHorizontal: 8, paddingVertical: 3, borderRadius: 10 },
  statusText: { color: '#FFF', fontSize: 11, fontWeight: '600' },
  rowMeta: { flexDirection: 'row', gap: 12 },
  metaText: { fontSize: 12 },
});
