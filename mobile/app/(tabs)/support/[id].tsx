// Support wedge — case detail with agent trigger flow (W3-T2, W3-T3)
// No edit button — edit removed from wedge. Copilot: /support/[id]/copilot.
import React from 'react';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import {
  SupportCaseDetailContent,
} from '../../../src/components/support/SupportCaseDetailContent';
import { useCase, useCaseActivities, useCaseNotes, useEntityTimeline } from '../../../src/hooks/useCRM';
import { useTriggerSupportAgent, useTriggerKBAgent } from '../../../src/hooks/useWedge';
import { CenteredLoadingState, CenteredMessageState } from '../../../src/components/ui/ScreenState';
import { wedgeHref } from '../../../src/utils/navigation';
import type { ThemeColors } from '../../../src/theme/types';
import {
  buildCaseTimeline,
  formatSignalSummary,
  parseCasePayload,
} from '../../../src/components/support/supportCaseDetail.model';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function normalizeParamId(id: string | string[] | undefined): string | undefined {
  return Array.isArray(id) ? id[0] : id;
}

function supportDetailHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Support Case',
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

function SupportCaseLoading({ colors }: { colors: ThemeColors }) {
  return <CenteredLoadingState
    testID="support-case-detail-loading"
    backgroundColor={colors.background}
    indicatorColor={colors.primary}
    message="Loading case..."
    messageColor={colors.onSurfaceVariant}
  />;
}

function SupportCaseError({ colors, message }: { colors: ThemeColors; message: string }) {
  return <CenteredMessageState
    testID="support-case-detail-error"
    backgroundColor={colors.background}
    message={message}
    messageColor={colors.error}
  />;
}

function useSupportCaseDetailModel(id: string | undefined, router: ReturnType<typeof useRouter>) {
  const caseQuery = useCase(id ?? '');
  const timelineQuery = useEntityTimeline('case', id ?? '');
  const activitiesQuery = useCaseActivities(id ?? '');
  const notesQuery = useCaseNotes(id ?? '');
  const caseData = parseCasePayload(caseQuery.data);
  const triggerAgent = useTriggerSupportAgent();
  const triggerKB = useTriggerKBAgent();
  const signalSummary = caseData ? formatSignalSummary(caseData.activeSignalCount) : null;
  const timelineEvents = buildCaseTimeline(timelineQuery.data, activitiesQuery.data, notesQuery.data);
  const timelineLoading = timelineQuery.isLoading || activitiesQuery.isLoading || notesQuery.isLoading;

  const handleKBTrigger = () => {
    if (!caseData) return;
    triggerKB.mutate({ caseId: caseData.id }, {
      onSuccess: (result) => {
        if (result?.runId) router.push(wedgeHref(`/activity/${result.runId}`));
      },
    });
  };

  return {
    caseQuery,
    caseData,
    triggerAgent,
    triggerKB,
    signalSummary,
    timelineEvents,
    timelineLoading,
    handleKBTrigger,
  };
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function SupportCaseDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = normalizeParamId(params.id);
  const {
    caseQuery,
    caseData,
    triggerAgent,
    triggerKB,
    signalSummary,
    timelineEvents,
    timelineLoading,
    handleKBTrigger,
  } = useSupportCaseDetailModel(id, router);

  if (caseQuery.isLoading) return <SupportCaseLoading colors={colors} />;
  if (caseQuery.error || !caseData) {
    return <SupportCaseError colors={colors} message={caseQuery.error?.message || 'Case not found'} />;
  }

  return (
    <>
      <Stack.Screen options={supportDetailHeaderOptions(colors)} />
      <SupportCaseDetailContent
        caseData={caseData}
        colors={colors}
        router={router}
        signalSummary={signalSummary}
        triggerAgent={triggerAgent}
        triggerKBIsPending={triggerKB.isPending}
        onTriggerKB={handleKBTrigger}
        timelineEvents={timelineEvents}
        timelineLoading={timelineLoading}
      />
    </>
  );
}
