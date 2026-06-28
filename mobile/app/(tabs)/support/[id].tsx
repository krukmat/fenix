// Support wedge — case detail with agent trigger flow (W3-T2, W3-T3)
// No edit button — edit removed from wedge. Copilot: /support/[id]/copilot.
import React from 'react';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import {
  SupportCaseDetailContent,
  type SupportCaseDetailData,
} from '../../../src/components/support/SupportCaseDetailContent';
import { useCase } from '../../../src/hooks/useCRM';
import { useTriggerSupportAgent, useTriggerKBAgent } from '../../../src/hooks/useWedge';
import { CenteredLoadingState, CenteredMessageState } from '../../../src/components/ui/ScreenState';
import { wedgeHref } from '../../../src/utils/navigation';
import type { ThemeColors } from '../../../src/theme/types';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function s(o: Record<string, unknown> | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

type R = Record<string, unknown>;

function parseCaseCore(c: R, handoff: R | undefined): Omit<SupportCaseDetailData, 'accountName' | 'activeSignalCount'> {
  return {
    id: String(c.id ?? ''),
    subject: s(c, 'subject'),
    status: s(c, 'status') ?? 'open',
    priority: (s(c, 'priority') as SupportCaseDetailData['priority'] | undefined) ?? 'medium',
    description: s(c, 'description'),
    accountId: s(c, 'accountId') ?? s(c, 'account_id'),
    slaDeadline: s(c, 'slaDeadline') ?? s(c, 'sla_deadline'),
    handoffStatus: s(handoff, 'status') ?? s(c, 'handoffStatus'),
    assignee: s(c, 'assignee'),
  };
}

function parseCasePayload(data: unknown): SupportCaseDetailData | undefined {
  const payload = (data ?? null) as R | null;
  const c = (payload?.case as R | undefined) ?? payload ?? undefined;
  if (!c) return undefined;
  const acct = payload?.account as R | undefined;
  const handoff = payload?.handoff as R | undefined;
  const signalCount = payload?.active_signal_count;
  return {
    ...parseCaseCore(c, handoff),
    accountName: s(acct, 'name'),
    activeSignalCount: typeof signalCount === 'number' ? signalCount : 0,
  };
}

function formatSignalSummary(activeSignalCount?: number): string | null {
  if (!activeSignalCount || activeSignalCount <= 0) {
    return null;
  }

  return activeSignalCount === 1 ? '1 active signal' : `${activeSignalCount} active signals`;
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

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function SupportCaseDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useCase(id);
  const caseData = parseCasePayload(data);
  const triggerAgent = useTriggerSupportAgent();
  const triggerKB = useTriggerKBAgent();
  const signalSummary = caseData ? formatSignalSummary(caseData.activeSignalCount) : null;

  const handleKBTrigger = () => {
    if (!caseData) return;
    triggerKB.mutate({ caseId: caseData.id }, {
      onSuccess: (result) => {
        if (result?.runId) router.push(wedgeHref(`/activity/${result.runId}`));
      },
    });
  };

  if (isLoading) return <SupportCaseLoading colors={colors} />;
  if (error || !caseData) return <SupportCaseError colors={colors} message={error?.message || 'Case not found'} />;

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
      />
    </>
  );
}
