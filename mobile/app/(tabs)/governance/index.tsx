// Governance — usage and quota visibility screen (W5-T3)
// Wave 1 (governance_mobile_enhancement_plan): enriched UsageDetailCard, nav links to audit + usage screens.
// Read-only: recent usage + quota states from governance/summary
import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { useGovernanceSummary } from '../../../src/hooks/useWedge';
import { UsageDetailCard } from '../../../src/components/governance/UsageDetailCard';
import { wedgeHref } from '../../../src/utils/navigation';
import { semanticColors } from '../../../src/theme/colors';
import { radius, spacing } from '../../../src/theme/spacing';
import { typography } from '../../../src/theme/typography';
import type { ThemeColors } from '../../../src/theme/types';
import type { UsageEvent, QuotaStateItem, GovernanceSummary } from '../../../src/services/api.types';

const SPACE_BETWEEN = 'space-between' as const;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function SectionHeader({ title, colors }: { title: string; colors: ThemeColors }) {
  return <Text style={[styles.sectionTitle, { color: colors.onSurfaceVariant }]}>{title}</Text>;
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function UsageList({
  events,
  colors,
  onViewAll,
}: {
  events: UsageEvent[];
  colors: ThemeColors;
  onViewAll: () => void;
}) {
  return (
    <View testID="governance-recent-usage">
      <View style={styles.sectionHeaderRow}>
        <SectionHeader title="Recent Usage" colors={colors} />
        <TouchableOpacity
          testID="governance-view-all-usage"
          onPress={onViewAll}
          accessibilityRole="button"
        >
          <Text style={[styles.viewAllLink, { color: colors.primary }]}>View All</Text>
        </TouchableOpacity>
      </View>
      {events.map((e, i) => (
        <UsageDetailCard
          key={e.id}
          event={e}
          testIDPrefix={`governance-usage-item-${i}`}
        />
      ))}
      {events.length === 0 && (
        <Text style={{ color: colors.onSurfaceVariant }}>No recent usage</Text>
      )}
    </View>
  );
}

function QuotaItem({ quota, index, colors }: { quota: QuotaStateItem; index: number; colors: ThemeColors }) {
  const pct = quota.limitValue > 0 ? Math.round((quota.currentValue / quota.limitValue) * 100) : 0;
  const fillColor = pct >= 90 ? colors.error : pct >= 70 ? semanticColors.warning : colors.primary;
  return (
    <View testID={`governance-quota-item-${index}`} style={[styles.quotaCard, { backgroundColor: colors.surface }]}>
      <View style={styles.quotaHeader}>
        <Text style={{ color: colors.onSurface, fontWeight: '600' }}>{quota.metricName}</Text>
        <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>{quota.resetPeriod} · {quota.enforcementMode}</Text>
      </View>
      <View style={styles.quotaBar}>
        <View style={[styles.quotaFill, { width: `${Math.min(pct, 100)}%` as `${number}%`, backgroundColor: fillColor }]} />
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

function QuotaList({ states, colors }: { states: QuotaStateItem[]; colors: ThemeColors }) {
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
  const router = useRouter();

  return (
    <ScrollView testID="governance-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={styles.section}>
        <UsageList
          events={summary.recentUsage}
          colors={colors}
          onViewAll={() => router.push(wedgeHref('/governance/usage'))}
        />
      </View>

      {/* Audit Trail nav link — Wave 2 */}
      <View style={[styles.section, styles.auditLinkSection]}>
        <TouchableOpacity
          testID="governance-audit-trail-link"
          onPress={() => router.push(wedgeHref('/governance/audit'))}
          style={[styles.auditLinkRow, { backgroundColor: colors.surface, borderLeftWidth: 3, borderLeftColor: colors.primary }]}
          accessibilityRole="button"
        >
          <Text style={[styles.auditLinkText, { color: colors.onSurface }]}>Audit Trail</Text>
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 16 }}>→</Text>
        </TouchableOpacity>
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
  section: { padding: spacing.base },
  sectionTitle: { ...typography.eyebrow, marginBottom: spacing.md },
  sectionHeaderRow: { flexDirection: 'row', justifyContent: SPACE_BETWEEN, alignItems: 'center', marginBottom: spacing.md },
  viewAllLink: { fontSize: 13, fontWeight: '600' },
  auditLinkSection: { paddingTop: 0 },
  auditLinkRow: { flexDirection: 'row', justifyContent: SPACE_BETWEEN, alignItems: 'center', padding: spacing.base, borderRadius: radius.md },
  auditLinkText: { fontSize: 15, fontWeight: '600' },
  quotaCard: { padding: spacing.base, borderRadius: radius.md, marginBottom: radius.md },
  quotaHeader: { flexDirection: 'row', justifyContent: SPACE_BETWEEN, alignItems: 'center', marginBottom: spacing.sm },
  quotaBar: { height: 4, borderRadius: radius.xs, backgroundColor: semanticColors.confidenceLow, overflow: 'hidden' },
  quotaFill: { height: 4, borderRadius: radius.xs },
});
