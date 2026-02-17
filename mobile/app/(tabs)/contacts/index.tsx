// Task 4.3 — Contacts List Screen

import React, { useState, useMemo, useCallback } from 'react';
import { Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMListScreen } from '../../../src/components/crm';
import { useContacts } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

interface ContactData {
  id: string;
  name?: string;
  email?: string;
  phone?: string;
  accountId?: string;
  accountName?: string;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

export default function ContactsListScreen() {
  const colors = useThemeColors();
  const router = useRouter();
  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, error, refetch, isRefetching } = useContacts();

  const [searchValue, setSearchValue] = useState('');

  const allContacts: ContactData[] = useMemo(
    () => (data?.pages ?? []).flatMap((p) => (p.data as ContactData[] | undefined) ?? []),
    [data]
  );

  const filteredContacts = useMemo(() => {
    if (!searchValue.trim()) return allContacts;
    const searchLower = searchValue.toLowerCase();
    return allContacts.filter(
      (contact: ContactData) =>
        contact.name?.toLowerCase().includes(searchLower) ||
        contact.email?.toLowerCase().includes(searchLower)
    );
  }, [allContacts, searchValue]);

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const handleSearchChange = useCallback((value: string) => {
    setSearchValue(value);
  }, []);

  const renderItem = useCallback(
    ({ item }: { item: ContactData }) => (
      <TouchableOpacity
        style={[styles.contactItem, { backgroundColor: colors.surface }]}
        onPress={() => router.push(`/contacts/${item.id}`)}
        testID={`contact-item-${item.id}`}
      >
        <Text style={[styles.contactName, { color: colors.onSurface }]}>{item.name || 'Unnamed Contact'}</Text>
        {item.email && (
          <Text style={[styles.contactEmail, { color: colors.onSurfaceVariant }]}>{item.email}</Text>
        )}
        {item.accountName && (
          <Text style={[styles.contactAccount, { color: colors.onSurfaceVariant }]}>
            Account: {item.accountName}
          </Text>
        )}
      </TouchableOpacity>
    ),
    [colors, router]
  );

  return (
    <CRMListScreen
      data={filteredContacts}
      loading={isLoading}
      error={error ? error.message : null}
      onRefresh={handleRefresh}
      searchValue={searchValue}
      onSearchChange={handleSearchChange}
      renderItem={renderItem}
      hasData={allContacts.length > 0}
      loadingMore={isFetchingNextPage}
      hasMore={hasNextPage ?? false}
      onEndReached={() => { if (hasNextPage && !isFetchingNextPage) fetchNextPage(); }}
      emptyTitle="No contacts found"
      emptySubtitle="Pull to refresh or add a new contact"
      testIDPrefix="contacts"
      isRefreshing={isRefetching}
      onRetry={handleRefresh}
    />
  );
}

const styles = StyleSheet.create({
  contactItem: {
    padding: 16,
    marginHorizontal: 16,
    marginBottom: 12,
    borderRadius: 8,
    elevation: 2,
  },
  contactName: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 4,
  },
  contactEmail: {
    fontSize: 14,
  },
  contactAccount: {
    fontSize: 12,
    marginTop: 4,
  },
});
