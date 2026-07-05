import React from 'react';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { useTheme } from 'react-native-paper';
import { useAgentRun } from '../../../src/hooks/useCRM';
import { useHandoffPackage } from '../../../src/hooks/useAgentSpec';
import { useRunUsage } from '../../../src/hooks/useWedge';
import { RunInspector, type RunInspectorDetail } from '../../../src/components/runs';
import { CenteredLoadingState, CenteredMessageState } from '../../../src/components/ui/ScreenState';
import { resolveWedgeHandoffPackageDestination, wedgeHref, wedgeHrefObject } from '../../../src/utils/navigation';
import type { UsageEvent } from '../../../src/services/api.types';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function activityRunHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Activity Run',
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

function useActivityRunDetailData() {
  const colors = useColors();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useAgentRun(id);
  const run = data?.data as RunInspectorDetail | undefined;
  const { data: usageData } = useRunUsage(id, Boolean(id));
  const usage = usageData as UsageEvent[] | undefined;
  const caseId = run?.entity_type === 'case' ? run.entity_id : undefined;
  const shouldLoadHandoff = run?.status === 'handed_off' && Boolean(run.id);
  const { data: handoff, isLoading: handoffLoading } = useHandoffPackage(run?.id, caseId, shouldLoadHandoff);

  return { colors, run, usage, handoff, handoffLoading, isLoading, error };
}

function ActivityRunDetailLoaded({
  colors,
  run,
  usage,
  handoff,
  handoffLoading,
}: {
  colors: ThemeColors;
  run: RunInspectorDetail;
  usage: UsageEvent[] | undefined;
  handoff: ReturnType<typeof useHandoffPackage>['data'];
  handoffLoading: boolean;
}) {
  const router = useRouter();

  return (
    <>
      <Stack.Screen options={activityRunHeaderOptions(colors)} />
      <RunInspector
        run={run}
        usage={usage}
        handoff={handoff}
        handoffLoading={handoffLoading}
        onOpenAuditTrail={() => {
          const traceId = run.trace_id ?? run.traceId;
          router.push(traceId ? wedgeHrefObject('/governance/audit', { trace_id: String(traceId) }) : wedgeHref('/governance/audit'));
        }}
        onOpenInbox={() => router.push(wedgeHref('/inbox'))}
        onOpenHandoffDestination={() => {
          if (!handoff) return;
          router.push(wedgeHref(resolveWedgeHandoffPackageDestination(handoff, run.id)));
        }}
        onViewFullUsage={() => router.push(wedgeHref(`/governance/usage?run_id=${run.id}`))}
      />
    </>
  );
}

export default function ActivityRunDetailScreen() {
  const { colors, run, usage, handoff, handoffLoading, isLoading, error } = useActivityRunDetailData();

  if (isLoading) {
    return <CenteredLoadingState
      testID="activity-run-detail-loading"
      backgroundColor={colors.background}
      indicatorColor={colors.primary}
      message="Loading run..."
      messageColor={colors.onSurfaceVariant}
    />;
  }

  if (error || !run) {
    return <CenteredMessageState
      testID="activity-run-detail-error"
      backgroundColor={colors.background}
      message="Unable to load this activity run."
      messageColor={colors.onSurfaceVariant}
    />;
  }

  return <ActivityRunDetailLoaded
    colors={colors}
    run={run}
    usage={usage}
    handoff={handoff}
    handoffLoading={handoffLoading}
  />;
}
