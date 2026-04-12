import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import type { AuditEvent, AuditOutcome } from '../../services/api.types';

interface AuditEventCardProps {
  event: AuditEvent;
  expanded?: boolean;
  onPress: () => void;
  testIDPrefix?: string;
}

function getOutcomeColor(outcome: AuditOutcome): string {
  const colors: Record<AuditOutcome, string> = {
    success: '#10B981',
    denied: '#EF4444',
    error: '#DC2626',
  };
  return colors[outcome] ?? '#6B7280';
}

function formatDetails(details: unknown): string {
  if (details === null || details === undefined) {
    return '—';
  }
  if (typeof details === 'string') {
    return details;
  }
  try {
    return JSON.stringify(details, null, 2);
  } catch {
    return String(details);
  }
}

export function AuditEventCard({
  event,
  expanded = false,
  onPress,
  testIDPrefix = 'audit-event',
}: AuditEventCardProps) {
  const { colors } = useTheme();
  const outcomeColor = getOutcomeColor(event.outcome);

  return (
    <Card testID={`${testIDPrefix}-card`} style={styles.card} onPress={onPress}>
      <Card.Content>
        <View style={styles.headerRow}>
          <View style={styles.headerText}>
            <Text
              testID={`${testIDPrefix}-title`}
              variant="titleSmall"
              style={{ color: colors.onSurface }}
            >
              {event.action}
            </Text>
            <Text
              testID={`${testIDPrefix}-subtitle`}
              variant="bodySmall"
              style={{ color: colors.onSurfaceVariant, marginTop: 4 }}
            >
              {event.actor_type} · {event.actor_id}
            </Text>
          </View>
          <View
            testID={`${testIDPrefix}-outcome-badge`}
            style={[styles.outcomeBadge, { backgroundColor: outcomeColor }]}
          >
            <Text style={styles.outcomeText}>{event.outcome}</Text>
          </View>
        </View>

        <Text
          testID={`${testIDPrefix}-created-at`}
          variant="bodySmall"
          style={{ color: colors.onSurfaceVariant, marginTop: 8 }}
        >
          {new Date(event.created_at).toLocaleString()}
        </Text>

        {expanded ? (
          <View style={styles.expandedContent}>
            <Text
              testID={`${testIDPrefix}-entity`}
              variant="bodySmall"
              style={{ color: colors.onSurface }}
            >
              {(event.entity_type ?? '—')} · {(event.entity_id ?? '—')}
            </Text>
            <Text
              testID={`${testIDPrefix}-trace-id`}
              variant="bodySmall"
              style={{ color: colors.onSurfaceVariant, marginTop: 4 }}
            >
              Trace: {event.trace_id ?? '—'}
            </Text>
            <Text
              testID={`${testIDPrefix}-details`}
              variant="bodySmall"
              style={[styles.detailsText, { color: colors.onSurfaceVariant }]}
            >
              {formatDetails(event.details)}
            </Text>
          </View>
        ) : null}
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
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: 12,
  },
  headerText: {
    flex: 1,
  },
  outcomeBadge: {
    borderRadius: 12,
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  outcomeText: {
    color: '#FFF',
    fontSize: 11,
    fontWeight: '700',
    textTransform: 'uppercase',
  },
  expandedContent: {
    marginTop: 12,
  },
  detailsText: {
    marginTop: 8,
    fontFamily: 'monospace',
  },
});
