import React, { useState } from 'react';
import { useLocalSearchParams } from 'expo-router';
import { useTheme } from 'react-native-paper';
import { UsageEventsList } from '../../../src/components/governance/UsageEventsList';
import { CenteredLoadingState, CenteredMessageState } from '../../../src/components/ui/ScreenState';
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
      <CenteredLoadingState
        testID="usage-loading"
        backgroundColor={colors.background}
        indicatorColor={colors.primary}
        message="Loading usage events..."
        messageColor={colors.onSurfaceVariant}
      />
    );
  }

  if (error) {
    return (
      <CenteredMessageState
        testID="usage-error"
        backgroundColor={colors.background}
        message={(error as Error).message}
        messageColor={colors.error}
      />
    );
  }

  if (events.length === 0) {
    return (
      <CenteredMessageState
        testID="usage-empty"
        backgroundColor={colors.background}
        message="No usage events found"
        messageColor={colors.onSurfaceVariant}
      />
    );
  }

  return (
    <UsageEventsList
      backgroundColor={colors.background}
      surfaceColor={colors.surface}
      textColor={colors.onSurface}
      spinnerColor={colors.primary}
      events={events}
      summary={summary}
      runId={runId}
      isFetching={isFetching}
      hasMore={hasMore}
      onReachEnd={() => setPage((current) => current + 1)}
    />
  );
}
