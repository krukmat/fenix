import React, { useMemo, useState } from 'react';
import { TouchableOpacity, View, StyleSheet } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import type { EvidenceSource } from '../../services/sse';
import { brandColors } from '../../theme/colors';
import { chipShape, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import { formatLabel } from '../runs/runInspector.model';

interface EvidenceCardProps {
  source: EvidenceSource;
  index: number;
  testIDPrefix?: string;
}

function truncate(value: string, len = 80): string {
  if (value.length <= len) return value;
  return `${value.slice(0, len)}…`;
}

export function EvidenceCard({ source, index, testIDPrefix = 'evidence' }: EvidenceCardProps) {
  const [expanded, setExpanded] = useState(false);
  const { colors } = useTheme();

  const collapsedTitle = useMemo(() => {
    const base = source.title?.trim() || source.snippet;
    return `[${index}] ${truncate(base)}`;
  }, [index, source.snippet, source.title]);

  const timestamp = useMemo(() => {
    const d = new Date(source.timestamp);
    if (Number.isNaN(d.getTime())) return source.timestamp;
    return d.toISOString();
  }, [source.timestamp]);

  const retrievalMethodLabel = useMemo(
    () => (source.retrieval_method ? formatLabel(source.retrieval_method) : null),
    [source.retrieval_method],
  );
  const hasTrustFields = Boolean(retrievalMethodLabel || source.pii_redacted || source.knowledge_item_id);

  return (
    <Card testID={testIDPrefix} style={styles.card}>
      <TouchableOpacity testID={`${testIDPrefix}-card`} onPress={() => setExpanded((v) => !v)}>
        <Card.Content>
          <View style={styles.header}>
            <Text variant="titleSmall" style={{ color: colors.onSurface }}>{collapsedTitle}</Text>
            <View testID={`${testIDPrefix}-score`} style={styles.scoreBadge}>
              <Text style={styles.scoreBadgeText}>{source.score.toFixed(2)}</Text>
            </View>
          </View>
          <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }} testID={`${testIDPrefix}-snippet`}>
            {expanded ? source.snippet : truncate(source.snippet)}
          </Text>
          <Text variant="labelSmall" style={[typography.monoSM, { color: colors.onSurfaceVariant }]}>{timestamp}</Text>
          {expanded && hasTrustFields && (
            <View style={styles.metaSection} testID={`${testIDPrefix}-meta`}>
              {retrievalMethodLabel ? (
                <View style={styles.methodChip} testID={`${testIDPrefix}-method`}>
                  <Text style={styles.methodChipText}>{retrievalMethodLabel}</Text>
                </View>
              ) : null}
              {source.pii_redacted ? (
                <View style={styles.flag} testID={`${testIDPrefix}-pii`}>
                  <Text style={[styles.flagText, { color: colors.onSurface }]}>PII redacted</Text>
                </View>
              ) : null}
              {source.knowledge_item_id ? (
                <Text
                  variant="labelSmall"
                  testID={`${testIDPrefix}-knowledge-item`}
                  style={[typography.monoSM, { color: colors.onSurfaceVariant }]}
                >
                  {`Knowledge item: ${source.knowledge_item_id}`}
                </Text>
              ) : null}
            </View>
          )}
        </Card.Content>
      </TouchableOpacity>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: { marginBottom: spacing.sm },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: spacing.sm },
  scoreBadge: {
    minWidth: 56,
    ...chipShape,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: brandColors.primaryContainer,
  },
  scoreBadgeText: {
    color: brandColors.onPrimaryContainer,
    ...typography.labelMD,
  },
  metaSection: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    alignItems: 'center',
    gap: spacing.xs,
    marginTop: spacing.sm,
  },
  methodChip: {
    ...chipShape,
    backgroundColor: brandColors.surfaceVariant,
  },
  methodChipText: {
    color: brandColors.onSurface,
    ...typography.labelMD,
  },
  flag: {
    ...chipShape,
    borderWidth: 1,
    borderColor: brandColors.primary,
  },
  flagText: {
    ...typography.labelMD,
  },
});
