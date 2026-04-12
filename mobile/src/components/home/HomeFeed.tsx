// UC-A5/A6: Unified feed — signals + approvals with chip filter
// FR-300 (Home), FR-071 (approvals badge)

import React, { useState, useCallback } from 'react';
import { View, FlatList, RefreshControl, StyleSheet, Pressable } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { SignalCard } from '../signals/SignalCard';
import { ApprovalCard } from '../approvals/ApprovalCard';
import type { Signal, ApprovalRequest } from '../../services/api';

type FeedFilter = 'all' | 'signals' | 'approvals';

interface HomeFeedProps {
  signals: Signal[];
  approvals: ApprovalRequest[];
  loadingSignals: boolean;
  loadingApprovals: boolean;
  onRefresh: () => void;
  onDismissSignal: (id: string) => void;
  onApprove: (id: string) => void;
  onReject: (id: string, reason: string) => void;
  onSignalPress?: (signal: Signal) => void;
  pendingApprovalCount?: number;
  testIDPrefix?: string;
}

type FeedItem =
  | { kind: 'signal'; data: Signal }
  | { kind: 'approval'; data: ApprovalRequest };

function buildFeedItems(signals: Signal[], approvals: ApprovalRequest[], filter: FeedFilter): FeedItem[] {
  const signalItems: FeedItem[] = filter !== 'approvals' ? signals.map((s) => ({ kind: 'signal', data: s })) : [];
  const approvalItems: FeedItem[] = filter !== 'signals' ? approvals.map((a) => ({ kind: 'approval', data: a })) : [];
  return [...approvalItems, ...signalItems];
}

function chipLabel(f: FeedFilter, pendingApprovalCount: number): string {
  if (f === 'approvals' && pendingApprovalCount > 0) return `Approvals (${pendingApprovalCount})`;
  return f.charAt(0).toUpperCase() + f.slice(1);
}

export function HomeFeed({
  signals, approvals, loadingSignals, loadingApprovals, onRefresh,
  onDismissSignal, onApprove, onReject, onSignalPress,
  pendingApprovalCount = 0, testIDPrefix = 'home-feed',
}: HomeFeedProps) {
  const theme = useTheme();
  const [filter, setFilter] = useState<FeedFilter>('all');
  const isRefreshing = loadingSignals || loadingApprovals;
  const items = buildFeedItems(signals, approvals, filter);

  const renderItem = useCallback(
    ({ item }: { item: FeedItem }) => {
      if (item.kind === 'signal') {
        return (
          <SignalCard signal={item.data} onDismiss={onDismissSignal} onPress={onSignalPress}
            testIDPrefix={`${testIDPrefix}-signal-${item.data.id}`} />
        );
      }
      return (
        <ApprovalCard approval={item.data} onApprove={onApprove} onReject={onReject}
          testIDPrefix={`${testIDPrefix}-approval-${item.data.id}`} />
      );
    },
    [onDismissSignal, onSignalPress, onApprove, onReject, testIDPrefix]
  );

  const ListHeader = (
    <View style={styles.chipRow} testID={`${testIDPrefix}-chips`}>
      {(['all', 'signals', 'approvals'] as FeedFilter[]).map((f) => (
        <Pressable
          key={f}
          onPress={() => setFilter(f)}
          style={[styles.chip, filter === f ? styles.chipSelected : styles.chipIdle]}
          testID={`${testIDPrefix}-chip-${f}`}
        >
          <Text style={[styles.chipText, filter === f ? styles.chipTextSelected : styles.chipTextIdle]}>
            {chipLabel(f, pendingApprovalCount)}
          </Text>
        </Pressable>
      ))}
    </View>
  );

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]} testID={testIDPrefix}>
      <FlatList data={items} keyExtractor={(item) => `${item.kind}-${item.data.id}`}
        renderItem={renderItem} ListHeaderComponent={() => ListHeader}
        ListEmptyComponent={() => (
          <Text variant="bodyMedium" style={[styles.empty, { color: theme.colors.onSurfaceVariant }]}
            testID={`${testIDPrefix}-empty`}>
            {isRefreshing ? 'Loading…' : 'No items'}
          </Text>
        )}
        refreshControl={<RefreshControl refreshing={isRefreshing} onRefresh={onRefresh} tintColor={theme.colors.primary} />}
        contentContainerStyle={styles.listContent} showsVerticalScrollIndicator={false}
        testID={`${testIDPrefix}-flatlist`} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  chipRow: { flexDirection: 'row', gap: 8, paddingHorizontal: 16, paddingVertical: 12 },
  chip: { borderRadius: 999, paddingHorizontal: 14, paddingVertical: 8, borderWidth: 1 },
  chipSelected: { backgroundColor: '#111827', borderColor: '#111827' },
  chipIdle: { backgroundColor: '#FFFFFF', borderColor: '#D1D5DB' },
  chipText: { fontSize: 13, fontWeight: '600' },
  chipTextSelected: { color: '#FFFFFF' },
  chipTextIdle: { color: '#111827' },
  listContent: { paddingBottom: 24 },
  empty: { textAlign: 'center', marginTop: 48 },
});
