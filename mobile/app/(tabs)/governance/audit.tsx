import React, { useEffect, useState } from 'react';
import { ActivityIndicator, FlatList, StyleSheet, Text, View } from 'react-native';
import { useTheme } from 'react-native-paper';
import { AuditEventCard } from '../../../src/components/governance/AuditEventCard';
import { AuditFilterBar } from '../../../src/components/governance/AuditFilterBar';
import { useAuditEvents } from '../../../src/hooks/useWedge';
import type { AuditEvent, AuditFilters, PaginatedResponse } from '../../../src/services/api.types';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function mergeAuditPages(previous: AuditEvent[], nextPage: AuditEvent[], page: number) {
  if (page === 1) {
    const sameIds =
      previous.length === nextPage.length &&
      previous.every((event, index) => event.id === nextPage[index]?.id);
    return sameIds ? previous : nextPage;
  }

  const existingIds = new Set(previous.map((event) => event.id));
  const appended = nextPage.filter((event) => !existingIds.has(event.id));
  return appended.length === 0 ? previous : [...previous, ...appended];
}

function AuditStateMessage({
  testID,
  message,
  color,
  backgroundColor,
}: {
  testID: string;
  message: string;
  color: string;
  backgroundColor: string;
}) {
  return (
    <View style={[styles.centered, { backgroundColor }]} testID={testID}>
      <Text style={{ color }}>{message}</Text>
    </View>
  );
}

export default function GovernanceAuditScreen() {
  const colors = useColors();
  const [filters, setFilters] = useState<AuditFilters>({});
  const [page, setPage] = useState(1);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [allEvents, setAllEvents] = useState<AuditEvent[]>([]);
  const { data, isLoading, isFetching, error } = useAuditEvents(filters, page);
  const payload = data as PaginatedResponse<AuditEvent> | undefined;

  useEffect(() => {
    const nextPage = payload?.data;
    if (!nextPage) return;
    setAllEvents((previous) => mergeAuditPages(previous, nextPage, page));
  }, [page, payload?.data]);

  const handleFilterChange = (nextFilters: AuditFilters) => {
    setExpandedId(null); setAllEvents([]); setPage(1);
    setFilters(nextFilters);
  };

  const hasMore = allEvents.length < (payload?.meta.total ?? 0);

  if (isLoading && allEvents.length === 0) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="audit-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading audit trail...</Text>
      </View>
    );
  }

  if (error && allEvents.length === 0) {
    return (
      <AuditStateMessage
        testID="audit-error"
        message={(error as Error).message}
        color={colors.error}
        backgroundColor={colors.background}
      />
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]} testID="audit-screen">
      <AuditFilterBar filters={filters} onChange={handleFilterChange} />
      {allEvents.length === 0 ? (
        <AuditStateMessage
          testID="audit-empty"
          message="No audit events found"
          color={colors.onSurfaceVariant}
          backgroundColor="transparent"
        />
      ) : (
        <FlatList
          testID="audit-list"
          data={allEvents}
          keyExtractor={(item) => item.id}
          renderItem={({ item, index }) => (
            <AuditEventCard
              event={item}
              expanded={expandedId === item.id}
              onPress={() => setExpandedId((current) => (current === item.id ? null : item.id))}
              testIDPrefix={`audit-event-${index}`}
            />
          )}
          onEndReached={() => {
            if (!isFetching && hasMore) {
              setPage((current) => current + 1);
            }
          }}
          onEndReachedThreshold={0.4}
          ListFooterComponent={
            isFetching ? (
              <View style={styles.footer}>
                <ActivityIndicator size="small" color={colors.primary} />
              </View>
            ) : null
          }
          contentContainerStyle={styles.listContent}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  listContent: {
    paddingTop: 8,
    paddingBottom: 16,
  },
  footer: {
    paddingVertical: 16,
  },
});
