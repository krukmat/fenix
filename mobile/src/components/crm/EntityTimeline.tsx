// Task 4.3 — Reusable Entity Timeline Component

import React from 'react';
import { View, Text, StyleSheet, FlatList } from 'react-native';
import { useTheme } from 'react-native-paper';
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
      return '#10B981'; // Green
    case 'status_change':
      return '#F59E0B'; // Amber
    case 'created':
      return '#3B82F6'; // Blue
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
    padding: 40,
  },
  emptyText: {
    fontSize: 14,
  },
  listContent: {
    padding: 16,
  },
  timelineItem: {
    flexDirection: 'row',
    marginBottom: 16,
  },
  timelineIndicator: {
    width: 32,
    height: 32,
    borderRadius: 16,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  timelineIcon: {
    fontSize: 14,
  },
  timelineContent: {
    flex: 1,
    padding: 12,
    borderRadius: 8,
  },
  eventTitle: {
    fontSize: 14,
    fontWeight: '500',
    marginBottom: 4,
  },
  eventDescription: {
    fontSize: 13,
    marginBottom: 8,
  },
  eventMeta: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  eventTimestamp: {
    fontSize: 12,
  },
  eventUser: {
    fontSize: 12,
  },
});

export default EntityTimeline;
