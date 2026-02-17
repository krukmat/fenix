// Task 4.3 — Deals List Screen

import React, { useState, useMemo, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMListScreen } from '../../../src/components/crm';
import { useDeals } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

interface DealData {
  id: string;
  name?: string;
  value?: number;
  status: 'open' | 'won' | 'lost';
  stage?: string;
  accountName?: string;
  closeDate?: string;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function getStatusColor(status: string, colors: ThemeColors): string {
  if (status === 'won') return '#10B981';
  if (status === 'lost') return '#EF4444';
  return colors.primary;
}

// FIX-7: Export for tests
export function renderDealItem({ item }: { item: DealData }, colors: ThemeColors, router: ReturnType<typeof useRouter>) {
  return (
    <TouchableOpacity
      style={[styles.dealItem, { backgroundColor: colors.surface }]}
      onPress={() => router.push(`/deals/${item.id}`)}
      testID={`deal-item-${item.id}`}
    >
      <View style={styles.dealHeader}>
        <Text style={[styles.dealName, { color: colors.onSurface }]}>{item.name || 'Unnamed Deal'}</Text>
        <View style={[styles.statusChip, { backgroundColor: getStatusColor(item.status, colors) }]} testID={`deal-status-${item.status}`}>
          <Text style={[styles.statusChipText, { color: '#FFFFFF' }]}>{item.status}</Text>
        </View>
      </View>
      {item.accountName && (
        <Text style={[styles.dealAccount, { color: colors.onSurfaceVariant }]}>
          {item.accountName}
        </Text>
      )}
      {item.value !== undefined && (
        <Text style={[styles.dealValue, { color: colors.onSurfaceVariant }]}>
          ${item.value.toLocaleString()}
        </Text>
      )}
    </TouchableOpacity>
  );
}

export default function DealsListScreen() {
  const colors = useThemeColors();
  const router = useRouter();
  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, error, refetch, isRefetching } = useDeals();

  const [searchValue, setSearchValue] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const allDeals: DealData[] = useMemo(
    () => (data?.pages ?? []).flatMap((p) => (p.data as DealData[] | undefined) ?? []),
    [data]
  );

  const filteredDeals = useMemo(() => {
    let result = allDeals;
    if (statusFilter) {
      result = result.filter((deal: DealData) => deal.status === statusFilter);
    }
    if (!searchValue.trim()) return result;
    const searchLower = searchValue.toLowerCase();
    return result.filter(
      (deal: DealData) =>
        deal.name?.toLowerCase().includes(searchLower) ||
        deal.accountName?.toLowerCase().includes(searchLower)
    );
  }, [allDeals, searchValue, statusFilter]);

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const handleSearchChange = useCallback((value: string) => {
    setSearchValue(value);
  }, []);

  const handleStatusChange = useCallback((value: string) => {
    setStatusFilter(value);
  }, []);

  const renderItem = useCallback(
    ({ item }: { item: DealData }) => renderDealItem({ item }, colors, router),
    [colors, router]
  );

  return (
    <CRMListScreen
      data={filteredDeals}
      hasData={allDeals.length > 0}
      loading={isLoading}
      loadingMore={isFetchingNextPage}
      hasMore={hasNextPage ?? false}
      onEndReached={() => { if (hasNextPage && !isFetchingNextPage) fetchNextPage(); }}
      error={error ? error.message : null}
      onRefresh={handleRefresh}
      searchValue={searchValue}
      onSearchChange={handleSearchChange}
      renderItem={renderItem}
      emptyTitle="No deals found"
      emptySubtitle="Pull to refresh or add a new deal"
      testIDPrefix="deals"
      isRefreshing={isRefetching}
      onRetry={handleRefresh}
      statusFilter={statusFilter}
      onStatusFilterChange={handleStatusChange}
      availableStatuses={['open', 'won', 'lost']}
    />
  );
}

const styles = StyleSheet.create({
  dealItem: {
    padding: 16,
    marginHorizontal: 16,
    marginBottom: 12,
    borderRadius: 8,
    elevation: 2,
  },
  dealHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 4,
  },
  dealName: {
    fontSize: 16,
    fontWeight: '600',
    flex: 1,
  },
  statusChip: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 12,
  },
  statusChipText: {
    fontSize: 12,
    fontWeight: '500',
  },
  dealAccount: {
    fontSize: 14,
  },
  dealValue: {
    fontSize: 14,
    fontWeight: '500',
    marginTop: 4,
  },
});
