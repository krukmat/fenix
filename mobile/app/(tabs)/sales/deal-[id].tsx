// Sales wedge — deal detail (W4-T2)
// Read-only: no edit button. Actions: Sales Brief + Copilot.
import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  TouchableOpacity,
} from 'react-native';
import { useTheme, Button } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader } from '../../../src/components/crm';
import { AgentActivitySection } from '../../../src/components/agents/AgentActivitySection';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import { useDeal } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface DealDetailData {
  id: string;
  title: string;
  status: string;
  amount?: number;
  stage?: string;
  closeDate?: string;
  accountId?: string;
  accountName?: string;
  activeSignalCount?: number;
}

type R = Record<string, unknown>;

function s(o: R | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function parseDealPayload(data: unknown): DealDetailData | undefined {
  const payload = (data ?? null) as R | null;
  if (!payload) return undefined;
  const deal = (payload.deal as R | undefined) ?? payload;
  if (!deal?.id) return undefined;
  const acct = payload.account as R | undefined;
  const signalCount = payload.active_signal_count;
  const rawAmount = deal.amount ?? deal.value;
  return {
    id: String(deal.id),
    title: s(deal, 'title') ?? s(deal, 'name') ?? 'Unnamed Deal',
    status: s(deal, 'status') ?? 'open',
    amount: typeof rawAmount === 'number' ? rawAmount : undefined,
    stage: s(deal, 'stage'),
    closeDate: s(deal, 'closeDate') ?? s(deal, 'close_date'),
    accountId: s(deal, 'accountId') ?? s(deal, 'account_id'),
    accountName: s(acct, 'name'),
    activeSignalCount: typeof signalCount === 'number' ? signalCount : 0,
  };
}

function getMetadata(d: DealDetailData) {
  return [
    { label: 'Status', value: d.status },
    { label: 'Stage', value: d.stage || 'N/A' },
    { label: 'Close Date', value: d.closeDate || 'Not set' },
  ];
}

function getStatusColor(status: string): string {
  if (status === 'won') return '#10B981';
  if (status === 'lost') return '#EF4444';
  return '#3B82F6';
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function SalesDealDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const rawId = Array.isArray(params.id) ? params.id[0] : params.id;
  // Route param is "deal-[id]" — strip prefix to get real deal id for API
  const dealId = rawId.startsWith('deal-') ? rawId.slice(5) : rawId;
  const { data, isLoading, error } = useDeal(dealId);
  const dealData = parseDealPayload(data);

  if (isLoading) {
    return (
      <View
        style={[styles.centered, { backgroundColor: colors.background }]}
        testID="sales-deal-detail-loading"
      >
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading deal...</Text>
      </View>
    );
  }

  if (error || !dealData) {
    return (
      <View
        style={[styles.centered, { backgroundColor: colors.background }]}
        testID="sales-deal-detail-error"
      >
        <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Deal not found'}</Text>
      </View>
    );
  }

  return (
    <>
      <Stack.Screen options={{ title: dealData.title }} />
      <ScrollView
        testID="sales-deal-detail-screen"
        style={[styles.container, { backgroundColor: colors.background }]}
      >
        <View style={[styles.statusBanner, { backgroundColor: getStatusColor(dealData.status) }]}>
          <Text style={styles.statusText}>STATUS: {dealData.status.toUpperCase()}</Text>
        </View>

        <CRMDetailHeader
          title={dealData.title}
          subtitle={dealData.accountName}
          metadata={getMetadata(dealData)}
          testIDPrefix="sales-deal-detail"
        />

        {dealData.amount !== undefined && (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Value</Text>
            <View style={[styles.card, { backgroundColor: colors.surface }]} testID="sales-deal-amount">
              <Text style={{ color: colors.onSurface, fontSize: 24, fontWeight: '700' }}>
                ${dealData.amount.toLocaleString()}
              </Text>
            </View>
          </View>
        )}

        {dealData.stage && (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Stage</Text>
            <View style={[styles.card, { backgroundColor: colors.surface }]} testID="sales-deal-stage">
              <Text style={{ color: colors.onSurface }}>{dealData.stage}</Text>
            </View>
          </View>
        )}

        {dealData.accountId && (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Account</Text>
            <TouchableOpacity
              style={[styles.card, { backgroundColor: colors.surface }]}
              onPress={() => router.push(`/sales/${dealData.accountId}` as any)}
            >
              <Text style={{ color: colors.onSurface, fontWeight: '500' }}>
                {dealData.accountName || 'View Account'}
              </Text>
            </TouchableOpacity>
          </View>
        )}

        {/* Actions — W4-T3: brief / W4-T4: copilot */}
        <View style={styles.section}>
          <Button
            mode="contained"
            testID="sales-deal-brief-button"
            style={styles.actionButton}
            onPress={() =>
              router.push({
                pathname: `/sales/${rawId}/brief` as any,
                params: { entity_type: 'deal', entity_id: dealData.id },
              })
            }
          >
            Sales Brief
          </Button>
          <Button
            mode="outlined"
            testID="sales-deal-copilot-button"
            onPress={() =>
              router.push({
                pathname: `/sales/${rawId}/copilot` as any,
                params: { entity_type: 'deal', entity_id: dealData.id },
              })
            }
          >
            Open Copilot
          </Button>
        </View>

        <AgentActivitySection
          entityType="deal"
          entityId={dealData.id}
          testIDPrefix="sales-deal-detail"
        />
        <EntitySignalsSection
          entityType="deal"
          entityId={dealData.id}
          testIDPrefix="sales-deal-detail"
        />
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  statusBanner: { padding: 8, alignItems: 'center' },
  statusText: { color: '#FFF', fontWeight: '600', fontSize: 14 },
  section: { padding: 16 },
  sectionTitle: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 16, borderRadius: 8 },
  actionButton: { marginBottom: 12 },
});
