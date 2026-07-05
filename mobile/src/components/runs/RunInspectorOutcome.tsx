import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import type { UsageEvent } from '../../services/api.types';
import { typography } from '../../theme/typography';
import { UsageDetailCard } from '../governance/UsageDetailCard';
import { formatJson, sumUsage } from './runInspector.model';
import { EmptyState, Section, sharedStyles, SummaryMetric } from './runInspector.shared';

function useSectionColors() {
  return useTheme().colors;
}

export function ReasoningTraceSection({ trace }: { trace: string[] }) {
  const colors = useSectionColors();
  if (trace.length === 0) return null;

  return (
    <Section title="Reasoning Trace" testID="activity-detail-reasoning">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        {trace.map((step, index) => (
          <View key={`${index}-${step}`} style={[sharedStyles.subpanel, { borderColor: colors.outline }]}>
            <Text style={[styles.traceStepIndex, { color: colors.onSurfaceVariant }]}>{`Step ${index + 1}`}</Text>
            <Text style={{ color: colors.onSurface }}>{step}</Text>
          </View>
        ))}
      </View>
    </Section>
  );
}

export function OutputSection({ output }: { output: unknown }) {
  const colors = useSectionColors();
  return (
    <Section title="Outcome Payload" testID="activity-detail-output">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        <Text style={[typography.monoSM, { color: colors.onSurface }]}>{formatJson(output)}</Text>
      </View>
    </Section>
  );
}

export function UsageSection({
  runId,
  events,
  onViewFullUsage,
}: {
  runId: string;
  events: UsageEvent[] | undefined;
  onViewFullUsage: () => void;
}) {
  const colors = useSectionColors();
  const totals = sumUsage(events);

  return (
    <Section title="Run Usage" testID="activity-detail-usage">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        <View style={sharedStyles.summaryGrid}>
          <SummaryMetric label="Input units" value={String(totals.inputUnits)} monospace />
          <SummaryMetric label="Output units" value={String(totals.outputUnits)} monospace />
          <SummaryMetric label="Estimated cost" value={`€${totals.estimatedCost.toFixed(5)}`} monospace />
          <SummaryMetric label="Latency sum" value={`${totals.latencyMs} ms`} monospace />
        </View>
        <Pressable testID="activity-view-full-usage" onPress={onViewFullUsage} style={sharedStyles.inlineAction}>
          <Text style={[sharedStyles.inlineActionText, { color: colors.primary }]}>View full usage</Text>
        </Pressable>
        {!events?.length ? (
          <EmptyState message="No usage data recorded" />
        ) : (
          events.map((event, index) => (
            <UsageDetailCard
              key={`${runId}-${event.id}`}
              event={event}
              testIDPrefix={`activity-usage-item-${index}`}
            />
          ))
        )}
      </View>
    </Section>
  );
}

const styles = StyleSheet.create({
  traceStepIndex: {
    ...typography.eyebrow,
  },
});
