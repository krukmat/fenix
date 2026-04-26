import React, { useMemo, useState } from 'react';
import { TouchableOpacity, View, StyleSheet } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import type { EvidenceSource } from '../../services/sse';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

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
    borderRadius: radius.full,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: brandColors.primaryContainer,
  },
  scoreBadgeText: {
    color: brandColors.onPrimaryContainer,
    ...typography.labelMD,
  },
});
