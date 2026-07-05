import React, { useEffect, useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { useLocalSearchParams } from 'expo-router';
import { useTheme } from 'react-native-paper';
import { AuditEventsList } from '../../../src/components/governance/AuditEventsList';
import { AuditFilterBar } from '../../../src/components/governance/AuditFilterBar';
import { CenteredLoadingState, CenteredMessageState } from '../../../src/components/ui/ScreenState';
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

export default function GovernanceAuditScreen() {
  const colors = useColors();
  const params = useLocalSearchParams<{ trace_id?: string | string[] }>();
  const traceIdParam = Array.isArray(params.trace_id) ? params.trace_id[0] : params.trace_id;

  const [filters, setFilters] = useState<AuditFilters>(() =>
    traceIdParam ? { trace_id: traceIdParam } : {}
  );
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
      <CenteredLoadingState
        testID="audit-loading"
        backgroundColor={colors.background}
        indicatorColor={colors.primary}
        message="Loading audit trail..."
        messageColor={colors.onSurfaceVariant}
      />
    );
  }

  if (error && allEvents.length === 0) {
    return (
      <CenteredMessageState
        testID="audit-error"
        message={(error as Error).message}
        messageColor={colors.error}
        backgroundColor={colors.background}
      />
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]} testID="audit-screen">
      <AuditFilterBar filters={filters} onChange={handleFilterChange} />
      <AuditEventsList
        backgroundColor={colors.background}
        emptyColor={colors.onSurfaceVariant}
        spinnerColor={colors.primary}
        events={allEvents}
        expandedId={expandedId}
        isFetching={isFetching}
        hasMore={hasMore}
        onToggleExpanded={(id) => setExpandedId((current) => (current === id ? null : id))}
        onReachEnd={() => setPage((current) => current + 1)}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
