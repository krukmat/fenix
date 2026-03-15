// Task Mobile P1.3 — Unified feed: Signals + Approvals with chip filter

import React, { useState, useCallback } from 'react';
import { View, FlatList, RefreshControl, StyleSheet } from 'react-native';
import { Text, Chip, useTheme } from 'react-native-paper';
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
  onDeny: (id: string, reason: string) => void;
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
  return [...approvalItems, ...signalItems]; // approvals first
}

export function HomeFeed({
  signals,
  approvals,
  loadingSignals,
  loadingApprovals,
  onRefresh,
  onDismissSignal,
  onApprove,
  onDeny,
  onSignalPress,
  pendingApprovalCount = 0,
  testIDPrefix = 'home-feed',
}: HomeFeedProps) {
  const theme = useTheme();
  const [filter, setFilter] = useState<FeedFilter>('all');

  const isRefreshing = loadingSignals || loadingApprovals;
  const items = buildFeedItems(signals, approvals, filter);

  const renderItem = useCallback(
    ({ item }: { item: FeedItem }) => {
      if (item.kind === 'signal') {
        return (
          <SignalCard
            signal={item.data}
            onDismiss={onDismissSignal}
            onPress={onSignalPress}
            testIDPrefix={`${testIDPrefix}-signal-${item.data.id}`}
          />
        );
      }
      return (
        <ApprovalCard
          approval={item.data}
          onApprove={onApprove}
          onDeny={onDeny}
          testIDPrefix={`${testIDPrefix}-approval-${item.data.id}`}
        />
      );
    },
    [onDismissSignal, onSignalPress, onApprove, onDeny, testIDPrefix]
  );

  const FilterChips = (
    <View style={styles.chipRow} testID={`${testIDPrefix}-chips`}>
      {(['all', 'signals', 'approvals'] as FeedFilter[]).map((f) => (
        <Chip
          key={f}
          selected={filter === f}
          onPress={() => setFilter(f)}
          style={styles.chip}
          testID={`${testIDPrefix}-chip-${f}`}
        >
          {f === 'approvals' && pendingApprovalCount > 0
            ? `Approvals (${pendingApprovalCount})`
            : f.charAt(0).toUpperCase() + f.slice(1)}
        </Chip>
      ))}
    </View>
  );

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]} testID={testIDPrefix}>
      <FlatList
        data={items}
        keyExtractor={(item) => `${item.kind}-${item.data.id}`}
        renderItem={renderItem}
        ListHeaderComponent={() => FilterChips}
        ListEmptyComponent={() => (
          <Text
            variant="bodyMedium"
            style={[styles.empty, { color: theme.colors.onSurfaceVariant }]}
            testID={`${testIDPrefix}-empty`}
          >
            {isRefreshing ? 'Loading…' : 'No items'}
          </Text>
        )}
        refreshControl={
          <RefreshControl refreshing={isRefreshing} onRefresh={onRefresh} tintColor={theme.colors.primary} />
        }
        contentContainerStyle={styles.listContent}
        showsVerticalScrollIndicator={false}
        testID={`${testIDPrefix}-flatlist`}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  chipRow: { flexDirection: 'row', gap: 8, paddingHorizontal: 16, paddingVertical: 12 },
  chip: { height: 32 },
  listContent: { paddingBottom: 24 },
  empty: { textAlign: 'center', marginTop: 48 },
});
