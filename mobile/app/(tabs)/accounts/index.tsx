// Task 4.3 — Accounts List Screen

import React, { useState, useMemo, useCallback } from 'react';
import { Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMListScreen } from '../../../src/components/crm';
import { useAccounts } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

// Account type inferred from API response
interface AccountData {
  id: string;
  name?: string;
  industry?: string;
  phone?: string;
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
    ({ item }: { item: AccountData }) => (
      <TouchableOpacity
        style={[styles.accountItem, { backgroundColor: colors.surface }]}
        onPress={() => router.push(`/accounts/${item.id}`)}
        testID={`account-item-${item.id}`}
      >
        <Text style={[styles.accountName, { color: colors.onSurface }]}>{item.name || 'Unnamed Account'}</Text>
        <Text style={[styles.accountIndustry, { color: colors.onSurfaceVariant }]}>
          {item.industry || 'No industry'}
        </Text>
        {item.phone && (
          <Text style={[styles.accountPhone, { color: colors.onSurfaceVariant }]}>{item.phone}</Text>
        )}
      </TouchableOpacity>
    ),
    [colors, router]
  );

  return (
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
  );
}

const styles = StyleSheet.create({
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
});
