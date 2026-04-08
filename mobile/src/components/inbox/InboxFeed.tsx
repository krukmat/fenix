import React from 'react';
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useRouter } from 'expo-router';
import { ApprovalCard } from '../approvals/ApprovalCard';
import { SignalCard } from '../signals/SignalCard';
import { resolveHandoffEntityContext, resolveWedgeHandoffPackageDestination, wedgeHref, wedgeHrefObject } from '../../utils/navigation';
import type { ApprovalRequest, HandoffPackage, Signal } from '../../services/api';

export type InboxFilter = 'all' | 'approval' | 'handoff' | 'signal';

export type InboxRenderableItem =
  | { type: 'approval'; id: string; approval: ApprovalRequest }
  | { type: 'handoff'; id: string; runId: string; handoff: HandoffPackage }
  | { type: 'signal'; id: string; signal: Signal };

const FILTER_CHIPS: { key: InboxFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'approval', label: 'Approvals' },
  { key: 'handoff', label: 'Handoffs' },
  { key: 'signal', label: 'Signals' },
];

function InboxHeader({ total, visible }: { total: number; visible: number }) {
  return (
    <View style={styles.header} testID="inbox-header">
      <Text style={styles.title}>Inbox</Text>
      <Text style={styles.subtitle}>Approvals, handoffs, and signals in one queue</Text>
      <Text style={styles.count} testID="inbox-total-count">{total} items</Text>
      <Text style={styles.visibleCount} testID="inbox-visible-count">{visible} shown</Text>
    </View>
  );
}

export function InboxLoading() {
  return (
    <View style={styles.state} testID="inbox-loading">
      <ActivityIndicator size="large" />
      <Text style={styles.stateTitle}>Loading inbox…</Text>
    </View>
  );
}

export function InboxError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <View style={styles.state} testID="inbox-error">
      <Text style={styles.stateTitle}>Inbox unavailable</Text>
      <Text style={styles.stateBody}>{message}</Text>
      <Pressable onPress={onRetry} testID="inbox-retry" style={styles.retryButton}>
        <Text style={styles.retryText}>Retry</Text>
      </Pressable>
    </View>
  );
}

export function InboxEmpty({ filter, onFilterChange }: { filter: InboxFilter; onFilterChange: (next: InboxFilter) => void }) {
  return (
    <View style={styles.container}>
      <InboxHeader total={0} visible={0} />
      <FilterChips value={filter} onChange={onFilterChange} />
      <View style={styles.state} testID="inbox-empty">
        <Text style={styles.stateTitle}>Nothing pending</Text>
        <Text style={styles.stateBody}>Approvals, handoffs, and signals will appear here.</Text>
      </View>
    </View>
  );
}

function FilterChips({ value, onChange }: { value: InboxFilter; onChange: (next: InboxFilter) => void }) {
  return (
    <View style={styles.chipsRow} testID="inbox-filter-chips">
      {FILTER_CHIPS.map((chip) => {
        const selected = chip.key === value;
        return (
          <Pressable
            key={chip.key}
            onPress={() => onChange(chip.key)}
            testID={`inbox-chip-${chip.key}`}
            style={[styles.chip, selected ? styles.chipSelected : styles.chipIdle]}
          >
            <Text style={[styles.chipText, selected ? styles.chipTextSelected : styles.chipTextIdle]}>
              {chip.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

function HandoffCard({
  handoff,
  runId,
  onPress,
}: {
  handoff: HandoffPackage;
  runId: string;
  onPress: () => void;
}) {
  const { entityType, entityId } = resolveHandoffEntityContext(handoff);

  return (
    <Pressable style={styles.handoffCard} onPress={onPress} testID={`inbox-handoff-${runId}`}>
      <Text style={styles.handoffEyebrow}>Handoff</Text>
      <Text style={styles.handoffReason} testID={`inbox-handoff-${runId}-reason`}>{handoff.reason}</Text>
      {entityType && entityId ? (
        <Text style={styles.handoffMeta} testID={`inbox-handoff-${runId}-entity`}>
          {entityType} · {entityId}
        </Text>
      ) : null}
      <Text style={styles.handoffMeta} testID={`inbox-handoff-${runId}-evidence`}>
        {handoff.evidence_count} evidence item{handoff.evidence_count === 1 ? '' : 's'}
      </Text>
    </Pressable>
  );
}

export function InboxBody({
  items,
  totalItems,
  filter,
  onFilterChange,
  actionError,
  onApprove,
  onReject,
  approvalsPending,
}: {
  items: InboxRenderableItem[];
  totalItems: number;
  filter: InboxFilter;
  onFilterChange: (next: InboxFilter) => void;
  actionError: string | null;
  onApprove: (id: string) => void;
  onReject: (id: string, reason: string) => void;
  approvalsPending: boolean;
}) {
  const router = useRouter();

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content} testID="inbox-screen">
      <InboxHeader total={totalItems} visible={items.length} />
      <FilterChips value={filter} onChange={onFilterChange} />
      {actionError ? (
        <View style={styles.inlineError} testID="inbox-approval-action-error">
          <Text style={styles.inlineErrorText}>{actionError}</Text>
        </View>
      ) : null}
      {items.map((item, index) => (
        <View
          key={`${item.type}-${item.id}`}
          testID={`inbox-item-${index}`}
          accessibilityLabel={`${item.type}:${item.id}`}
        >
          {item.type === 'approval' ? (
            <ApprovalCard
              approval={item.approval}
              onApprove={onApprove}
              onReject={onReject}
              testIDPrefix={`inbox-approval-${item.approval.id}`}
              disabled={approvalsPending}
            />
          ) : null}
          {item.type === 'handoff' ? (
            <HandoffCard
              handoff={item.handoff}
              runId={item.runId}
              onPress={() => router.push(wedgeHref(resolveWedgeHandoffPackageDestination(item.handoff, item.runId)))}
            />
          ) : null}
          {item.type === 'signal' ? (
            <SignalCard
              signal={item.signal}
              onDismiss={() => {}}
              onPress={(signal) => router.push(wedgeHrefObject('/(tabs)/home/signal/[id]', {
                id: signal.id,
                entity_type: signal.entity_type,
                entity_id: signal.entity_id,
              }))}
              testIDPrefix={`inbox-signal-${item.signal.id}`}
            />
          ) : null}
        </View>
      ))}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#FFFFFF' },
  content: { paddingVertical: 20 },
  header: { paddingHorizontal: 20, paddingBottom: 16, gap: 4 },
  title: { fontSize: 28, fontWeight: '700', color: '#111827' },
  subtitle: { fontSize: 14, color: '#6B7280' },
  count: { fontSize: 13, fontWeight: '600', color: '#1F2937' },
  visibleCount: { fontSize: 13, color: '#6B7280' },
  chipsRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, paddingHorizontal: 16, paddingBottom: 16 },
  chip: { borderRadius: 999, paddingHorizontal: 14, paddingVertical: 8, borderWidth: 1 },
  chipSelected: { backgroundColor: '#111827', borderColor: '#111827' },
  chipIdle: { backgroundColor: '#FFFFFF', borderColor: '#D1D5DB' },
  chipText: { fontSize: 13, fontWeight: '600' },
  chipTextSelected: { color: '#FFFFFF' },
  chipTextIdle: { color: '#111827' },
  inlineError: {
    marginHorizontal: 16,
    marginBottom: 12,
    paddingHorizontal: 14,
    paddingVertical: 10,
    borderRadius: 10,
    backgroundColor: '#FEF2F2',
    borderWidth: 1,
    borderColor: '#FCA5A5',
  },
  inlineErrorText: { color: '#991B1B', fontSize: 13, fontWeight: '500' },
  state: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  stateTitle: { fontSize: 20, fontWeight: '600', color: '#111827', marginTop: 12, marginBottom: 8 },
  stateBody: { fontSize: 14, color: '#6B7280', textAlign: 'center' },
  retryButton: { marginTop: 16, backgroundColor: '#111827', borderRadius: 999, paddingHorizontal: 16, paddingVertical: 10 },
  retryText: { color: '#FFFFFF', fontWeight: '600' },
  handoffCard: {
    marginHorizontal: 16,
    marginBottom: 8,
    padding: 16,
    borderRadius: 12,
    backgroundColor: '#FEF3C7',
    borderWidth: 1,
    borderColor: '#F59E0B',
  },
  handoffEyebrow: { fontSize: 12, fontWeight: '700', color: '#92400E', textTransform: 'uppercase', marginBottom: 6 },
  handoffReason: { fontSize: 16, fontWeight: '600', color: '#111827', marginBottom: 6 },
  handoffMeta: { fontSize: 13, color: '#4B5563' },
});
