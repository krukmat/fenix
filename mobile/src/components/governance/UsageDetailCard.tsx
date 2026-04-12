// Wave 1 — governance_mobile_enhancement_plan: rich usage event card
// Replaces the barebones metric_name+value row in governance/index.tsx.

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import type { UsageEvent } from '../../services/api.types';

interface UsageDetailCardProps {
  event: UsageEvent;
  testIDPrefix?: string;
  onPress?: () => void;
}

function formatCost(cost: number | undefined): string {
  if (cost === undefined) return '—';
  return `€${cost.toFixed(5)}`;
}

function formatLatency(ms: number | undefined): string {
  if (ms === undefined) return '—';
  return `${ms} ms`;
}

function formatTimestamp(iso: string): string {
  return new Date(iso).toLocaleString();
}

export function UsageDetailCard({ event, testIDPrefix = 'udc', onPress }: UsageDetailCardProps) {
  const { colors } = useTheme();

  return (
    <Card
      testID={`${testIDPrefix}-card`}
      style={styles.card}
      onPress={onPress}
    >
      <Card.Content>
        {/* Row 1: actor type badge + tool name */}
        <View style={styles.headerRow}>
          <View style={[styles.badge, { backgroundColor: ((colors as unknown) as Record<string, string>).primaryContainer ?? '#E8DEF8' }]}>
            <Text
              testID={`${testIDPrefix}-actor-type`}
              style={[styles.badgeText, { color: ((colors as unknown) as Record<string, string>).onPrimaryContainer ?? '#21005D' }]}
            >
              {event.actorType ?? '—'}
            </Text>
          </View>
          <Text
            testID={`${testIDPrefix}-tool-name`}
            variant="titleSmall"
            style={[styles.toolName, { color: colors.onSurface }]}
          >
            {event.toolName ?? '—'}
          </Text>
        </View>

        {/* Row 2: model name */}
        <Text
          testID={`${testIDPrefix}-model-name`}
          variant="bodySmall"
          style={{ color: colors.onSurfaceVariant, marginBottom: 6 }}
        >
          {event.modelName ?? '—'}
        </Text>

        {/* Row 3: cost + latency + timestamp */}
        <View style={styles.metaRow}>
          <Text
            testID={`${testIDPrefix}-cost`}
            variant="bodySmall"
            style={{ color: colors.onSurface, fontWeight: '600' }}
        >
            {formatCost(event.estimatedCost)}
          </Text>
          <Text
            testID={`${testIDPrefix}-latency`}
            variant="bodySmall"
            style={{ color: colors.onSurfaceVariant }}
        >
            {formatLatency(event.latencyMs)}
          </Text>
          <Text
            testID={`${testIDPrefix}-timestamp`}
            variant="bodySmall"
            style={{ color: colors.onSurfaceVariant }}
        >
            {formatTimestamp(event.createdAt)}
          </Text>
        </View>
      </Card.Content>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: {
    marginBottom: 8,
    marginHorizontal: 16,
  },
  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
    gap: 8,
  },
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 4,
  },
  badgeText: {
    fontSize: 11,
    fontWeight: '700',
    textTransform: 'uppercase',
    letterSpacing: 0.4,
  },
  toolName: {
    flex: 1,
  },
  metaRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    flexWrap: 'wrap',
    gap: 4,
  },
});
