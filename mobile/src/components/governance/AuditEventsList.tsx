import React from 'react';
import { ActivityIndicator, FlatList, StyleSheet, View } from 'react-native';
import { AuditEventCard } from './AuditEventCard';
import { CenteredMessageState } from '../ui/ScreenState';
import type { AuditEvent } from '../../services/api.types';

interface AuditEventsListProps {
  backgroundColor: string;
  emptyColor: string;
  spinnerColor: string;
  events: AuditEvent[];
  expandedId: string | null;
  isFetching: boolean;
  hasMore: boolean;
  onToggleExpanded: (id: string) => void;
  onReachEnd: () => void;
}

export function AuditEventsList({
  backgroundColor,
  emptyColor,
  spinnerColor,
  events,
  expandedId,
  isFetching,
  hasMore,
  onToggleExpanded,
  onReachEnd,
}: AuditEventsListProps) {
  if (events.length === 0) {
    return (
      <CenteredMessageState
        testID="audit-empty"
        message="No audit events found"
        messageColor={emptyColor}
        backgroundColor="transparent"
      />
    );
  }

  return (
    <FlatList
      testID="audit-list"
      data={events}
      keyExtractor={(item) => item.id}
      renderItem={({ item, index }) => (
        <AuditEventCard
          event={item}
          expanded={expandedId === item.id}
          onPress={() => onToggleExpanded(item.id)}
          testIDPrefix={`audit-event-${index}`}
        />
      )}
      onEndReached={() => {
        if (!isFetching && hasMore) {
          onReachEnd();
        }
      }}
      onEndReachedThreshold={0.4}
      ListFooterComponent={
        isFetching ? (
          <View style={styles.footer}>
            <ActivityIndicator size="small" color={spinnerColor} />
          </View>
        ) : null
      }
      contentContainerStyle={styles.listContent}
      style={[styles.list, { backgroundColor }]}
    />
  );
}

const styles = StyleSheet.create({
  list: {
    flex: 1,
  },
  listContent: {
    paddingTop: 8,
    paddingBottom: 16,
  },
  footer: {
    paddingVertical: 16,
  },
});
