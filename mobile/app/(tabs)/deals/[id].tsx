// Task 4.3 — Deal Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader } from '../../../src/components/crm';
import { useDeal } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface DealDetailData {
  id: string;
  name?: string;
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
    { label: 'Value', value: deal.value ? `$${deal.value.toLocaleString()}` : 'Not specified' },
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
      <CRMDetailHeader title={deal.name || 'Unnamed Deal'} subtitle={deal.description} metadata={metadata} testIDPrefix="deal-detail" />
      {renderAccountSection(deal.accountId, deal.accountName, router, colors)}
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
  const deal: DealDetailData | undefined = data?.data;

  // FIX-1: Removed useMemo wrapping JSX
  const content = deal ? renderContent(deal, router, colors) : null;

  return (
    <>
      <Stack.Screen options={{ title: deal?.name || 'Deal' }} />
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
