import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import type { HandoffPackage } from '../../services/api.types';
import { radius, spacing } from '../../theme/spacing';
import { getAgentStatusColor } from '../../theme/semantic';
import { typography } from '../../theme/typography';
import {
  extractApprovalLinkage,
  formatLabel,
  formatMoney,
  formatMs,
  formatTimestamp,
  type RunInspectorDetail,
} from './runInspector.model';
import { EmptyState, MetaChip, Section, sharedStyles, SummaryMetric } from './runInspector.shared';

export function HighlightsSection({
  run,
  onOpenAuditTrail,
}: {
  run: RunInspectorDetail;
  onOpenAuditTrail: () => void;
}) {
  const { colors } = useTheme();
  const traceId = run.trace_id ?? run.traceId;
  const statusColor = getAgentStatusColor(run.status);

  return (
    <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]} testID="activity-detail-highlights">
      <View style={sharedStyles.headerRow}>
        <View style={styles.headerCopy}>
          <Text style={[styles.eyebrow, { color: colors.onSurfaceVariant }]}>Run Inspector</Text>
          <Text style={[styles.agentName, { color: colors.onSurface }]}>{run.agent_name}</Text>
        </View>
        <View testID="activity-detail-public-status" style={[styles.statusBadge, { backgroundColor: statusColor }]}>
          <Text style={styles.statusBadgeText}>{formatLabel(run.status)}</Text>
        </View>
      </View>

      <View style={sharedStyles.summaryGrid}>
        <SummaryMetric label="Trigger" value={run.trigger_type ? formatLabel(run.trigger_type) : '—'} />
        <SummaryMetric label="Latency" value={formatMs(run.latency_ms)} monospace />
        <SummaryMetric label="Estimated cost" value={formatMoney(run.cost_euros)} monospace />
        <SummaryMetric
          label="Entity"
          value={run.entity_type && run.entity_id ? `${run.entity_type} · ${run.entity_id}` : '—'}
        />
      </View>

      <View style={sharedStyles.metaRow}>
        {run.runtime_status ? <MetaChip label={`Runtime: ${formatLabel(run.runtime_status)}`} tone="status" /> : null}
        {run.triggered_by ? <MetaChip label={`Actor: ${run.triggered_by}`} /> : null}
      </View>

      {traceId ? (
        <View style={styles.traceBlock}>
          <Text style={[styles.traceLabel, { color: colors.onSurfaceVariant }]}>Trace ID</Text>
          <Text testID="activity-detail-trace-id" style={[typography.monoLG, { color: colors.onSurface }]}>
            {traceId}
          </Text>
          <Pressable testID="activity-detail-open-audit" onPress={onOpenAuditTrail} style={sharedStyles.inlineAction}>
            <Text style={[sharedStyles.inlineActionText, { color: colors.primary }]}>Open audit trail</Text>
          </Pressable>
        </View>
      ) : null}
    </View>
  );
}

export function RejectionSection({ reason }: { reason?: string }) {
  const { colors } = useTheme();
  if (!reason) return null;

  return (
    <Section title="Rejection Reason" testID="activity-detail-rejection-reason">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        <Text style={{ color: colors.error }}>{reason}</Text>
      </View>
    </Section>
  );
}

export function ApprovalLinkageSection({
  run,
  onOpenInbox,
}: {
  run: RunInspectorDetail;
  onOpenInbox: () => void;
}) {
  const { colors } = useTheme();
  if (run.status !== 'awaiting_approval') return null;

  const linkage = extractApprovalLinkage(run);

  return (
    <Section title="Approval Linkage" testID="activity-detail-approval-linkage">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        <Text style={{ color: colors.onSurface }}>
          This run is paused until a human records the governed decision.
        </Text>
        {linkage.action ? (
          <Text style={[styles.supportingText, { color: colors.onSurfaceVariant }]}>
            Suggested action: {formatLabel(linkage.action)}
          </Text>
        ) : null}
        {linkage.approvalId ? (
          <Text testID="activity-detail-approval-id" style={[typography.monoLG, { color: colors.onSurface, marginTop: spacing.sm }]}>
            {linkage.approvalId}
          </Text>
        ) : null}
        <Pressable testID="activity-detail-open-inbox" onPress={onOpenInbox} style={sharedStyles.inlineAction}>
          <Text style={[sharedStyles.inlineActionText, { color: colors.primary }]}>Open approvals queue</Text>
        </Pressable>
      </View>
    </Section>
  );
}

export function HandoffSection({
  run,
  handoff,
  handoffLoading,
  onOpenHandoffDestination,
}: {
  run: RunInspectorDetail;
  handoff?: HandoffPackage;
  handoffLoading: boolean;
  onOpenHandoffDestination: () => void;
}) {
  const { colors } = useTheme();
  if (run.status !== 'handed_off') return null;

  return (
    <Section title="Handoff Payload" testID="activity-detail-handoff-payload">
      <View style={[sharedStyles.panel, { backgroundColor: colors.surface }]}>
        {handoffLoading ? (
          <EmptyState message="Loading handoff payload..." />
        ) : handoff ? (
          <>
            <Text style={{ color: colors.onSurface }}>{handoff.reason}</Text>
            {handoff.conversation_context ? (
              <Text style={[styles.supportingText, { color: colors.onSurfaceVariant }]}>
                {handoff.conversation_context}
              </Text>
            ) : null}
            <View style={sharedStyles.metaRow}>
              <MetaChip label={`${handoff.evidence_count} evidence item${handoff.evidence_count === 1 ? '' : 's'}`} />
              {handoff.entity_type && handoff.entity_id ? (
                <MetaChip label={`${handoff.entity_type} · ${handoff.entity_id}`} />
              ) : null}
            </View>
            <Text style={[typography.monoSM, { color: colors.onSurfaceVariant, marginTop: spacing.sm }]}>
              {formatTimestamp(handoff.created_at)}
            </Text>
            <Pressable testID="activity-detail-open-handoff-target" onPress={onOpenHandoffDestination} style={sharedStyles.inlineAction}>
              <Text style={[sharedStyles.inlineActionText, { color: colors.primary }]}>Open handoff destination</Text>
            </Pressable>
          </>
        ) : (
          <EmptyState message="No handoff payload available" />
        )}
      </View>
    </Section>
  );
}

const styles = StyleSheet.create({
  headerCopy: {
    flex: 1,
  },
  eyebrow: {
    ...typography.eyebrow,
    marginBottom: spacing.xs,
  },
  agentName: {
    ...typography.headingLG,
  },
  statusBadge: {
    borderRadius: radius.full,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
  },
  statusBadgeText: {
    color: '#fff',
    ...typography.eyebrow,
  },
  traceBlock: {
    marginTop: spacing.sm,
    gap: spacing.xs,
  },
  traceLabel: {
    ...typography.labelMD,
  },
  supportingText: {
    marginTop: spacing.sm,
  },
});
