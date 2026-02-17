// Task 4.3 — Cases List Screen

import React, { useState, useMemo, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMListScreen } from '../../../src/components/crm';
import { useCases } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

interface CaseData {
  id: string;
  subject?: string;
  status: string;
  priority: 'low' | 'medium' | 'high';
  accountName?: string;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function getPriorityColor(priority: string): string {
  if (priority === 'high') return '#EF4444';
  if (priority === 'medium') return '#F59E0B';
  return '#10B981';
}

export default function CasesListScreen() {
  const colors = useThemeColors();
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
    if (statusFilter) {
      result = result.filter((c: CaseData) => c.status === statusFilter);
    }
    if (!searchValue.trim()) return result;
    const searchLower = searchValue.toLowerCase();
    return result.filter(
      (c: CaseData) =>
        c.subject?.toLowerCase().includes(searchLower) ||
        c.accountName?.toLowerCase().includes(searchLower)
    );
  }, [allCases, searchValue, statusFilter]);

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
    ({ item }: { item: CaseData }) => (
      <TouchableOpacity
        style={[styles.caseItem, { backgroundColor: colors.surface }]}
        onPress={() => router.push(`/cases/${item.id}`)}
        testID={`case-item-${item.id}`}
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
      </TouchableOpacity>
    ),
    [colors, router]
  );

  return (
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
      emptySubtitle="Pull to refresh or add a new case"
      testIDPrefix="cases"
      isRefreshing={isRefetching}
      onRetry={handleRefresh}
      statusFilter={statusFilter}
      onStatusFilterChange={handleStatusChange}
      availableStatuses={['open', 'closed', 'pending']}
    />
  );
}

const styles = StyleSheet.create({
  caseItem: { padding: 16, marginHorizontal: 16, marginBottom: 12, borderRadius: 8, elevation: 2 },
  caseHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 },
  caseSubject: { fontSize: 16, fontWeight: '600', flex: 1 },
  priorityBadge: { paddingHorizontal: 8, paddingVertical: 4, borderRadius: 12 },
  priorityText: { color: '#FFFFFF', fontSize: 12, fontWeight: '500' },
  caseAccount: { fontSize: 14 },
  caseStatus: { fontSize: 12, marginTop: 4 },
});
