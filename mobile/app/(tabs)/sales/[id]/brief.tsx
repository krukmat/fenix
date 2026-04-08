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
import type { SalesBrief } from '../../../../src/services/api';
import type { ThemeColors } from '../../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
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

function BriefCard({
  title,
  children,
  colors,
  testID,
}: {
  title: string;
  children: React.ReactNode;
  colors: ThemeColors;
  testID: string;
}) {
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>{title}</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID={testID}>
        {children}
      </View>
    </View>
  );
}

function BriefContent({ brief, colors }: { brief: SalesBrief; colors: ThemeColors }) {
  const evidenceSummary = `${brief.evidencePack.source_count} sources · ${brief.evidencePack.confidence} confidence`;

  return (
    <View testID="sales-brief-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <ScrollView testID="sales-brief-scroll" style={styles.container}>
        <BriefCard title="Outcome" colors={colors} testID="sales-brief-outcome">
          <Text style={{ color: colors.onSurface }}>{brief.outcome}</Text>
        </BriefCard>

        <BriefCard title="Confidence" colors={colors} testID="sales-brief-confidence">
          <Text style={{ color: colors.onSurface }}>{brief.confidence}</Text>
        </BriefCard>

        {brief.summary ? (
          <BriefCard title="Summary" colors={colors} testID="sales-brief-summary">
            <Text style={{ color: colors.onSurface }}>{brief.summary}</Text>
          </BriefCard>
        ) : null}

        {brief.risks && brief.risks.length > 0 ? (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Risks</Text>
            <View testID="sales-brief-risks">
              {brief.risks.map((risk, i) => (
                <View key={i} style={[styles.recItem, { backgroundColor: colors.surface }]} testID={`sales-brief-risk-${i}`}>
                  <Text style={{ color: colors.onSurface }}>{risk}</Text>
                </View>
              ))}
            </View>
          </View>
        ) : null}

        {brief.nextBestActions && brief.nextBestActions.length > 0 ? (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Next Best Actions</Text>
            <View testID="sales-brief-next-best-actions">
              {brief.nextBestActions.map((action, i) => (
                <View key={i} style={[styles.recItem, { backgroundColor: colors.surface }]} testID={`sales-brief-next-best-action-${i}`}>
                  <Text style={{ color: colors.onSurface }}>{action.title}</Text>
                  {action.description ? (
                    <Text style={{ color: colors.onSurfaceVariant, marginTop: 4 }}>{action.description}</Text>
                  ) : null}
                </View>
              ))}
            </View>
          </View>
        ) : null}

        {brief.abstentionReason ? (
          <BriefCard title="Abstention Reason" colors={colors} testID="sales-brief-abstention-reason">
            <Text style={{ color: colors.onSurface }}>{brief.abstentionReason}</Text>
          </BriefCard>
        ) : null}

        <BriefCard title="Evidence Pack" colors={colors} testID="sales-brief-evidence-pack">
          <Text style={{ color: colors.onSurface }} testID="sales-brief-evidence-summary">
            {evidenceSummary}
          </Text>
          <Text style={{ color: colors.onSurfaceVariant, marginTop: 8 }} testID="sales-brief-evidence-query">
            Query: {brief.evidencePack.query}
          </Text>
          <Text style={{ color: colors.onSurfaceVariant, marginTop: 4 }} testID="sales-brief-evidence-methods">
            Methods: {brief.evidencePack.retrieval_methods_used.join(', ') || 'none'}
          </Text>
          {brief.evidencePack.warnings.length > 0 ? (
            <View style={styles.warningBlock} testID="sales-brief-evidence-warnings">
              {brief.evidencePack.warnings.map((warning, i) => (
                <Text key={i} style={{ color: colors.onSurface }} testID={`sales-brief-evidence-warning-${i}`}>
                  {warning}
                </Text>
              ))}
            </View>
          ) : null}
        </BriefCard>
      </ScrollView>
    </View>
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
  const brief = data;

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
  warningBlock: { marginTop: 10, gap: 6 },
});
