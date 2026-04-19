// crm-dentro-governance: Shared Contacts list for canonical /contacts and Sales tab.
import React, { useCallback, useMemo, useState } from 'react';
import { Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMListScreen } from '../crm';
import { useContacts } from '../../hooks/useCRM';
import { wedgeHref } from '../../utils/navigation';
import type { ThemeColors } from '../../theme/types';

interface ContactData {
  id: string;
  name?: string;
  email?: string;
  phone?: string;
  accountName?: string;
  title?: string;
}

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function renderContactItem(
  { item, index }: { item: ContactData; index: number },
  colors: ThemeColors,
  router: ReturnType<typeof useRouter>,
) {
  return (
    <TouchableOpacity
      style={[styles.contactItem, { backgroundColor: colors.surface }]}
      onPress={() => router.push(wedgeHref(`/contacts/${item.id}`))}
      testID={`contacts-list-item-${index}`}
    >
      <Text style={[styles.contactName, { color: colors.onSurface }]}>
        {item.name || 'Unknown Contact'}
      </Text>
      {item.title ? (
        <Text style={[styles.contactMeta, { color: colors.onSurfaceVariant }]}>{item.title}</Text>
      ) : null}
      {item.accountName ? (
        <Text style={[styles.contactMeta, { color: colors.onSurfaceVariant }]}>{item.accountName}</Text>
      ) : null}
      {item.email ? (
        <Text style={[styles.contactMeta, { color: colors.onSurfaceVariant }]}>{item.email}</Text>
      ) : null}
    </TouchableOpacity>
  );
}

export function ContactsListContent() {
  const colors = useColors();
  const router = useRouter();
  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, error, refetch, isRefetching } =
    useContacts();

  const [searchValue, setSearchValue] = useState('');

  const allContacts: ContactData[] = useMemo(
    () => (data?.pages ?? []).flatMap((p) => (p.data as ContactData[] | undefined) ?? []),
    [data],
  );

  const filteredContacts = useMemo(() => {
    if (!searchValue.trim()) return allContacts;
    const q = searchValue.toLowerCase();
    return allContacts.filter(
      (c) =>
        c.name?.toLowerCase().includes(q) ||
        c.email?.toLowerCase().includes(q) ||
        c.accountName?.toLowerCase().includes(q),
    );
  }, [allContacts, searchValue]);

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const handleSearchChange = useCallback((v: string) => {
    setSearchValue(v);
  }, []);

  const renderItem = useCallback(
    ({ item, index }: { item: ContactData; index: number }) =>
      renderContactItem({ item, index }, colors, router),
    [colors, router],
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
      onEndReached={() => {
        if (hasNextPage && !isFetchingNextPage) fetchNextPage();
      }}
      emptyTitle="No contacts found"
      emptySubtitle="No contacts available"
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
  contactName: { fontSize: 16, fontWeight: '600', marginBottom: 2 },
  contactMeta: { fontSize: 13, marginTop: 2 },
});
