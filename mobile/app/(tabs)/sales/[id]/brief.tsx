// Sales wedge — sales brief route (W4-T3)
// Route: /sales/[id]/brief  params: entity_type, entity_id
import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
} from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack } from 'expo-router';
import { useSalesBrief } from '../../../../src/hooks/useWedge';
import type { ThemeColors } from '../../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface BriefData {
  summary?: string;
  recommendations?: string[];
}

function BriefLoading({ colors }: { colors: ThemeColors }) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-brief-loading">
      <ActivityIndicator size="large" color={colors.primary} />
      <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Generating brief...</Text>
    </View>
  );
}

function BriefError({ colors, message }: { colors: ThemeColors; message: string }) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-brief-error">
      <Text style={{ color: colors.error, fontSize: 16 }}>{message}</Text>
    </View>
  );
}

function BriefContent({ brief, colors }: { brief: BriefData; colors: ThemeColors }) {
  return (
    <ScrollView testID="sales-brief-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      {brief.summary ? (
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Summary</Text>
          <View style={[styles.card, { backgroundColor: colors.surface }]} testID="sales-brief-summary">
            <Text style={{ color: colors.onSurface }}>{brief.summary}</Text>
          </View>
        </View>
      ) : null}
      {brief.recommendations && brief.recommendations.length > 0 ? (
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Recommendations</Text>
          {brief.recommendations.map((rec, i) => (
            <View key={i} style={[styles.recItem, { backgroundColor: colors.surface }]} testID={`sales-brief-rec-${i}`}>
              <Text style={{ color: colors.onSurface }}>{rec}</Text>
            </View>
          ))}
        </View>
      ) : null}
    </ScrollView>
  );
}

function resolveEntityId(params: { id: string | string[]; entity_id?: string }): string {
  const rawId = Array.isArray(params.id) ? params.id[0] : params.id;
  if (params.entity_id) return params.entity_id;
  return rawId.startsWith('deal-') ? rawId.slice(5) : rawId;
}

export default function SalesBriefScreen() {
  const colors = useColors();
  const params = useLocalSearchParams<{ id: string | string[]; entity_type?: string; entity_id?: string }>();
  const entityType = params.entity_type ?? 'account';
  const entityId = resolveEntityId(params);

  const { data, isLoading, error } = useSalesBrief(entityType, entityId, true);
  const brief = data as BriefData | null | undefined;

  if (isLoading) return <BriefLoading colors={colors} />;
  if (error || !brief) {
    return <BriefError colors={colors} message={(error as Error | null)?.message ?? 'Brief unavailable'} />;
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Sales Brief' }} />
      <BriefContent brief={brief} colors={colors} />
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  section: { padding: 16 },
  sectionTitle: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 16, borderRadius: 8 },
  recItem: { padding: 12, borderRadius: 8, marginBottom: 8 },
});
