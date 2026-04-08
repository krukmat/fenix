// Activity Log — run detail screen (W5-T2)
// public status primary, runtime_status secondary diagnostics, evidence, audit, tool calls, output, per-run usage
import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack } from 'expo-router';
import { useAgentRun } from '../../../src/hooks/useCRM';
import { useRunUsage } from '../../../src/hooks/useWedge';
import { HandoffBanner } from '../../../src/components/agents/HandoffBanner';
import type { AgentRunPublicStatus, AgentRunRuntimeStatus } from '../../../src/services/api.types';
import type { ThemeColors } from '../../../src/theme/types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface RunDetail {
  id: string;
  agent_name: string;
  status: AgentRunPublicStatus;
  runtime_status?: AgentRunRuntimeStatus;
  entity_type?: string;
  entity_id?: string;
  triggered_by?: string;
  trigger_type?: string;
  inputs?: Record<string, unknown>;
  evidence_retrieved?: { source_id: string; score: number; snippet: string }[];
  reasoning_trace?: string[];
  tool_calls?: { tool_name: string; params: Record<string, unknown>; result: Record<string, unknown>; latency_ms: number }[];
  output?: unknown;
  audit_events?: { actor_id: string; action: string; timestamp: string; outcome: string }[];
  latency_ms?: number;
  cost_euros?: number;
  rejection_reason?: string;
}

interface UsageEvent {
  id: string;
  metric_name: string;
  value: number;
  recorded_at: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function getPublicStatusColor(status: AgentRunPublicStatus): string {
  const map: Record<AgentRunPublicStatus, string> = {
    completed: '#10B981', completed_with_warnings: '#F59E0B',
    abstained: '#6B7280', awaiting_approval: '#3B82F6',
    handed_off: '#8B5CF6', denied_by_policy: '#EF4444', failed: '#DC2626',
  };
  return map[status] ?? '#999';
}

function formatMs(ms?: number): string {
  if (!ms) return '—';
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`;
}

function formatOutput(output: unknown): string {
  if (output === null || output === undefined) return 'No output';
  if (typeof output === 'string') return output;
  try {
    return JSON.stringify(output, null, 2);
  } catch {
    return String(output);
  }
}

// ─── Section components ───────────────────────────────────────────────────────

function SectionHeader({ title, colors }: { title: string; colors: ThemeColors }) {
  return <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>{title}</Text>;
}

function EvidenceSection({ items, colors }: { items: RunDetail['evidence_retrieved']; colors: ThemeColors }) {
  if (!items?.length) return <Text style={{ color: colors.onSurfaceVariant }}>No evidence</Text>;
  return (
    <>
      {items.map((e, i) => (
        <View key={i} style={[styles.card, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurface, fontWeight: '500' }}>#{i + 1} — score {e.score.toFixed(2)}</Text>
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 12, marginTop: 4 }}>{e.snippet}</Text>
        </View>
      ))}
    </>
  );
}

function ToolCallsSection({ calls, colors }: { calls: RunDetail['tool_calls']; colors: ThemeColors }) {
  if (!calls?.length) return <Text style={{ color: colors.onSurfaceVariant }}>No tool calls</Text>;
  return (
    <>
      {calls.map((c, i) => (
        <View key={i} style={[styles.card, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.primary, fontWeight: '600' }}>{c.tool_name}</Text>
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 11, marginTop: 4 }}>{formatMs(c.latency_ms)}</Text>
        </View>
      ))}
    </>
  );
}

function AuditSection({ events, colors }: { events: RunDetail['audit_events']; colors: ThemeColors }) {
  if (!events?.length) return <Text style={{ color: colors.onSurfaceVariant }}>No audit events</Text>;
  return (
    <>
      {events.map((e, i) => (
        <View key={i} style={[styles.card, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurface }}>{e.action}</Text>
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 11 }}>{e.actor_id} · {new Date(e.timestamp).toLocaleString()}</Text>
        </View>
      ))}
    </>
  );
}

function UsageSection({ events, colors }: { events: UsageEvent[] | undefined; colors: ThemeColors }) {
  if (!events?.length) return <Text style={{ color: colors.onSurfaceVariant }}>No usage data</Text>;
  return (
    <>
      {events.map((e, i) => (
        <View key={i} style={[styles.usageRow, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurface }}>{e.metric_name}</Text>
          <Text style={{ color: colors.primary, fontWeight: '600' }}>{e.value}</Text>
        </View>
      ))}
    </>
  );
}

// ─── Detail content ───────────────────────────────────────────────────────────

function RunDetailContent({ run, usage, colors }: { run: RunDetail; usage: UsageEvent[] | undefined; colors: ThemeColors }) {
  return (
    <ScrollView testID="activity-run-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      {run.status === 'handed_off' && (
        <HandoffBanner
          runId={run.id}
          caseId={run.entity_type === 'case' ? run.entity_id : undefined}
          testIDPrefix="activity-detail-handoff"
        />
      )}

      {/* Public status — primary */}
      <View style={[styles.summaryCard, { backgroundColor: colors.surface }]}>
        <Text style={[styles.agentName, { color: colors.onSurface }]}>{run.agent_name}</Text>
        <View style={styles.statusRow}>
          <View testID="activity-detail-public-status" style={[styles.statusChip, { backgroundColor: getPublicStatusColor(run.status) }]}>
            <Text style={styles.statusChipText}>{run.status.replace(/_/g, ' ')}</Text>
          </View>
          {run.latency_ms ? <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>{formatMs(run.latency_ms)}</Text> : null}
          {run.cost_euros !== undefined ? <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>{run.cost_euros.toFixed(4)} €</Text> : null}
        </View>
        {/* Runtime status — secondary diagnostics */}
        {run.runtime_status ? (
          <Text testID="activity-detail-runtime-status" style={{ color: colors.onSurfaceVariant, fontSize: 11, marginTop: 6 }}>
            Runtime: {run.runtime_status}
          </Text>
        ) : null}
      </View>

      {run.status === 'denied_by_policy' ? (
        <View style={styles.section} testID="activity-detail-rejection-reason">
          <SectionHeader title="Rejection Reason" colors={colors} />
          <View style={[styles.card, { backgroundColor: colors.surface }]}>
            <Text style={{ color: colors.error }}>{run.rejection_reason ?? 'No reason provided'}</Text>
          </View>
        </View>
      ) : null}

      <View style={styles.section} testID="activity-detail-evidence">
        <SectionHeader title="Evidence" colors={colors} />
        <EvidenceSection items={run.evidence_retrieved} colors={colors} />
      </View>

      <View style={styles.section} testID="activity-detail-tool-calls">
        <SectionHeader title="Tool Calls" colors={colors} />
        <ToolCallsSection calls={run.tool_calls} colors={colors} />
      </View>

      <View style={styles.section} testID="activity-detail-output">
        <SectionHeader title="Output" colors={colors} />
        <View style={[styles.card, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurface }}>{formatOutput(run.output)}</Text>
        </View>
      </View>

      <View style={styles.section} testID="activity-detail-audit">
        <SectionHeader title="Audit Events" colors={colors} />
        <AuditSection events={run.audit_events} colors={colors} />
      </View>

      <View style={styles.section} testID="activity-detail-usage">
        <SectionHeader title="Run Usage" colors={colors} />
        <UsageSection events={usage} colors={colors} />
      </View>
    </ScrollView>
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function ActivityRunDetailScreen() {
  const colors = useColors();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useAgentRun(id);
  const run = data?.data as RunDetail | undefined;
  const { data: usageData } = useRunUsage(id, !!id);
  const usage = usageData as UsageEvent[] | undefined;

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="activity-run-detail-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading run...</Text>
      </View>
    );
  }

  if (error || !run) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="activity-run-detail-error">
        <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Run not found'}</Text>
      </View>
    );
  }

  return (
    <>
      <Stack.Screen options={{ title: run.agent_name }} />
      <RunDetailContent run={run} usage={usage} colors={colors} />
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  summaryCard: { margin: 16, padding: 16, borderRadius: 8, elevation: 2 },
  agentName: { fontSize: 18, fontWeight: '600', marginBottom: 8 },
  statusRow: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  statusChip: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 6 },
  statusChipText: { color: '#FFF', fontSize: 12, fontWeight: '600' },
  section: { paddingHorizontal: 16, paddingBottom: 12 },
  sectionTitle: { fontSize: 13, fontWeight: '700', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 8 },
  card: { padding: 12, borderRadius: 6, marginBottom: 6 },
  usageRow: { flexDirection: 'row', justifyContent: 'space-between', padding: 10, borderRadius: 6, marginBottom: 4 },
});
