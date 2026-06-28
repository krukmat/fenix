// Sales wedge — sales brief route (W4-T3)
// Route: /sales/[id]/brief  params: entity_type, entity_id
import React from 'react';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack } from 'expo-router';
import { SalesBriefContent } from '../../../../src/components/sales/SalesBriefContent';
import { CenteredLoadingState, CenteredMessageState } from '../../../../src/components/ui/ScreenState';
import { useSalesBrief } from '../../../../src/hooks/useWedge';
import type { ThemeColors } from '../../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function resolveEntityId(params: { id: string | string[]; entity_id?: string }): string {
  const rawId = Array.isArray(params.id) ? params.id[0] : params.id;
  if (params.entity_id) return params.entity_id;
  return rawId.startsWith('deal-') ? rawId.slice(5) : rawId;
}

function salesBriefHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Sales Brief',
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

export default function SalesBriefScreen() {
  const colors = useColors();
  const params = useLocalSearchParams<{ id: string | string[]; entity_type?: string; entity_id?: string }>();
  const entityType = params.entity_type ?? 'account';
  const entityId = resolveEntityId(params);

  const { data, isLoading, error } = useSalesBrief(entityType, entityId, true);
  const brief = data;

  if (isLoading) {
    return <CenteredLoadingState
      testID="sales-brief-loading"
      backgroundColor={colors.background}
      indicatorColor={colors.primary}
      message="Generating brief..."
      messageColor={colors.onSurfaceVariant}
    />;
  }

  if (error || !brief) {
    return <CenteredMessageState
      testID="sales-brief-error"
      backgroundColor={colors.background}
      message={(error as Error | null)?.message ?? 'Brief unavailable'}
      messageColor={colors.error}
    />;
  }

  return (
    <>
      <Stack.Screen options={salesBriefHeaderOptions(colors)} />
      <SalesBriefContent brief={brief} colors={colors} />
    </>
  );
}
