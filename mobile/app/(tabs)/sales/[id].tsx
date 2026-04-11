// Sales wedge — account detail (W4-T2)
// Read-only: no edit button. Actions: Sales Brief + Copilot.
import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator } from 'react-native';
import { useTheme, Button } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader } from '../../../src/components/crm';
import { AgentActivitySection } from '../../../src/components/agents/AgentActivitySection';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import { SignalCountBadge } from '../../../src/components/signals/SignalCountBadge';
import { useAccount } from '../../../src/hooks/useCRM';
import { wedgeHrefObject } from '../../../src/utils/navigation';
import type { ThemeColors } from '../../../src/theme/types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface AccountDetailData {
  id: string;
  name: string;
  industry?: string;
  phone?: string;
  owner?: string;
  activeSignalCount?: number;
}

type R = Record<string, unknown>;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function s(o: R | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

function parseAccountPayload(data: unknown): AccountDetailData | undefined {
  const payload = (data ?? null) as R | null;
  if (!payload) return undefined;
  const acct = (payload.account as R | undefined) ?? payload;
  if (!acct?.id) return undefined;
  const signalCount = payload.active_signal_count;
  return {
    id: String(acct.id),
    name: s(acct, 'name') ?? 'Unknown Account',
    industry: s(acct, 'industry'),
    phone: s(acct, 'phone'),
    owner: s(acct, 'owner'),
    activeSignalCount: typeof signalCount === 'number' ? signalCount : 0,
  };
}

function getMetadata(a: AccountDetailData) {
  return [
    { label: 'Industry', value: a.industry || 'N/A' },
    { label: 'Owner', value: a.owner || 'Unassigned' },
    { label: 'Phone', value: a.phone || 'Not set' },
  ];
}

// ─── Content ─────────────────────────────────────────────────────────────────

function AccountDetailContent({ account, colors, router }: {
  account: AccountDetailData;
  colors: ThemeColors;
  router: ReturnType<typeof useRouter>;
}) {
  return (
    <ScrollView testID="sales-account-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <CRMDetailHeader title={account.name} subtitle={account.industry} metadata={getMetadata(account)} testIDPrefix="sales-account-detail" />
      <View style={styles.section}>
        <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Signals</Text>
        <SignalCountBadge count={account.activeSignalCount} testID="sales-account-signal-badge" />
      </View>
      <View style={styles.section}>
        <Button mode="contained" testID="sales-brief-button" style={styles.actionButton}
          onPress={() => router.push(wedgeHrefObject(`/sales/${account.id}/brief`, { entity_type: 'account', entity_id: account.id }))}>
          Sales Brief
        </Button>
        <Button mode="outlined" testID="sales-copilot-button"
          onPress={() => router.push(wedgeHrefObject(`/sales/${account.id}/copilot`, { entity_type: 'account', entity_id: account.id }))}>
          Open Copilot
        </Button>
      </View>
      <AgentActivitySection entityType="account" entityId={account.id} testIDPrefix="sales-account-detail" />
      <EntitySignalsSection entityType="account" entityId={account.id} testIDPrefix="sales-account-detail" />
    </ScrollView>
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function SalesAccountDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useAccount(id);
  const accountData = parseAccountPayload(data);

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-account-detail-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading account...</Text>
      </View>
    );
  }
  if (error || !accountData) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-account-detail-error">
        <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Account not found'}</Text>
      </View>
    );
  }
  return (
    <>
      <Stack.Screen
        options={{
          title: 'Sales Account',
          headerBackButtonDisplayMode: 'minimal',
          headerShadowVisible: false,
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.primary,
          headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' },
        }}
      />
      <AccountDetailContent account={accountData} colors={colors} router={router} />
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  section: { padding: 16 },
  sectionTitle: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  actionButton: { marginBottom: 12 },
});
