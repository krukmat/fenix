import React, { useMemo, useState } from 'react';
import { TouchableOpacity, View, StyleSheet } from 'react-native';
import { Card, Text, Chip } from 'react-native-paper';
import type { EvidenceSource } from '../../services/sse';

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
    <Card testID={`${testIDPrefix}-card-container`} style={styles.card}>
      <TouchableOpacity testID={`${testIDPrefix}-card`} onPress={() => setExpanded((v) => !v)}>
        <Card.Content>
          <View style={styles.header}>
            <Text variant="titleSmall">{collapsedTitle}</Text>
            <Chip compact testID={`${testIDPrefix}-score`}>{source.score.toFixed(2)}</Chip>
          </View>
          <Text variant="bodySmall" testID={`${testIDPrefix}-snippet`}>
            {expanded ? source.snippet : truncate(source.snippet)}
          </Text>
          <Text variant="labelSmall">{timestamp}</Text>
        </Card.Content>
      </TouchableOpacity>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: { marginBottom: 8 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: 8 },
});
