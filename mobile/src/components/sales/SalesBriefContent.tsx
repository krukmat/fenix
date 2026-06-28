import React from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import type { SalesBrief } from '../../services/api';
import type { ThemeColors } from '../../theme/types';

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

function BriefListSection({
  title,
  testID,
  children,
  colors,
}: {
  title: string;
  testID: string;
  children: React.ReactNode;
  colors: ThemeColors;
}) {
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>{title}</Text>
      <View testID={testID}>{children}</View>
    </View>
  );
}

export function SalesBriefContent({ brief, colors }: { brief: SalesBrief; colors: ThemeColors }) {
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
          <BriefListSection title="Risks" testID="sales-brief-risks" colors={colors}>
            {brief.risks.map((risk, i) => (
              <View key={i} style={[styles.recItem, { backgroundColor: colors.surface }]} testID={`sales-brief-risk-${i}`}>
                <Text style={{ color: colors.onSurface }}>{risk}</Text>
              </View>
            ))}
          </BriefListSection>
        ) : null}

        {brief.nextBestActions && brief.nextBestActions.length > 0 ? (
          <BriefListSection title="Next Best Actions" testID="sales-brief-next-best-actions" colors={colors}>
            {brief.nextBestActions.map((action, i) => (
              <View
                key={i}
                style={[styles.recItem, { backgroundColor: colors.surface }]}
                testID={`sales-brief-next-best-action-${i}`}
              >
                <Text style={{ color: colors.onSurface }}>{action.title}</Text>
                {action.description ? (
                  <Text style={{ color: colors.onSurfaceVariant, marginTop: 4 }}>{action.description}</Text>
                ) : null}
              </View>
            ))}
          </BriefListSection>
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

const styles = StyleSheet.create({
  container: { flex: 1 },
  section: { padding: 16 },
  sectionTitle: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 16, borderRadius: 8 },
  recItem: { padding: 12, borderRadius: 8, marginBottom: 8 },
  warningBlock: { marginTop: 10, gap: 6 },
});
