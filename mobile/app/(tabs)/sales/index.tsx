// Sales wedge — segmented accounts + deals browsing (W4-T1)
// No create/edit CTAs — wedge is read+brief+copilot only
import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  FlatList,
  ActivityIndicator,
} from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useAccounts, useDeals } from '../../../src/hooks/useCRM';
import { wedgeHref } from '../../../src/utils/navigation';
import { SignalCountBadge } from '../../../src/components/signals/SignalCountBadge';
import { LeadListTab } from '../../../src/components/sales/LeadListTab';
import { ContactsListContent } from '../../../src/components/contacts/ContactsListContent';
import type { ThemeColors } from '../../../src/theme/types';
import { brandColors } from '../../../src/theme/colors';
import { elevation, radius, spacing } from '../../../src/theme/spacing';
import { getAgentStatusColor } from '../../../src/theme/semantic';
import { typography } from '../../../src/theme/typography';

// ─── Types ────────────────────────────────────────────────────────────────────

type Tab = 'accounts' | 'deals' | 'leads' | 'contacts';

interface AccountItem {
  id: string;
  name?: string;
  industry?: string;
  active_signal_count?: number;
}

interface DealItem {
  id: string;
  title?: string;
  name?: string;
  status: 'open' | 'won' | 'lost';
  amount?: number;
  accountName?: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function TabBar({ active, onSelect, colors }: { active: Tab; onSelect: (t: Tab) => void; colors: ThemeColors }) {
  return (
    <View style={[styles.tabBar, { backgroundColor: colors.surface }]}>
      <TouchableOpacity
        testID="sales-tab-accounts"
        style={[styles.tab, active === 'accounts' && { borderBottomColor: colors.primary, borderBottomWidth: 2 }]}
        onPress={() => onSelect('accounts')}
      >
        <Text style={[styles.tabText, { color: active === 'accounts' ? colors.primary : colors.onSurfaceVariant }]}>
          Accounts
        </Text>
      </TouchableOpacity>
      <TouchableOpacity
        testID="sales-tab-deals"
        style={[styles.tab, active === 'deals' && { borderBottomColor: colors.primary, borderBottomWidth: 2 }]}
        onPress={() => onSelect('deals')}
      >
        <Text style={[styles.tabText, { color: active === 'deals' ? colors.primary : colors.onSurfaceVariant }]}>
          Deals
        </Text>
      </TouchableOpacity>
      <TouchableOpacity
        testID="sales-tab-leads"
        style={[styles.tab, active === 'leads' && { borderBottomColor: colors.primary, borderBottomWidth: 2 }]}
        onPress={() => onSelect('leads')}
      >
        <Text style={[styles.tabText, { color: active === 'leads' ? colors.primary : colors.onSurfaceVariant }]}>
          Leads
        </Text>
      </TouchableOpacity>
      <TouchableOpacity
        testID="sales-tab-contacts"
        style={[styles.tab, active === 'contacts' && { borderBottomColor: colors.primary, borderBottomWidth: 2 }]}
        onPress={() => onSelect('contacts')}
      >
        <Text style={[styles.tabText, { color: active === 'contacts' ? colors.primary : colors.onSurfaceVariant }]}>
          Contacts
        </Text>
      </TouchableOpacity>
    </View>
  );
}

function AccountRow({
  item, index, colors, onPress,
}: { item: AccountItem; index: number; colors: ThemeColors; onPress: () => void }) {
  return (
    <TouchableOpacity
      testID={`sales-account-item-${index}`}
      style={[styles.row, { backgroundColor: colors.surface }]}
      onPress={onPress}
    >
      <Text style={[styles.rowTitle, { color: colors.onSurface }]}>{item.name || 'Unnamed Account'}</Text>
      {item.industry ? (
        <Text style={[styles.rowSub, { color: colors.onSurfaceVariant }]}>{item.industry}</Text>
      ) : null}
      <SignalCountBadge count={item.active_signal_count} testID={`sales-account-signals-${item.id}`} />
    </TouchableOpacity>
  );
}

function DealRow({
  item, index, colors, onPress,
}: { item: DealItem; index: number; colors: ThemeColors; onPress: () => void }) {
  return (
    <TouchableOpacity
      testID={`sales-deal-item-${index}`}
      style={[styles.row, { backgroundColor: colors.surface }]}
      onPress={onPress}
    >
      <View style={styles.dealHeader}>
        <Text style={[styles.rowTitle, { color: colors.onSurface, flex: 1 }]}>
          {item.title || item.name || 'Unnamed Deal'}
        </Text>
        <View style={[styles.statusChip, { backgroundColor: getAgentStatusColor(item.status) }]}>
          <Text style={styles.statusChipText}>{item.status}</Text>
        </View>
      </View>
      {item.accountName ? (
        <Text style={[styles.rowSub, { color: colors.onSurfaceVariant }]}>{item.accountName}</Text>
      ) : null}
      {item.amount !== undefined ? (
        <Text style={[styles.rowSub, typography.monoLG, { color: colors.onSurfaceVariant }]}>
          ${item.amount.toLocaleString()}
        </Text>
      ) : null}
    </TouchableOpacity>
  );
}

// ─── Accounts tab ─────────────────────────────────────────────────────────────

function AccountsTab({ colors, router }: { colors: ThemeColors; router: ReturnType<typeof useRouter> }) {
  const { data, isLoading, fetchNextPage, hasNextPage } = useAccounts();

  if (isLoading) {
    return (
      <View style={styles.center} testID="sales-accounts-loading">
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  const items: AccountItem[] = (data?.pages ?? []).flatMap(
    (p: { data?: unknown[] }) => (p.data ?? []) as AccountItem[],
  );

  if (items.length === 0) {
    return (
      <View style={styles.center} testID="sales-accounts-empty">
        <Text style={{ color: colors.onSurfaceVariant }}>No accounts yet</Text>
      </View>
    );
  }

  return (
    <FlatList
      data={items}
      keyExtractor={(item) => item.id}
      renderItem={({ item, index }) => (
        <AccountRow
          item={item}
          index={index}
          colors={colors}
          onPress={() => router.push(wedgeHref(`/sales/${item.id}`))}
        />
      )}
      contentContainerStyle={styles.listContent}
      onEndReached={() => { if (hasNextPage) fetchNextPage(); }}
      onEndReachedThreshold={0.3}
    />
  );
}

// ─── Deals tab ────────────────────────────────────────────────────────────────

function DealsTab({ colors, router }: { colors: ThemeColors; router: ReturnType<typeof useRouter> }) {
  const { data, isLoading, fetchNextPage, hasNextPage } = useDeals();

  if (isLoading) {
    return (
      <View style={styles.center} testID="sales-deals-loading">
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  const items: DealItem[] = (data?.pages ?? []).flatMap(
    (p: { data?: unknown[] }) => (p.data ?? []) as DealItem[],
  );

  if (items.length === 0) {
    return (
      <View style={styles.center} testID="sales-deals-empty">
        <Text style={{ color: colors.onSurfaceVariant }}>No deals yet</Text>
      </View>
    );
  }

  return (
    <FlatList
      data={items}
      keyExtractor={(item) => item.id}
      renderItem={({ item, index }) => (
        <DealRow
          item={item}
          index={index}
          colors={colors}
          onPress={() => router.push(wedgeHref(`/sales/deals/${item.id}`))}
        />
      )}
      contentContainerStyle={styles.listContent}
      onEndReached={() => { if (hasNextPage) fetchNextPage(); }}
      onEndReachedThreshold={0.3}
    />
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function SalesScreen() {
  const colors = useColors();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<Tab>('accounts');

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]} testID="sales-screen">
      <TabBar active={activeTab} onSelect={setActiveTab} colors={colors} />
      {activeTab === 'accounts' ? <AccountsTab colors={colors} router={router} /> : null}
      {activeTab === 'deals' ? <DealsTab colors={colors} router={router} /> : null}
      {activeTab === 'leads' ? <LeadListTab colors={colors} router={router} /> : null}
      {activeTab === 'contacts' ? <ContactsListContent /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  tabBar: { flexDirection: 'row', ...elevation.card },
  tab: { flex: 1, alignItems: 'center', paddingVertical: spacing.md },
  tabText: { fontSize: 14, fontWeight: '600' },
  listContent: { padding: spacing.base },
  row: { padding: spacing.base, borderRadius: radius.md, marginBottom: spacing.md, ...elevation.card },
  rowTitle: { fontSize: 16, fontWeight: '600', marginBottom: 2 },
  rowSub: { fontSize: 14, marginTop: 2 },
  dealHeader: { flexDirection: 'row', alignItems: 'center', marginBottom: 2 },
  statusChip: { paddingHorizontal: spacing.sm, paddingVertical: spacing.xs, borderRadius: radius.full },
  statusChipText: { color: brandColors.onError, ...typography.labelMD },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: spacing.xl },
});
