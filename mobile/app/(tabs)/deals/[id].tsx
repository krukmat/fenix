// Task 4.3 — Deal Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme, Button } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { AgentActivitySection } from '../../../src/components/agents/AgentActivitySection';
import { CRMDetailHeader } from '../../../src/components/crm';
import { useDeal } from '../../../src/hooks/useCRM';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import { SignalCountBadge } from '../../../src/components/signals/SignalCountBadge';
import type { ThemeColors } from '../../../src/theme/types';

const NOT_SPECIFIED = 'Not specified';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface DealDetailData {
  id: string;
  title?: string;
  name?: string;
  amount?: number;
  value?: number;
  status: 'open' | 'won' | 'lost';
  stage?: string;
  accountId?: string;
  accountName?: string;
  closeDate?: string;
  description?: string;
  pipeline?: string;
  activeSignalCount?: number;
}

function getStatusColor(status: string): string {
  if (status === 'won') return '#10B981';
  if (status === 'lost') return '#EF4444';
  return '#F59E0B';
}

function getMetadata(deal: DealDetailData) {
  return [
    { label: 'Value', value: (deal.amount ?? deal.value) ? `$${((deal.amount ?? deal.value) as number).toLocaleString()}` : NOT_SPECIFIED },
    { label: 'Stage', value: deal.stage || NOT_SPECIFIED },
    { label: 'Pipeline', value: deal.pipeline || 'Default' },
    { label: 'Close Date', value: deal.closeDate || NOT_SPECIFIED },
  ];
}

function renderAccountSection(accountId: string | undefined, accountName: string | undefined, router: ReturnType<typeof useRouter>, colors: ThemeColors) {
  if (!accountId) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.title, { color: colors.onSurface }]}>Account</Text>
      <TouchableOpacity
        style={[styles.card, { backgroundColor: colors.surface }]}
        onPress={() => router.push(`/accounts/${accountId}`)}
      >
        <Text style={{ color: colors.onSurface, fontWeight: '500' }}>{accountName || 'View Account'}</Text>
      </TouchableOpacity>
    </View>
  );
}

function renderContent(deal: DealDetailData, router: ReturnType<typeof useRouter>, colors: ThemeColors) {
  const metadata = getMetadata(deal);
  return (
    <>
      <View style={[styles.statusBanner, { backgroundColor: getStatusColor(deal.status) }]}>
        <Text style={styles.statusText}>{deal.status.toUpperCase()}</Text>
      </View>
      <CRMDetailHeader title={deal.title || deal.name || 'Unnamed Deal'} subtitle={deal.description} metadata={metadata} testIDPrefix="deal-detail" />
      <View style={styles.section}>
        <Text style={[styles.title, { color: colors.onSurface }]}>Signals</Text>
        <SignalCountBadge count={deal.activeSignalCount} testID="deal-detail-signal-badge" />
      </View>
      {renderAccountSection(deal.accountId, deal.accountName, router, colors)}
      <AgentActivitySection entityType="deal" entityId={deal.id} testIDPrefix="deal-agent-activity" />
      <EntitySignalsSection entityType="deal" entityId={deal.id} testIDPrefix="deal-signals" />
      <View style={styles.section}>
        <Button mode="contained" onPress={() => router.push(`/deals/edit/${deal.id}`)} testID="deal-edit-button">
          Edit Deal
        </Button>
      </View>
      <View style={styles.section}>
        <Button
          mode="contained"
          onPress={() => router.push({ pathname: '/copilot', params: { entity_type: 'deal', entity_id: deal.id } })}
          testID="deal-copilot-open-button"
        >
          Open Copilot
        </Button>
      </View>
    </>
  );
}

function s(o: Record<string, unknown> | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

function n(o: Record<string, unknown> | null | undefined, key: string): number | undefined {
  return o?.[key] as number | undefined;
}

function parseDealCore(d: Record<string, unknown>): Omit<DealDetailData, 'accountName' | 'activeSignalCount'> {
  return {
    id: String(d.id ?? ''),
    title: s(d, 'title'),
    name: s(d, 'name') ?? s(d, 'title'),
    amount: n(d, 'amount'),
    value: n(d, 'value') ?? n(d, 'amount'),
    status: (s(d, 'status') as 'open' | 'won' | 'lost' | undefined) ?? 'open',
    stage: s(d, 'stage'),
    accountId: s(d, 'accountId') ?? s(d, 'account_id'),
    closeDate: s(d, 'closeDate') ?? s(d, 'expectedClose'),
    description: s(d, 'description'),
    pipeline: s(d, 'pipeline'),
  };
}

function parseDealPayload(data: unknown): DealDetailData | undefined {
  const payload = (data ?? null) as Record<string, unknown> | null;
  const d = (payload?.deal as Record<string, unknown> | undefined) ?? payload ?? undefined;
  if (!d) return undefined;
  const acct = payload?.account as Record<string, unknown> | undefined;
  const signalCount = payload?.active_signal_count;
  return {
    ...parseDealCore(d),
    accountName: s(acct, 'name'),
    activeSignalCount: typeof signalCount === 'number' ? signalCount : 0,
  };
}

function renderScreenBody(
  isLoading: boolean,
  error: Error | null,
  content: React.ReactNode,
  colors: ThemeColors,
  loadingLabel: string,
  emptyLabel: string,
): React.ReactNode {
  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]}>
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>{loadingLabel}</Text>
      </View>
    );
  }
  if (error || !content) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]}>
        <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || emptyLabel}</Text>
      </View>
    );
  }
  return content;
}

export default function DealDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  // FIX-4: Runtime guard for id param
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useDeal(id);
  const deal = parseDealPayload(data);
  const content = deal ? renderContent(deal, router, colors) : null;

  return (
    <>
      <Stack.Screen options={{ title: deal?.title || deal?.name || 'Deal' }} />
      <ScrollView testID="deal-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
        {renderScreenBody(isLoading, error ?? null, content, colors, 'Loading deal...', 'Deal not found')}
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  statusBanner: { padding: 8, alignItems: 'center' },
  statusText: { color: '#FFF', fontWeight: '600', fontSize: 14 },
  section: { padding: 16 },
  title: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 16, borderRadius: 8 },
});
