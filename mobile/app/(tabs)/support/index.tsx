// Support wedge — case list with inbox badge connection (W3-T1, W3-T5)
// No create FAB — creation removed from wedge. Navigation: /support/[id]
import React, { useState, useMemo, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMListScreen } from '../../../src/components/crm';
import { useCases } from '../../../src/hooks/useCRM';
import { useInbox } from '../../../src/hooks/useWedge';
import { SignalCountBadge } from '../../../src/components/signals/SignalCountBadge';
import type { ThemeColors } from '../../../src/theme/types';

interface CaseData {
  id: string;
  subject?: string;
  status: string;
  priority: 'low' | 'medium' | 'high';
  accountName?: string;
  active_signal_count?: number;
}

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function getPriorityColor(priority: string): string {
  if (priority === 'high') return '#EF4444';
  if (priority === 'medium') return '#F59E0B';
  return '#10B981';
}

function InboxBadge({ router, colors }: { router: ReturnType<typeof useRouter>; colors: ThemeColors }) {
  const { data } = useInbox();
  const count = (data?.approvals?.length ?? 0) + (data?.handoffs?.length ?? 0);
  if (count === 0) return null;
  return (
    <TouchableOpacity
      testID="support-inbox-badge"
      style={[styles.inboxBadge, { backgroundColor: colors.primary }]}
      onPress={() => router.push('/inbox')}
    >
      <Text style={styles.inboxBadgeText}>{count > 99 ? '99+' : count} pending</Text>
    </TouchableOpacity>
  );
}

export function renderSupportCaseItem(
  { item, index }: { item: CaseData; index: number },
  colors: ThemeColors,
  router: ReturnType<typeof useRouter>
) {
  return (
    <TouchableOpacity
      style={[styles.caseItem, { backgroundColor: colors.surface }]}
      onPress={() => router.push(`/support/${item.id}`)}
      testID={`support-cases-list-item-${index}`}
    >
      <View style={styles.caseHeader}>
        <Text style={[styles.caseSubject, { color: colors.onSurface }]}>{item.subject || 'No Subject'}</Text>
        <View style={[styles.priorityBadge, { backgroundColor: getPriorityColor(item.priority) }]}>
          <Text style={styles.priorityText}>{item.priority}</Text>
        </View>
      </View>
      {item.accountName && (
        <Text style={[styles.caseAccount, { color: colors.onSurfaceVariant }]}>{item.accountName}</Text>
      )}
      <Text style={[styles.caseStatus, { color: colors.onSurfaceVariant }]}>Status: {item.status}</Text>
      <View style={styles.badgeRow}>
        <SignalCountBadge count={item.active_signal_count} testID={`case-signals-badge-${item.id}`} />
      </View>
    </TouchableOpacity>
  );
}

export default function SupportScreen() {
  const colors = useColors();
  const router = useRouter();
  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, error, refetch, isRefetching } = useCases();

  const [searchValue, setSearchValue] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const allCases: CaseData[] = useMemo(
    () => (data?.pages ?? []).flatMap((p) => (p.data as CaseData[] | undefined) ?? []),
    [data]
  );

  const filteredCases = useMemo(() => {
    let result = allCases;
    if (statusFilter) result = result.filter((c) => c.status === statusFilter);
    if (!searchValue.trim()) return result;
    const q = searchValue.toLowerCase();
    return result.filter(
      (c) => c.subject?.toLowerCase().includes(q) || c.accountName?.toLowerCase().includes(q)
    );
  }, [allCases, searchValue, statusFilter]);

  const handleRefresh = useCallback(() => { refetch(); }, [refetch]);
  const handleSearchChange = useCallback((v: string) => { setSearchValue(v); }, []);
  const handleStatusChange = useCallback((v: string) => { setStatusFilter(v); }, []);

  const renderItem = useCallback(
    ({ item, index }: { item: CaseData; index: number }) =>
      renderSupportCaseItem({ item, index }, colors, router),
    [colors, router]
  );

  return (
    <View style={styles.container}>
      <InboxBadge router={router} colors={colors} />
      <CRMListScreen
        data={filteredCases}
        loading={isLoading}
        error={error ? error.message : null}
        onRefresh={handleRefresh}
        searchValue={searchValue}
        onSearchChange={handleSearchChange}
        renderItem={renderItem}
        hasData={allCases.length > 0}
        loadingMore={isFetchingNextPage}
        hasMore={hasNextPage ?? false}
        onEndReached={() => { if (hasNextPage && !isFetchingNextPage) fetchNextPage(); }}
        emptyTitle="No cases found"
        emptySubtitle="No open cases right now"
        testIDPrefix="support-cases"
        isRefreshing={isRefetching}
        onRetry={handleRefresh}
        statusFilter={statusFilter}
        onStatusFilterChange={handleStatusChange}
        availableStatuses={['open', 'closed', 'pending']}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  caseItem: { padding: 16, marginHorizontal: 16, marginBottom: 12, borderRadius: 8, elevation: 2 },
  caseHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 },
  caseSubject: { fontSize: 16, fontWeight: '600', flex: 1 },
  priorityBadge: { paddingHorizontal: 8, paddingVertical: 4, borderRadius: 12 },
  priorityText: { color: '#FFFFFF', fontSize: 12, fontWeight: '500' },
  caseAccount: { fontSize: 14 },
  caseStatus: { fontSize: 12, marginTop: 4 },
  badgeRow: { alignItems: 'flex-start', marginTop: 8 },
  inboxBadge: { margin: 12, padding: 10, borderRadius: 8, alignItems: 'center' },
  inboxBadgeText: { color: '#FFF', fontWeight: '700', fontSize: 13 },
});
