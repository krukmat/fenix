import React from 'react';
import { Text, useTheme } from 'react-native-paper';
import { StyleSheet, View } from 'react-native';
import { brandColors, semanticColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import { EvidenceCard } from '../copilot/EvidenceCard';
import {
  extractEvidenceMeta,
  formatJson,
  formatLabel,
  formatMs,
  formatTimestamp,
  type NormalizedToolCall,
  type RunEvidenceItem,
  type RunInspectorDetail,
} from './runInspector.model';
import { EmptyState, MetaChip, Section, sharedStyles } from './runInspector.shared';

function useSectionColors() {
  return useTheme().colors;
}

function EvidenceMetaBlock({
  run,
  itemCount,
  colors,
}: {
  run: RunInspectorDetail;
  itemCount: number;
  colors: ReturnType<typeof useSectionColors>;
}) {
  const evidenceMeta = extractEvidenceMeta(run);
  const chips = [
    evidenceMeta?.confidence ? `${formatLabel(evidenceMeta.confidence)} confidence` : null,
    evidenceMeta?.schemaVersion ? `Schema ${evidenceMeta.schemaVersion}` : null,
    `${itemCount} source${itemCount === 1 ? '' : 's'}`,
  ].filter((value): value is string => Boolean(value));

  return (
    <>
      <View style={sharedStyles.metaRow}>
        {chips.map((label, index) => (
          <MetaChip key={`${label}-${index}`} label={label} tone={index === 0 && evidenceMeta?.confidence ? 'status' : 'neutral'} />
        ))}
      </View>
      <EvidenceMethods methods={evidenceMeta?.retrievalMethodsUsed} labelColor={colors.onSurfaceVariant} />
      <EvidenceWarnings warnings={evidenceMeta?.warnings} labelColor={colors.onSurfaceVariant} />
      {evidenceMeta?.builtAt ? (
        <Text style={[typography.monoSM, { color: colors.onSurfaceVariant }]}>
          Built at {formatTimestamp(evidenceMeta.builtAt)}
        </Text>
      ) : null}
    </>
  );
}

function EvidenceMethods({
  methods,
  labelColor,
}: {
  methods: string[] | undefined;
  labelColor: string;
}) {
  if (!methods?.length) return null;

  return (
    <View style={styles.subsection}>
      <Text style={[styles.subsectionTitle, { color: labelColor }]}>Retrieval methods</Text>
      <View style={sharedStyles.metaRow}>
        {methods.map((method) => (
          <MetaChip key={method} label={formatLabel(method)} />
        ))}
      </View>
    </View>
  );
}

function EvidenceWarnings({
  warnings,
  labelColor,
}: {
  warnings: string[] | undefined;
  labelColor: string;
}) {
  if (!warnings?.length) return null;

  return (
    <View style={styles.subsection} testID="activity-detail-evidence-warnings">
      <Text style={[styles.subsectionTitle, { color: labelColor }]}>Warnings</Text>
      {warnings.map((warning) => (
        <Text key={warning} style={{ color: semanticColors.warning }}>{`• ${warning}`}</Text>
      ))}
    </View>
  );
}

export function EvidenceSection({
  run,
  items,
}: {
  run: RunInspectorDetail;
  items: RunEvidenceItem[];
}) {
  const colors = useSectionColors();

  return (
    <Section title="Evidence Pack" testID="activity-detail-evidence">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        <EvidenceMetaBlock run={run} itemCount={items.length} colors={colors} />
        {items.length === 0 ? (
          <EmptyState message="No evidence recorded" />
        ) : (
          items.map((item, index) => (
            <EvidenceCard
              key={item.id || `activity-evidence-${index}`}
              source={item}
              index={index + 1}
              testIDPrefix={`activity-evidence-${index}`}
            />
          ))
        )}
      </View>
    </Section>
  );
}

function CodeBlock({
  label,
  value,
  colors,
}: {
  label: string;
  value: unknown;
  colors: ReturnType<typeof useSectionColors>;
}) {
  return (
    <View style={styles.codeBlock}>
      <Text style={[styles.codeLabel, { color: colors.onSurfaceVariant }]}>{label}</Text>
      <Text style={[typography.monoSM, { color: colors.onSurface }]}>{formatJson(value)}</Text>
    </View>
  );
}

export function ToolActivitySection({ calls }: { calls: NormalizedToolCall[] }) {
  const colors = useSectionColors();

  return (
    <Section title="Tool Activity" testID="activity-detail-tool-calls">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        {calls.length === 0 ? (
          <EmptyState message="No tool calls recorded" />
        ) : (
          calls.map((call, index) => (
            <View key={`${call.toolName}-${index}`} style={[sharedStyles.subpanel, { borderColor: colors.outline }]}>
              <View style={sharedStyles.headerRow}>
                <Text style={[styles.toolName, { color: colors.primary }]}>{call.toolName}</Text>
                {call.status ? <MetaChip label={formatLabel(call.status)} tone="status" /> : null}
              </View>
              <View style={sharedStyles.metaRow}>
                <Text style={[typography.monoSM, { color: colors.onSurfaceVariant }]}>{`Latency: ${formatMs(call.latencyMs)}`}</Text>
                {call.idempotencyKey ? (
                  <Text style={[typography.monoSM, { color: colors.onSurfaceVariant }]}>{call.idempotencyKey}</Text>
                ) : null}
              </View>
              {call.input !== undefined ? <CodeBlock label="Input" value={call.input} colors={colors} /> : null}
              {call.output !== undefined ? <CodeBlock label="Output" value={call.output} colors={colors} /> : null}
            </View>
          ))
        )}
      </View>
    </Section>
  );
}

const styles = StyleSheet.create({
  subsection: {
    gap: spacing.xs,
  },
  subsectionTitle: {
    ...typography.labelMD,
  },
  toolName: {
    ...typography.headingMD,
    flex: 1,
  },
  codeBlock: {
    borderRadius: radius.md,
    backgroundColor: brandColors.surfaceVariant,
    padding: spacing.sm,
    gap: spacing.xs,
  },
  codeLabel: {
    ...typography.labelMD,
  },
});
