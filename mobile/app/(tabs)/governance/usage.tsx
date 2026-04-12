import React, { useState } from 'react';
import { ActivityIndicator, FlatList, StyleSheet, Text, View } from 'react-native';
import { useLocalSearchParams } from 'expo-router';
import { useTheme } from 'react-native-paper';
import { UsageCostSummaryCard } from '../../../src/components/governance/UsageCostSummaryCard';
import { UsageDetailCard } from '../../../src/components/governance/UsageDetailCard';
import { useUsageEvents } from '../../../src/hooks/useWedge';
import type { PaginatedResponse, UsageCostSummary, UsageEvent, UsageFilters } from '../../../src/services/api.types';
import type { ThemeColors } from '../../../src/theme/types';

const PAGE_SIZE = 20;

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function summarizeUsage(events: UsageEvent[]): UsageCostSummary {
  return events.reduce<UsageCostSummary>(
    (summary, event) => ({
      totalCost: summary.totalCost + (event.estimatedCost ?? 0),
      totalInputUnits: summary.totalInputUnits + (event.inputUnits ?? 0),
      totalOutputUnits: summary.totalOutputUnits + (event.outputUnits ?? 0),
      eventCount: summary.eventCount + 1,
    }),
    { totalCost: 0, totalInputUnits: 0, totalOutputUnits: 0, eventCount: 0 }
  );
}

export default function GovernanceUsageScreen() {
  const colors = useColors();
  const params = useLocalSearchParams<{ run_id?: string | string[] }>();
  const runId = Array.isArray(params.run_id) ? params.run_id[0] : params.run_id;
  const filters: UsageFilters | undefined = runId ? { run_id: runId } : undefined;
  const [page, setPage] = useState(1);
  const { data, isLoading, isFetching, error } = useUsageEvents(filters, page);
  const payload = data as PaginatedResponse<UsageEvent> | undefined;
  const events = payload?.data ?? [];
  const summary = summarizeUsage(events);
  const requestedLimit = page * PAGE_SIZE;
  const hasMore = events.length >= requestedLimit;

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="usage-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading usage events...</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="usage-error">
        <Text style={{ color: colors.error }}>{(error as Error).message}</Text>
      </View>
    );
  }

  if (events.length === 0) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="usage-empty">
        <Text style={{ color: colors.onSurfaceVariant }}>No usage events found</Text>
      </View>
    );
  }

  return (
    <FlatList
      testID="usage-screen"
      style={{ backgroundColor: colors.background }}
      data={events}
      keyExtractor={(item) => item.id}
      renderItem={({ item, index }) => (
        <UsageDetailCard event={item} testIDPrefix={`usage-event-${index}`} />
      )}
      onEndReached={() => {
        if (!isFetching && hasMore) {
          setPage((current) => current + 1);
        }
      }}
      onEndReachedThreshold={0.4}
      ListHeaderComponent={
        <View>
          <UsageCostSummaryCard summary={summary} testIDPrefix="usage-summary" />
          {runId ? (
            <View style={[styles.filterBanner, { backgroundColor: colors.surface }]} testID="usage-run-filter">
              <Text style={{ color: colors.onSurface }}>
                Filtered by run <Text style={{ fontWeight: '700' }}>{runId}</Text>
              </Text>
            </View>
          ) : null}
        </View>
      }
      ListFooterComponent={
        isFetching ? (
          <View style={styles.footer}>
            <ActivityIndicator size="small" color={colors.primary} />
          </View>
        ) : null
      }
      contentContainerStyle={styles.listContent}
    />
  );
}

const styles = StyleSheet.create({
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  listContent: {
    paddingBottom: 16,
  },
  filterBanner: {
    marginHorizontal: 16,
    marginBottom: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderRadius: 8,
  },
  footer: {
    paddingVertical: 16,
  },
});
