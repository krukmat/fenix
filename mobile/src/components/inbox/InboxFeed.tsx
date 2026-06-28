import React from 'react';
import { Pressable, ScrollView, Text, View } from 'react-native';
import { InboxListItem, type InboxRenderableItem } from './InboxListItem';
import { styles } from './InboxStyles';

export { InboxError, InboxLoading } from './InboxStateBlocks';
export type { InboxRenderableItem } from './InboxListItem';

export type InboxFilter = 'all' | 'approval' | 'handoff' | 'signal' | 'rejected';

const FILTER_CHIPS: { key: InboxFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'approval', label: 'Approvals' },
  { key: 'handoff', label: 'Handoffs' },
  { key: 'signal', label: 'Signals' },
  { key: 'rejected', label: 'Rejected' },
];

function InboxHeader({ total, visible }: { total: number; visible: number }) {
  return (
    <View style={styles.header} testID="inbox-header">
      <Text style={styles.title}>Inbox</Text>
      <Text style={styles.subtitle}>Approvals, handoffs, signals, and rejections in one queue</Text>
      <Text style={styles.count} testID="inbox-total-count">{total} items</Text>
      <Text style={styles.visibleCount} testID="inbox-visible-count">{visible} shown</Text>
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
        <InboxListItem
          key={`${item.type}-${item.id}`}
          item={item}
          index={index}
          onApprove={onApprove}
          onReject={onReject}
          approvalsPending={approvalsPending}
        />
      ))}
    </ScrollView>
  );
}
