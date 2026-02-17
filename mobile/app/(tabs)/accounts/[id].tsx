// Task 4.3 — Account Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack } from 'expo-router';
import { CRMDetailHeader, EntityTimeline } from '../../../src/components/crm';
import { useAccount } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface ContactItem { id: string; name: string; email?: string; }
interface DealItem { id: string; name: string; value?: number; status: string; }
interface TimelineItem { id: string; type: 'note' | 'activity' | 'status_change' | 'created' | 'updated'; title: string; description?: string; timestamp: string; userName?: string; }
interface AccountData { id: string; name?: string; industry?: string; phone?: string; email?: string; website?: string; description?: string; contacts?: ContactItem[]; deals?: DealItem[]; timeline?: TimelineItem[]; }

function getMetadata(account: AccountData) {
  return [
    { label: 'Industry', value: account.industry || 'Not specified' },
    { label: 'Phone', value: account.phone || 'Not available' },
    { label: 'Email', value: account.email || 'Not available' },
    { label: 'Website', value: account.website || 'Not available' },
  ];
}

function renderContactsSection(contacts: ContactItem[], colors: ThemeColors) {
  return (
    <View style={styles.section}>
      <Text style={[styles.title, { color: colors.onSurface }]}>Related Contacts</Text>
      {contacts.map(c => (
        <View key={c.id} style={[styles.card, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurface, fontWeight: '500' }}>{c.name}</Text>
          {c.email && <Text style={{ color: colors.onSurfaceVariant, fontSize: 12, marginTop: 4 }}>{c.email}</Text>}
        </View>
      ))}
    </View>
  );
}

function renderDealsSection(deals: DealItem[], colors: ThemeColors) {
  return (
    <View style={styles.section}>
      <Text style={[styles.title, { color: colors.onSurface }]}>Related Deals</Text>
      {deals.map(d => {
        const statusColor = d.status === 'won' ? '#10B981' : d.status === 'lost' ? '#EF4444' : colors.primary;
        return (
          <View key={d.id} style={[styles.card, { backgroundColor: colors.surface }]}>
            <View style={styles.row}>
              <Text style={{ color: colors.onSurface, fontWeight: '500', flex: 1 }}>{d.name}</Text>
              <View style={[styles.badge, { backgroundColor: statusColor }]}>
                <Text style={styles.badgeText}>{d.status}</Text>
              </View>
            </View>
            {d.value !== undefined && <Text style={{ color: colors.onSurfaceVariant, fontSize: 12, marginTop: 4 }}>${d.value.toLocaleString()}</Text>}
          </View>
        );
      })}
    </View>
  );
}

function renderTimelineSection(timeline: TimelineItem[], colors: ThemeColors) {
  return (
    <View style={styles.section}>
      <Text style={[styles.title, { color: colors.onSurface }]}>Activity</Text>
      <EntityTimeline events={timeline} testIDPrefix="account-timeline" emptyMessage="No activity yet" />
    </View>
  );
}

function renderContent(account: AccountData, colors: ThemeColors) {
  const metadata = getMetadata(account);
  return (
    <>
      <CRMDetailHeader title={account.name || 'Unnamed Account'} subtitle={account.description} metadata={metadata} testIDPrefix="account-detail" />
      {account.contacts && account.contacts.length > 0 && renderContactsSection(account.contacts, colors)}
      {account.deals && account.deals.length > 0 && renderDealsSection(account.deals, colors)}
      {renderTimelineSection(account.timeline || [], colors)}
    </>
  );
}

// eslint-disable-next-line complexity
export default function AccountDetailScreen() {
  const colors = useColors();
  // FIX-4: Runtime guard for id param
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useAccount(id);
  const account: AccountData | undefined = data?.data;

  // FIX-1: Removed useMemo wrapping JSX
  const content = account ? renderContent(account, colors) : null;
  const title = account?.name || 'Account';

  return (
    <>
      <Stack.Screen options={{ title }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]}>
        {isLoading ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading account...</Text>
          </View>
        ) : error || !account ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Account not found'}</Text>
          </View>
        ) : content}
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  section: { padding: 16 },
  title: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 12, borderRadius: 8, marginBottom: 8 },
  row: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  badge: { paddingHorizontal: 8, paddingVertical: 4, borderRadius: 12 },
  badgeText: { color: '#FFF', fontSize: 12, fontWeight: '500' },
});
