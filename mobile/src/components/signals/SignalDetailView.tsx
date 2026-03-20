// Task Mobile P1.3 — UC-A5/B4: Full signal detail with evidence list

import React from 'react';
import { ScrollView, View, StyleSheet } from 'react-native';
import { Text, Chip, Divider, useTheme } from 'react-native-paper';
import { EvidenceCard } from '../copilot/EvidenceCard';
import type { Signal } from '../../services/api';
import type { EvidenceSource } from '../../services/sse';

interface SignalDetailViewProps {
  signal: Signal;
  testIDPrefix?: string;
}

function confidenceColor(confidence: number): string {
  if (confidence >= 0.8) return '#2e7d32';
  if (confidence >= 0.5) return '#e65100';
  return '#757575';
}

function confidenceLabel(confidence: number): string {
  if (confidence >= 0.8) return 'High';
  if (confidence >= 0.5) return 'Medium';
  return 'Low';
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
  const color = confidenceColor(signal.confidence);
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
        <Chip
          compact
          testID={`${testIDPrefix}-confidence`}
          style={[styles.confidenceChip, { backgroundColor: color }]}
          textStyle={styles.confidenceText}
        >
          {`${confidenceLabel(signal.confidence)} · ${(signal.confidence * 100).toFixed(0)}%`}
        </Chip>
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
  content: { padding: 16, paddingBottom: 32 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginBottom: 4 },
  confidenceChip: { height: 24 },
  confidenceText: { color: '#ffffff', fontSize: 11 },
  divider: { marginVertical: 12 },
  summary: { marginBottom: 16 },
  sectionLabel: { marginBottom: 8, textTransform: 'uppercase', letterSpacing: 0.5 },
});
