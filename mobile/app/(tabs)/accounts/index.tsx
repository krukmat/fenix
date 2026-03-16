// Task 4.3 — Accounts List Screen
// Task 4.8 — GAP 1: Added FAB for account creation

import React, { useState, useMemo, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme, FAB } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMListScreen } from '../../../src/components/crm';
import { useAccounts } from '../../../src/hooks/useCRM';
import { SignalCountBadge } from '../../../src/components/signals/SignalCountBadge';
import type { ThemeColors } from '../../../src/theme/types';

// Account type inferred from API response
interface AccountData {
  id: string;
  name?: string;
  industry?: string;
  phone?: string;
  active_signal_count?: number;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

export default function AccountsListScreen() {
  const colors = useThemeColors();
  const router = useRouter();
  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, error, refetch, isRefetching } = useAccounts();

  const [searchValue, setSearchValue] = useState('');

  const allAccounts: AccountData[] = useMemo(
    () => (data?.pages ?? []).flatMap((p) => (p.data as AccountData[] | undefined) ?? []),
    [data]
  );

  const filteredAccounts = useMemo(() => {
    if (!searchValue.trim()) return allAccounts;
    const searchLower = searchValue.toLowerCase();
    return allAccounts.filter(
      (account: AccountData) =>
        account.name?.toLowerCase().includes(searchLower) ||
        account.industry?.toLowerCase().includes(searchLower)
    );
  }, [allAccounts, searchValue]);

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const handleSearchChange = useCallback((value: string) => {
    setSearchValue(value);
  }, []);

  const renderItem = useCallback(
    ({ item, index }: { item: AccountData; index: number }) => (
      <TouchableOpacity
        style={[styles.accountItem, { backgroundColor: colors.surface }]}
        onPress={() => router.push(`/accounts/${item.id}`)}
        testID={`accounts-list-item-${index}`}
      >
        <Text style={[styles.accountName, { color: colors.onSurface }]}>{item.name || 'Unnamed Account'}</Text>
        <Text style={[styles.accountIndustry, { color: colors.onSurfaceVariant }]}>
          {item.industry || 'No industry'}
        </Text>
        <View style={styles.badgeRow}>
          <SignalCountBadge count={item.active_signal_count} testID={`account-signals-badge-${item.id}`} />
        </View>
        {item.phone && (
          <Text style={[styles.accountPhone, { color: colors.onSurfaceVariant }]}>{item.phone}</Text>
        )}
      </TouchableOpacity>
    ),
    [colors, router]
  );

  return (
    <View style={{ flex: 1 }}>
      <CRMListScreen
        data={filteredAccounts}
        loading={isLoading}
        error={error ? error.message : null}
        onRefresh={handleRefresh}
        searchValue={searchValue}
        onSearchChange={handleSearchChange}
        renderItem={renderItem}
        hasData={allAccounts.length > 0}
        loadingMore={isFetchingNextPage}
        hasMore={hasNextPage ?? false}
        onEndReached={() => { if (hasNextPage && !isFetchingNextPage) fetchNextPage(); }}
        emptyTitle="No accounts found"
        emptySubtitle="Pull to refresh or add a new account"
        testIDPrefix="accounts"
        isRefreshing={isRefetching}
        onRetry={handleRefresh}
      />
      <FAB
        testID="create-account-fab"
        icon="plus"
        style={[styles.fab, { backgroundColor: colors.primary }]}
        onPress={() => router.push('/accounts/new')}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  fab: {
    position: 'absolute',
    right: 16,
    bottom: 24,
  },
  accountItem: {
    padding: 16,
    marginHorizontal: 16,
    marginBottom: 12,
    borderRadius: 8,
    elevation: 2,
  },
  accountName: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 4,
  },
  accountIndustry: {
    fontSize: 14,
  },
  accountPhone: {
    fontSize: 12,
    marginTop: 4,
  },
  badgeRow: {
    alignItems: 'flex-start',
    marginTop: 8,
  },
});
