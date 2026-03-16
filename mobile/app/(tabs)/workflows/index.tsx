// Task Mobile P1.4 — FR-300/UC-A4: Workflows list screen

import React, { useCallback, useState } from 'react';
import { View, FlatList, RefreshControl, StyleSheet, ActivityIndicator } from 'react-native';
import { Text, Chip, Button, useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { WorkflowCard } from '../../../src/components/workflows/WorkflowCard';
import { useWorkflows } from '../../../src/hooks/useAgentSpec';
import type { Workflow, WorkflowStatus } from '../../../src/services/api';

const STATUS_FILTERS: (WorkflowStatus | 'all')[] = ['all', 'active', 'draft', 'testing', 'archived'];

function WorkflowListHeader({
  statusFilter,
  onSelectStatus,
  onCreate,
}: {
  statusFilter: WorkflowStatus | 'all';
  onSelectStatus: (status: WorkflowStatus | 'all') => void;
  onCreate: () => void;
}) {
  return (
    <View>
      <View style={styles.actionsRow}>
        <Button mode="contained" testID="workflows-new-btn" onPress={onCreate}>
          New Workflow
        </Button>
      </View>
      <View style={styles.chipRow} testID="workflows-filter-chips">
        {STATUS_FILTERS.map((status) => (
          <Chip
            key={status}
            selected={statusFilter === status}
            onPress={() => onSelectStatus(status)}
            style={styles.chip}
            testID={`workflows-chip-${status}`}
          >
            {status.charAt(0).toUpperCase() + status.slice(1)}
          </Chip>
        ))}
      </View>
    </View>
  );
}

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
          ListHeaderComponent={() => (
            <WorkflowListHeader
              statusFilter={statusFilter}
              onSelectStatus={setStatusFilter}
              onCreate={() => router.push('/workflows/new')}
            />
          )}
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
  actionsRow: { paddingHorizontal: 16, paddingTop: 16 },
  chipRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, padding: 16 },
  chip: { height: 32 },
  listContent: { paddingBottom: 24 },
  empty: { textAlign: 'center', marginTop: 48 },
  footer: { padding: 16 },
});
