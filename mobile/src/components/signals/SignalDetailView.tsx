// Task Mobile P1.3 — UC-A5/B4: Full signal detail with evidence list

import React from 'react';
import { ScrollView, View, StyleSheet } from 'react-native';
import { Text, Divider, useTheme } from 'react-native-paper';
import { EvidenceCard } from '../copilot/EvidenceCard';
import type { Signal } from '../../services/api';
import type { EvidenceSource } from '../../services/sse';
import { brandColors } from '../../theme/colors';
import { getConfidenceColor, getConfidenceLabel } from '../../theme/semantic';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

interface SignalDetailViewProps {
  signal: Signal;
  testIDPrefix?: string;
}

/** Map signal evidence_ids to minimal EvidenceSource stubs for EvidenceCard */
function toEvidenceSources(signal: Signal): EvidenceSource[] {
  return signal.evidence_ids.map((id) => ({
    id,
    snippet: `Evidence ID: ${id}`,
    score: signal.confidence,
    timestamp: signal.created_at,
    title: id,
  }));
}

export function SignalDetailView({ signal, testIDPrefix = 'signal-detail' }: SignalDetailViewProps) {
  const theme = useTheme();
  const color = getConfidenceColor(signal.confidence);
  const sources = toEvidenceSources(signal);

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      testID={testIDPrefix}
    >
      {/* Header */}
      <View style={styles.header}>
        <Text variant="titleMedium" testID={`${testIDPrefix}-type`}>
          {signal.signal_type}
        </Text>
        <View
          testID={`${testIDPrefix}-confidence`}
          style={[styles.confidenceBadge, { backgroundColor: color }]}
        >
          <Text style={styles.confidenceText}>
            {`${getConfidenceLabel(signal.confidence)} · ${(signal.confidence * 100).toFixed(0)}%`}
          </Text>
        </View>
      </View>

      <Text
        variant="bodySmall"
        style={{ color: theme.colors.onSurfaceVariant }}
        testID={`${testIDPrefix}-entity`}
      >
        {`${signal.entity_type} · ${signal.entity_id}`}
      </Text>

      <Divider style={styles.divider} />

      {/* Metadata summary */}
      {typeof signal.metadata?.['summary'] === 'string' && (
        <Text variant="bodyMedium" style={styles.summary} testID={`${testIDPrefix}-summary`}>
          {signal.metadata['summary'] as string}
        </Text>
      )}

      {/* Evidence */}
      <Text
        variant="labelMedium"
        style={[styles.sectionLabel, { color: theme.colors.onSurfaceVariant }]}
      >
        Evidence ({sources.length})
      </Text>

      {sources.length === 0 ? (
        <Text
          variant="bodySmall"
          style={{ color: theme.colors.onSurfaceVariant }}
          testID={`${testIDPrefix}-no-evidence`}
        >
          No evidence available
        </Text>
      ) : (
        sources.map((src, i) => (
          <EvidenceCard
            key={src.id}
            source={src}
            index={i + 1}
            testIDPrefix={`${testIDPrefix}-evidence-${i}`}
          />
        ))
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: spacing.base, paddingBottom: spacing.xxl },
  header: { flexDirection: 'row', alignItems: 'center', gap: radius.md, flexWrap: 'wrap', marginBottom: spacing.xs },
  confidenceBadge: { borderRadius: radius.full, paddingHorizontal: spacing.sm, paddingVertical: spacing.xs },
  confidenceText: { color: brandColors.onError, ...typography.labelMD },
  divider: { marginVertical: spacing.md },
  summary: { marginBottom: spacing.base },
  sectionLabel: { ...typography.eyebrow, marginBottom: spacing.sm },
});
