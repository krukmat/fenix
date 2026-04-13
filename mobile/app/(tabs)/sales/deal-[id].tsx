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
import { useTriggerDealRiskAgent } from '../../../src/hooks/useWedge';
import { wedgeHref, wedgeHrefObject } from '../../../src/utils/navigation';
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

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function s(o: R | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

function parseAmount(deal: R): number | undefined {
  const raw = deal.amount ?? deal.value;
  return typeof raw === 'number' ? raw : undefined;
}

function parseDealCore(deal: R): Omit<DealDetailData, 'accountName' | 'activeSignalCount'> {
  return {
    id: String(deal.id),
    title: s(deal, 'title') ?? s(deal, 'name') ?? 'Unnamed Deal',
    status: s(deal, 'status') ?? 'open',
    amount: parseAmount(deal),
    stage: s(deal, 'stage'),
    closeDate: s(deal, 'closeDate') ?? s(deal, 'close_date'),
    accountId: s(deal, 'accountId') ?? s(deal, 'account_id'),
  };
}

function parseDealPayload(data: unknown): DealDetailData | undefined {
  const payload = (data ?? null) as R | null;
  if (!payload) return undefined;
  const deal = (payload.deal as R | undefined) ?? payload;
  if (!deal?.id) return undefined;
  const acct = payload.account as R | undefined;
  const signalCount = payload.active_signal_count;
  return {
    ...parseDealCore(deal),
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

// ─── Section components ───────────────────────────────────────────────────────

function DealAmountSection({ amount, colors }: { amount?: number; colors: ThemeColors }) {
  if (amount === undefined) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Value</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="sales-deal-amount">
        <Text style={{ color: colors.onSurface, fontSize: 24, fontWeight: '700' }}>${amount.toLocaleString()}</Text>
      </View>
    </View>
  );
}

function DealStageSection({ stage, colors }: { stage?: string; colors: ThemeColors }) {
  if (!stage) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Stage</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="sales-deal-stage">
        <Text style={{ color: colors.onSurface }}>{stage}</Text>
      </View>
    </View>
  );
}

function DealAccountSection({
  accountId, accountName, router, colors,
}: { accountId?: string; accountName?: string; router: ReturnType<typeof useRouter>; colors: ThemeColors }) {
  if (!accountId) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Account</Text>
      <TouchableOpacity
        style={[styles.card, { backgroundColor: colors.surface }]}
        onPress={() => router.push(wedgeHref(`/sales/${accountId}`))}
      >
        <Text style={{ color: colors.onSurface, fontWeight: '500' }}>{accountName || 'View Account'}</Text>
      </TouchableOpacity>
    </View>
  );
}

function DealRiskActionSection({
  dealId,
  router,
  isPending,
  onTrigger,
}: {
  dealId: string;
  router: ReturnType<typeof useRouter>;
  isPending: boolean;
  onTrigger: (dealId: string, onSuccess: (runId: string) => void) => void;
}) {
  const label = isPending ? 'Running...' : 'Analyze Deal Risk';
  return (
    <View style={styles.section}>
      <Button
        mode="outlined"
        testID="deal-risk-trigger-button"
        disabled={isPending}
        onPress={() => onTrigger(dealId, (runId) => router.push(wedgeHref(`/activity/${runId}`)))}
      >
        {label}
      </Button>
    </View>
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function SalesDealDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const rawId = Array.isArray(params.id) ? params.id[0] : params.id;
  const dealId = rawId.startsWith('deal-deal-') ? rawId.slice(5) : rawId;
  const dealRouteId = `deal-${dealId}`;
  const { data, isLoading, error } = useDeal(dealId);
  const triggerDealRiskAgent = useTriggerDealRiskAgent();
  const dealData = parseDealPayload(data);

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-deal-detail-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading deal...</Text>
      </View>
    );
  }

  if (error || !dealData) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-deal-detail-error">
        <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Deal not found'}</Text>
      </View>
    );
  }
  const handleDealRiskTrigger = (currentDealId: string, onSuccess: (runId: string) => void) => {
    triggerDealRiskAgent.mutate(
      { dealId: currentDealId, language: 'es' },
      {
        onSuccess: (result) => {
          if (result?.runId) onSuccess(result.runId);
        },
      },
    );
  };

  return (
    <>
      <Stack.Screen
        options={{
          title: 'Sales Deal',
          headerBackButtonDisplayMode: 'minimal',
          headerShadowVisible: false,
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.primary,
          headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' },
        }}
      />
      <ScrollView testID="sales-deal-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
        <View style={[styles.statusBanner, { backgroundColor: getStatusColor(dealData.status) }]}>
          <Text style={styles.statusText}>STATUS: {dealData.status.toUpperCase()}</Text>
        </View>
        <CRMDetailHeader title={dealData.title} subtitle={dealData.accountName} metadata={getMetadata(dealData)} testIDPrefix="sales-deal-detail" />
        <DealAmountSection amount={dealData.amount} colors={colors} />
        <DealStageSection stage={dealData.stage} colors={colors} />
        <DealAccountSection accountId={dealData.accountId} accountName={dealData.accountName} router={router} colors={colors} />
        <View style={styles.section}>
          <Button mode="contained" testID="sales-deal-brief-button" style={styles.actionButton}
            onPress={() => router.push(wedgeHrefObject(`/sales/${dealRouteId}/brief`, { entity_type: 'deal', entity_id: dealData.id }))}>
            Sales Brief
          </Button>
          <Button mode="outlined" testID="sales-deal-copilot-button"
            onPress={() => router.push(wedgeHrefObject(`/sales/${dealRouteId}/copilot`, { entity_type: 'deal', entity_id: dealData.id }))}>
            Open Copilot
          </Button>
        </View>
        <DealRiskActionSection
          dealId={dealData.id}
          router={router}
          isPending={triggerDealRiskAgent.isPending}
          onTrigger={handleDealRiskTrigger}
        />
        <AgentActivitySection entityType="deal" entityId={dealData.id} testIDPrefix="sales-deal-detail" />
        <EntitySignalsSection entityType="deal" entityId={dealData.id} testIDPrefix="sales-deal-detail" />
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
