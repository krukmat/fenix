// Task 4.3 — Deal Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme, Button } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader } from '../../../src/components/crm';
import { useDeal } from '../../../src/hooks/useCRM';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import type { ThemeColors } from '../../../src/theme/types';

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
}

function getStatusColor(status: string): string {
  if (status === 'won') return '#10B981';
  if (status === 'lost') return '#EF4444';
  return '#F59E0B';
}

function getMetadata(deal: DealDetailData) {
  return [
    { label: 'Value', value: (deal.amount ?? deal.value) ? `$${(deal.amount ?? deal.value)!.toLocaleString()}` : 'Not specified' },
    { label: 'Stage', value: deal.stage || 'Not specified' },
    { label: 'Pipeline', value: deal.pipeline || 'Default' },
    { label: 'Close Date', value: deal.closeDate || 'Not specified' },
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
      {renderAccountSection(deal.accountId, deal.accountName, router, colors)}
      <EntitySignalsSection entityType="deal" entityId={deal.id} testIDPrefix="deal-signals" />
      <View style={styles.section}>
        <Button mode="contained" onPress={() => router.push(`/deals/edit/${deal.id}`)} testID="deal-edit-button">
          Edit Deal
        </Button>
      </View>
    </>
  );
}

// eslint-disable-next-line complexity
export default function DealDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  // FIX-4: Runtime guard for id param
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useDeal(id);
  const payload = (data?.data ?? data ?? null) as Record<string, unknown> | null;
  const dealObj = (payload?.deal as Record<string, unknown> | undefined) ?? payload ?? undefined;
  const accountObj = payload?.account as Record<string, unknown> | undefined;
  const deal: DealDetailData | undefined = dealObj
    ? {
        id: String(dealObj.id ?? ''),
        title: dealObj.title as string | undefined,
        name: (dealObj.name as string | undefined) ?? (dealObj.title as string | undefined),
        amount: dealObj.amount as number | undefined,
        value: (dealObj.value as number | undefined) ?? (dealObj.amount as number | undefined),
        status: ((dealObj.status as 'open' | 'won' | 'lost' | undefined) ?? 'open'),
        stage: dealObj.stage as string | undefined,
        accountId: (dealObj.accountId as string | undefined) ?? (dealObj.account_id as string | undefined),
        accountName: accountObj?.name as string | undefined,
        closeDate: (dealObj.closeDate as string | undefined) ?? (dealObj.expectedClose as string | undefined),
        description: dealObj.description as string | undefined,
        pipeline: dealObj.pipeline as string | undefined,
      }
    : undefined;

  // FIX-1: Removed useMemo wrapping JSX
  const content = deal ? renderContent(deal, router, colors) : null;

  return (
    <>
      <Stack.Screen options={{ title: deal?.title || deal?.name || 'Deal' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]}>
        {isLoading ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading deal...</Text>
          </View>
        ) : error || !deal ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Deal not found'}</Text>
          </View>
        ) : content}
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
