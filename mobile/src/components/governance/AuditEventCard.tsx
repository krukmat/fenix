import React from 'react';
import { View, StyleSheet, Pressable } from 'react-native';
import { useRouter } from 'expo-router';
import { Card, Text, useTheme } from 'react-native-paper';
import type { AuditEvent, AuditOutcome } from '../../services/api.types';
import { brandColors } from '../../theme/colors';
import { getAgentStatusColor } from '../../theme/semantic';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import { wedgeHrefObject } from '../../utils/navigation';

interface AuditEventCardProps {
  event: AuditEvent;
  expanded?: boolean;
  onPress: () => void;
  testIDPrefix?: string;
}

function getOutcomeColor(outcome: AuditOutcome): string {
  return getAgentStatusColor(outcome);
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

function TraceIdLink({
  traceId,
  testIDPrefix,
  colors,
}: {
  traceId: string | undefined;
  testIDPrefix: string;
  colors: { primary: string; onSurfaceVariant: string };
}) {
  const router = useRouter();

  if (!traceId) {
    return (
      <Text
        testID={`${testIDPrefix}-trace-id`}
        variant="bodySmall"
        style={[typography.monoSM, { color: colors.onSurfaceVariant, marginTop: 4 }]}
      >
        Trace: —
      </Text>
    );
  }

  return (
    <Pressable onPress={() => router.push(wedgeHrefObject('/governance/audit', { trace_id: traceId }))}>
      <Text
        testID={`${testIDPrefix}-trace-id`}
        variant="bodySmall"
        style={[typography.monoSM, { color: colors.primary, marginTop: 4 }]}
      >
        Trace: {traceId}
      </Text>
    </Pressable>
  );
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
    <Card
      testID={`${testIDPrefix}-card`}
      style={[styles.card, { borderLeftWidth: 2, borderLeftColor: outcomeColor }]}
      onPress={onPress}
    >
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
          style={[typography.monoSM, { color: colors.onSurfaceVariant, marginTop: 8 }]}
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
            <TraceIdLink traceId={event.trace_id} testIDPrefix={testIDPrefix} colors={colors} />
            <Text
              testID={`${testIDPrefix}-details`}
              variant="bodySmall"
              style={[styles.detailsText, typography.monoSM, { color: colors.onSurfaceVariant }]}
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
    marginBottom: spacing.sm,
    marginHorizontal: spacing.base,
  },
  headerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: spacing.md,
  },
  headerText: {
    flex: 1,
  },
  outcomeBadge: {
    borderRadius: radius.full,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
  },
  outcomeText: {
    color: brandColors.onError,
    ...typography.eyebrow,
    textTransform: 'uppercase',
  },
  expandedContent: {
    marginTop: spacing.md,
  },
  detailsText: {
    marginTop: spacing.sm,
  },
});
