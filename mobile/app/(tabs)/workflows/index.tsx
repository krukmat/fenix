// Task Mobile P1.4 — FR-300/UC-A4: Workflows list screen

import React, { useState, useCallback } from 'react';
import { View, FlatList, RefreshControl, StyleSheet, ActivityIndicator } from 'react-native';
import { Text, Chip, useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { WorkflowCard } from '../../../src/components/workflows/WorkflowCard';
import { useWorkflows } from '../../../src/hooks/useAgentSpec';
import type { Workflow, WorkflowStatus } from '../../../src/services/api';

const STATUS_FILTERS: Array<WorkflowStatus | 'all'> = ['all', 'active', 'draft', 'testing', 'archived'];

export default function WorkflowsListScreen() {
  const theme = useTheme();
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<WorkflowStatus | 'all'>('all');

  const filters = statusFilter !== 'all' ? { status: statusFilter as WorkflowStatus } : undefined;
  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, refetch, isRefetching } =
    useWorkflows(filters);

  const workflows: Workflow[] = (data?.pages ?? []).flat() as Workflow[];

  const handleRefresh = useCallback(() => refetch(), [refetch]);

  const handleEndReached = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) fetchNextPage();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const renderItem = useCallback(
    ({ item }: { item: Workflow }) => (
      <WorkflowCard
        workflow={item}
        onPress={() => router.push(`/workflows/${item.id}`)}
        testIDPrefix={`workflow-${item.id}`}
      />
    ),
    [router]
  );

  const FilterChips = (
    <View style={styles.chipRow} testID="workflows-filter-chips">
      {STATUS_FILTERS.map((f) => (
        <Chip
          key={f}
          selected={statusFilter === f}
          onPress={() => setStatusFilter(f)}
          style={styles.chip}
          testID={`workflows-chip-${f}`}
        >
          {f.charAt(0).toUpperCase() + f.slice(1)}
        </Chip>
      ))}
    </View>
  );

  if (isLoading && workflows.length === 0) {
    return (
      <View style={[styles.centered, { backgroundColor: theme.colors.background }]}>
        <ActivityIndicator size="large" color={theme.colors.primary} />
      </View>
    );
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Workflows', headerShown: true }} />
      <View style={[styles.container, { backgroundColor: theme.colors.background }]} testID="workflows-list">
        <FlatList
          data={workflows}
          keyExtractor={(item) => item.id}
          renderItem={renderItem}
          ListHeaderComponent={() => FilterChips}
          ListEmptyComponent={() => (
            <Text
              variant="bodyMedium"
              style={[styles.empty, { color: theme.colors.onSurfaceVariant }]}
              testID="workflows-empty"
            >
              No workflows found
            </Text>
          )}
          ListFooterComponent={() =>
            isFetchingNextPage ? (
              <ActivityIndicator size="small" color={theme.colors.primary} style={styles.footer} />
            ) : null
          }
          refreshControl={
            <RefreshControl refreshing={isRefetching} onRefresh={handleRefresh} tintColor={theme.colors.primary} />
          }
          onEndReached={handleEndReached}
          onEndReachedThreshold={0.5}
          contentContainerStyle={styles.listContent}
          showsVerticalScrollIndicator={false}
          testID="workflows-flatlist"
        />
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  chipRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, padding: 16 },
  chip: { height: 32 },
  listContent: { paddingBottom: 24 },
  empty: { textAlign: 'center', marginTop: 48 },
  footer: { padding: 16 },
});
