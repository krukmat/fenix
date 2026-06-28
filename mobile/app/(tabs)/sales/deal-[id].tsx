// Sales wedge — deal detail (W4-T2)
// Read-only: no edit button. Actions: Sales Brief + Copilot.
import React from 'react';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { SalesDealDetailContent, type SalesDealDetailData } from '../../../src/components/sales/SalesDealDetailContent';
import { CenteredLoadingState, CenteredMessageState } from '../../../src/components/ui/ScreenState';
import { useDeal } from '../../../src/hooks/useCRM';
import { useTriggerDealRiskAgent } from '../../../src/hooks/useWedge';
import type { ThemeColors } from '../../../src/theme/types';

type R = Record<string, unknown>;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function s(o: R | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

function parseAmount(deal: R): number | undefined {
  const raw = deal.amount ?? deal.value;
  return typeof raw === 'number' ? raw : undefined;
}

function parseDealCore(deal: R): Omit<SalesDealDetailData, 'accountName' | 'activeSignalCount'> {
  return {
    id: String(deal.id),
    title: s(deal, 'title') ?? s(deal, 'name') ?? 'Unnamed Deal',
    status: s(deal, 'status') ?? 'open',
    amount: parseAmount(deal),
    stage: s(deal, 'stage'),
    closeDate: s(deal, 'closeDate') ?? s(deal, 'close_date'),
    accountId: s(deal, 'accountId') ?? s(deal, 'account_id'),
  };
}

function parseDealPayload(data: unknown): SalesDealDetailData | undefined {
  const payload = (data ?? null) as R | null;
  if (!payload) return undefined;
  const deal = (payload.deal as R | undefined) ?? payload;
  if (!deal?.id) return undefined;
  const acct = payload.account as R | undefined;
  const signalCount = payload.active_signal_count;
  return {
    ...parseDealCore(deal),
    accountName: s(acct, 'name'),
    activeSignalCount: typeof signalCount === 'number' ? signalCount : 0,
  };
}

function salesDealHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Sales Deal',
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

export default function SalesDealDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const rawId = Array.isArray(params.id) ? params.id[0] : params.id;
  const dealId = rawId.startsWith('deal-deal-') ? rawId.slice(5) : rawId;
  const dealRouteId = `deal-${dealId}`;
  const { data, isLoading, error } = useDeal(dealId);
  const triggerDealRiskAgent = useTriggerDealRiskAgent();
  const dealData = parseDealPayload(data);

  if (isLoading) {
    return <CenteredLoadingState
      testID="sales-deal-detail-loading"
      backgroundColor={colors.background}
      indicatorColor={colors.primary}
      message="Loading deal..."
      messageColor={colors.onSurfaceVariant}
    />;
  }

  if (error || !dealData) {
    return <CenteredMessageState
      testID="sales-deal-detail-error"
      backgroundColor={colors.background}
      message={error?.message || 'Deal not found'}
      messageColor={colors.error}
    />;
  }

  const handleDealRiskTrigger = (currentDealId: string, onSuccess: (runId: string) => void) => {
    triggerDealRiskAgent.mutate(
      { dealId: currentDealId, language: 'es' },
      {
        onSuccess: (result) => {
          if (result?.runId) onSuccess(result.runId);
        },
      },
    );
  };

  return (
    <>
      <Stack.Screen options={salesDealHeaderOptions(colors)} />
      <SalesDealDetailContent
        dealData={dealData}
        dealRouteId={dealRouteId}
        colors={colors}
        router={router}
        riskPending={triggerDealRiskAgent.isPending}
        onTriggerRisk={handleDealRiskTrigger}
      />
    </>
  );
}
