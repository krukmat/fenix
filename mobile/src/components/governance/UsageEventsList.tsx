import React from 'react';
import { ActivityIndicator, FlatList, StyleSheet, Text, View } from 'react-native';
import { UsageCostSummaryCard } from './UsageCostSummaryCard';
import { UsageDetailCard } from './UsageDetailCard';
import type { UsageCostSummary, UsageEvent } from '../../services/api.types';

interface UsageEventsListProps {
  backgroundColor: string;
  surfaceColor: string;
  textColor: string;
  spinnerColor: string;
  events: UsageEvent[];
  summary: UsageCostSummary;
  runId?: string;
  isFetching: boolean;
  hasMore: boolean;
  onReachEnd: () => void;
}

export function UsageEventsList({
  backgroundColor,
  surfaceColor,
  textColor,
  spinnerColor,
  events,
  summary,
  runId,
  isFetching,
  hasMore,
  onReachEnd,
}: UsageEventsListProps) {
  return (
    <FlatList
      testID="usage-screen"
      style={[styles.list, { backgroundColor }]}
      data={events}
      keyExtractor={(item) => item.id}
      renderItem={({ item, index }) => (
        <UsageDetailCard event={item} testIDPrefix={`usage-event-${index}`} />
      )}
      onEndReached={() => {
        if (!isFetching && hasMore) {
          onReachEnd();
        }
      }}
      onEndReachedThreshold={0.4}
      ListHeaderComponent={
        <View>
          <UsageCostSummaryCard summary={summary} testIDPrefix="usage-summary" />
          {runId ? (
            <View style={[styles.filterBanner, { backgroundColor: surfaceColor }]} testID="usage-run-filter">
              <Text style={{ color: textColor }}>
                Filtered by run <Text style={styles.bold}>{runId}</Text>
              </Text>
            </View>
          ) : null}
        </View>
      }
      ListFooterComponent={
        isFetching ? (
          <View style={styles.footer}>
            <ActivityIndicator size="small" color={spinnerColor} />
          </View>
        ) : null
      }
      contentContainerStyle={styles.listContent}
    />
  );
}

const styles = StyleSheet.create({
  list: {
    flex: 1,
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
  bold: {
    fontWeight: '700',
  },
});
