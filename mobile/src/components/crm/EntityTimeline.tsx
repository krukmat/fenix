// Task 4.3 — Reusable Entity Timeline Component

import React from 'react';
import { View, Text, StyleSheet, FlatList } from 'react-native';
import { useTheme } from 'react-native-paper';
import { semanticColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { ThemeColors } from '../../theme/types';

export interface TimelineEvent {
  id: string;
  type: 'note' | 'activity' | 'status_change' | 'created' | 'updated';
  title: string;
  description?: string;
  timestamp: string;
  userName?: string;
}

export interface EntityTimelineProps {
  events: TimelineEvent[];
  testIDPrefix?: string;
  emptyMessage?: string;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function getEventColor(type: TimelineEvent['type'], colors: ThemeColors): string {
  switch (type) {
    case 'note':
      return colors.primary;
    case 'activity':
      return semanticColors.success;
    case 'status_change':
      return semanticColors.warning;
    case 'created':
      return colors.primary;
    case 'updated':
      return colors.onSurfaceVariant;
    default:
      return colors.onSurfaceVariant;
  }
}

function getEventIcon(type: TimelineEvent['type']): string {
  switch (type) {
    case 'note':
      return '📝';
    case 'activity':
      return '📅';
    case 'status_change':
      return '🔄';
    case 'created':
      return '✨';
    case 'updated':
      return '✏️';
    default:
      return '•';
  }
}

function TimelineItem({ event, colors }: { event: TimelineEvent; colors: ThemeColors }) {
  return (
    <View style={styles.timelineItem}>
      <View style={[styles.timelineIndicator, { backgroundColor: getEventColor(event.type, colors) }]}>
        <Text style={styles.timelineIcon}>{getEventIcon(event.type)}</Text>
      </View>
      <View style={[styles.timelineContent, { backgroundColor: colors.surface }]}>
        <Text style={[styles.eventTitle, { color: colors.onSurface }]}>{event.title}</Text>
        {event.description && (
          <Text style={[styles.eventDescription, { color: colors.onSurfaceVariant }]}>
            {event.description}
          </Text>
        )}
        <View style={styles.eventMeta}>
          <Text style={[styles.eventTimestamp, { color: colors.onSurfaceVariant }]}>
            {event.timestamp}
          </Text>
          {event.userName && (
            <Text style={[styles.eventUser, { color: colors.onSurfaceVariant }]}>
              by {event.userName}
            </Text>
          )}
        </View>
      </View>
    </View>
  );
}

export function EntityTimeline({ events, testIDPrefix = 'entity-timeline', emptyMessage = 'No activity yet' }: EntityTimelineProps) {
  const colors = useThemeColors();

  if (!events || events.length === 0) {
    return (
      <View style={[styles.emptyContainer, { backgroundColor: colors.background }]} testID={`${testIDPrefix}-empty`}>
        <Text style={[styles.emptyText, { color: colors.onSurfaceVariant }]}>{emptyMessage}</Text>
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]} testID={`${testIDPrefix}-list`}>
      <FlatList
        data={events}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <TimelineItem event={item} colors={colors} />}
        contentContainerStyle={styles.listContent}
        scrollEnabled={false}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xxl,
  },
  emptyText: {
    fontSize: 14,
  },
  listContent: {
    padding: spacing.base,
  },
  timelineItem: {
    flexDirection: 'row',
    marginBottom: spacing.base,
  },
  timelineIndicator: {
    width: 32,
    height: 32,
    borderRadius: radius.full,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: spacing.md,
  },
  timelineIcon: {
    fontSize: 14,
  },
  timelineContent: {
    flex: 1,
    padding: spacing.md,
    borderRadius: radius.md,
  },
  eventTitle: {
    fontSize: 14,
    fontWeight: '500',
    marginBottom: spacing.xs,
  },
  eventDescription: {
    fontSize: 13,
    marginBottom: spacing.sm,
  },
  eventMeta: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  eventTimestamp: {
    ...typography.monoSM,
  },
  eventUser: {
    fontSize: typography.monoSM.fontSize,
  },
});

export default EntityTimeline;
