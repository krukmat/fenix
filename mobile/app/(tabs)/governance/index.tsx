// Governance — usage and quota visibility screen (W5-T3)
// Read-only: recent usage + quota states from governance/summary
import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { Stack } from 'expo-router';
import { useGovernanceSummary } from '../../../src/hooks/useWedge';
import type { ThemeColors } from '../../../src/theme/types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface UsageEvent {
  id: string;
  metric_name: string;
  value: number;
  recorded_at: string;
  run_id?: string;
}

interface QuotaState {
  policyId: string;
  policyType: string;
  metricName: string;
  limitValue: number;
  resetPeriod: string;
  enforcementMode: string;
  currentValue: number;
  periodStart: string;
  periodEnd: string;
  lastEventAt?: string;
  statePresent: boolean;
}

interface GovernanceSummary {
  recentUsage: UsageEvent[];
  quotaStates: QuotaState[];
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function SectionHeader({ title, colors }: { title: string; colors: ThemeColors }) {
  return <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>{title}</Text>;
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function UsageList({ events, colors }: { events: UsageEvent[]; colors: ThemeColors }) {
  return (
    <View testID="governance-recent-usage">
      <SectionHeader title="Recent Usage" colors={colors} />
      {events.map((e, i) => (
        <View key={e.id} testID={`governance-usage-item-${i}`} style={[styles.row, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurface }}>{e.metric_name}</Text>
          <Text style={{ color: colors.primary, fontWeight: '600' }}>{e.value}</Text>
        </View>
      ))}
      {events.length === 0 && (
        <Text style={{ color: colors.onSurfaceVariant }}>No recent usage</Text>
      )}
    </View>
  );
}

function QuotaItem({ quota, index, colors }: { quota: QuotaState; index: number; colors: ThemeColors }) {
  const pct = quota.limitValue > 0 ? Math.round((quota.currentValue / quota.limitValue) * 100) : 0;
  return (
    <View testID={`governance-quota-item-${index}`} style={[styles.quotaCard, { backgroundColor: colors.surface }]}>
      <View style={styles.quotaHeader}>
        <Text style={{ color: colors.onSurface, fontWeight: '600' }}>{quota.metricName}</Text>
        <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>{quota.resetPeriod} · {quota.enforcementMode}</Text>
      </View>
      <View style={styles.quotaBar}>
        <View style={[styles.quotaFill, { width: `${Math.min(pct, 100)}%` as `${number}%`, backgroundColor: pct >= 90 ? '#EF4444' : colors.primary }]} />
      </View>
      <Text style={{ color: colors.onSurfaceVariant, fontSize: 12, marginTop: 4 }}>
        {quota.currentValue} / {quota.limitValue} ({pct}%)
      </Text>
      {!quota.statePresent && (
        <Text style={{ color: colors.onSurfaceVariant, fontSize: 11, marginTop: 2 }}>No state yet for current period</Text>
      )}
    </View>
  );
}

function QuotaList({ states, colors }: { states: QuotaState[]; colors: ThemeColors }) {
  if (states.length === 0) {
    return (
      <View testID="governance-no-quota">
        <Text style={{ color: colors.onSurfaceVariant }}>No active quota policies</Text>
      </View>
    );
  }
  return (
    <View testID="governance-quota-states">
      <SectionHeader title="Quota States" colors={colors} />
      {states.map((q, i) => <QuotaItem key={q.policyId} quota={q} index={i} colors={colors} />)}
    </View>
  );
}

function GovernanceContent({ summary, colors }: { summary: GovernanceSummary; colors: ThemeColors }) {
  return (
    <ScrollView testID="governance-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={styles.section}>
        <UsageList events={summary.recentUsage} colors={colors} />
      </View>
      <View style={styles.section}>
        <QuotaList states={summary.quotaStates} colors={colors} />
      </View>
    </ScrollView>
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function GovernanceScreen() {
  const colors = useColors();
  const { data, isLoading, error } = useGovernanceSummary();
  const summary = data as GovernanceSummary | null | undefined;

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="governance-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading governance...</Text>
      </View>
    );
  }

  if (error || !summary) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="governance-error">
        <Text style={{ color: colors.error, fontSize: 16 }}>{(error as Error | null)?.message ?? 'Governance unavailable'}</Text>
      </View>
    );
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Governance' }} />
      <GovernanceContent summary={summary} colors={colors} />
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  section: { padding: 16 },
  sectionTitle: { fontSize: 15, fontWeight: '700', marginBottom: 12, textTransform: 'uppercase', letterSpacing: 0.5 },
  row: { flexDirection: 'row', justifyContent: 'space-between', padding: 12, borderRadius: 6, marginBottom: 6 },
  quotaCard: { padding: 14, borderRadius: 8, marginBottom: 10 },
  quotaHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 },
  quotaBar: { height: 6, borderRadius: 3, backgroundColor: '#E5E7EB', overflow: 'hidden' },
  quotaFill: { height: 6, borderRadius: 3 },
});
