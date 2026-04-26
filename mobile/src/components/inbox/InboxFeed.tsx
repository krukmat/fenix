import React from 'react';
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useRouter } from 'expo-router';
import { ApprovalCard } from '../approvals/ApprovalCard';
import { SignalCard } from '../signals/SignalCard';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import { resolveHandoffEntityContext, resolveWedgeHandoffPackageDestination, wedgeHref, wedgeHrefObject } from '../../utils/navigation';
import type { AgentRun, ApprovalRequest, HandoffPackage, Signal } from '../../services/api';

export type InboxFilter = 'all' | 'approval' | 'handoff' | 'signal' | 'rejected';

export type InboxRenderableItem =
  | { type: 'approval'; id: string; approval: ApprovalRequest }
  | { type: 'handoff'; id: string; runId: string; handoff: HandoffPackage }
  | { type: 'signal'; id: string; signal: Signal }
  | { type: 'rejected'; id: string; run: AgentRun };

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

function RejectedCard({ run, onPress }: { run: AgentRun; onPress: () => void }) {
  return (
    <Pressable style={styles.rejectedCard} onPress={onPress} testID={`inbox-rejected-${run.id}`}>
      <Text style={styles.rejectedEyebrow}>Rejected</Text>
      <Text style={styles.rejectedReason} testID={`inbox-rejected-${run.id}-reason`}>
        {run.rejection_reason ?? 'Policy blocked this run'}
      </Text>
      {run.entity_type && run.entity_id ? (
        <Text style={styles.rejectedMeta} testID={`inbox-rejected-${run.id}-entity`}>
          {run.entity_type} · {run.entity_id}
        </Text>
      ) : null}
      <Text style={styles.rejectedMeta} testID={`inbox-rejected-${run.id}-status`}>
        {run.status.replace(/_/g, ' ')}
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
          style={styles.item}
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
          {item.type === 'rejected' ? (
            <RejectedCard
              run={item.run}
              onPress={() => router.push(wedgeHref(`/activity/${item.run.id}`))}
            />
          ) : null}
        </View>
      ))}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: brandColors.background },
  content: { paddingVertical: spacing.lg },
  item: { marginBottom: spacing.sm },
  header: { paddingHorizontal: spacing.lg, paddingBottom: spacing.base, gap: spacing.xs },
  title: { ...typography.headingLG, color: brandColors.onBackground },
  subtitle: { fontSize: 14, color: brandColors.onSurfaceVariant },
  count: { fontSize: 13, fontWeight: '600', color: brandColors.onSurface },
  visibleCount: { fontSize: 13, color: brandColors.onSurfaceVariant },
  chipsRow: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.sm, paddingHorizontal: spacing.base, paddingBottom: spacing.base },
  chip: { borderRadius: radius.full, paddingHorizontal: spacing.base, paddingVertical: spacing.sm, borderWidth: 1 },
  chipSelected: { backgroundColor: brandColors.primary, borderColor: brandColors.primary },
  chipIdle: { backgroundColor: brandColors.surfaceVariant, borderColor: brandColors.outline },
  chipText: typography.labelMD,
  chipTextSelected: { color: brandColors.onPrimary },
  chipTextIdle: { color: brandColors.onSurface },
  inlineError: {
    marginHorizontal: spacing.base,
    marginBottom: spacing.md,
    paddingHorizontal: spacing.base,
    paddingVertical: spacing.md,
    borderRadius: radius.md,
    backgroundColor: brandColors.errorContainer,
    borderWidth: 1,
    borderColor: brandColors.error,
  },
  inlineErrorText: { color: brandColors.onErrorContainer, fontSize: 13, fontWeight: '500' },
  state: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: spacing.xl },
  stateTitle: { ...typography.headingMD, color: brandColors.onBackground, marginTop: spacing.md, marginBottom: spacing.sm },
  stateBody: { fontSize: 14, color: brandColors.onSurfaceVariant, textAlign: 'center' },
  retryButton: { marginTop: spacing.base, backgroundColor: brandColors.primary, borderRadius: radius.full, paddingHorizontal: spacing.base, paddingVertical: spacing.md },
  retryText: { color: brandColors.onPrimary, fontWeight: '600' },
  handoffCard: {
    marginHorizontal: spacing.base, marginBottom: spacing.sm, padding: spacing.base, borderRadius: radius.md,
    backgroundColor: brandColors.secondaryContainer,
    borderWidth: 1, borderColor: brandColors.secondary, borderLeftWidth: 3,
  },
  handoffEyebrow: { ...typography.eyebrow, color: brandColors.onSecondaryContainer, marginBottom: spacing.sm },
  handoffReason: { fontSize: 16, fontWeight: '600', color: brandColors.onBackground, marginBottom: spacing.sm },
  handoffMeta: { fontSize: 13, color: brandColors.onSecondaryContainer },
  rejectedCard: {
    marginHorizontal: spacing.base, marginBottom: spacing.sm, padding: spacing.base, borderRadius: radius.md,
    backgroundColor: brandColors.errorContainer,
    borderWidth: 1, borderColor: brandColors.error, borderLeftWidth: 3,
  },
  rejectedEyebrow: { ...typography.eyebrow, color: brandColors.onErrorContainer, marginBottom: spacing.sm },
  rejectedReason: { fontSize: 16, fontWeight: '600', color: brandColors.onBackground, marginBottom: spacing.sm },
  rejectedMeta: { fontSize: 13, color: brandColors.onErrorContainer },
});
