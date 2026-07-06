import React from 'react';
import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { Button } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { AgentActivitySection } from '../agents/AgentActivitySection';
import { EntitySignalsSection } from '../signals/EntitySignalsSection';
import { useAgentRuns } from '../../hooks/useWedge';
import { wedgeHref } from '../../utils/navigation';
import { brandColors, semanticColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { getAgentStatusColor } from '../../theme/semantic';
import { typography } from '../../theme/typography';
import type { AgentRun } from '../../services/api';
import type { ThemeColors } from '../../theme/types';
import { CaseStatusPath } from './CaseStatusPath';
import type { SupportCaseDetailData } from './supportCaseDetail.model';

export const CASE_STATUS_ORDER = ['open', 'in_progress', 'waiting', 'resolved', 'closed'];

function getPriorityColor(priority: string): string {
  if (priority === 'high') return brandColors.error;
  if (priority === 'medium') return semanticColors.warning;
  return semanticColors.success;
}

function toLabel(value: string): string {
  return value
    .split(/[_\s]+/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function SummaryCell({
  label,
  value,
  colors,
  testID,
  onPress,
}: {
  label: string;
  value: string;
  colors: ThemeColors;
  testID: string;
  onPress?: () => void;
}) {
  const content = (
    <>
      <Text style={[styles.summaryLabel, { color: colors.onSurfaceVariant }]}>{label}</Text>
      <Text style={[styles.summaryValue, { color: colors.onSurface }]}>{value}</Text>
    </>
  );

  return onPress ? (
    <TouchableOpacity style={[styles.summaryCell, { backgroundColor: colors.surface }]} onPress={onPress} testID={testID}>
      {content}
    </TouchableOpacity>
  ) : (
    <View style={[styles.summaryCell, { backgroundColor: colors.surface }]} testID={testID}>
      {content}
    </View>
  );
}

function HighlightsSection({
  caseData,
  colors,
  router,
}: {
  caseData: SupportCaseDetailData;
  colors: ThemeColors;
  router: ReturnType<typeof useRouter>;
}) {
  return (
    <View style={styles.section}>
      <View style={styles.sectionHeaderRow}>
        <Text style={[styles.sectionTitle, styles.sectionTitleNoMargin, { color: colors.onSurface }]}>Highlights</Text>
        <View style={[styles.statusChip, { backgroundColor: getAgentStatusColor(caseData.status) }]} testID="support-case-status-chip">
          <Text style={styles.statusChipText}>{toLabel(caseData.status)}</Text>
        </View>
      </View>

      <View style={styles.summaryGrid}>
        <SummaryCell
          label="Account"
          value={caseData.accountName || 'Not linked'}
          colors={colors}
          testID="support-case-account-highlight"
          onPress={caseData.accountId ? () => router.push(wedgeHref(`/sales/${caseData.accountId}`)) : undefined}
        />
        <SummaryCell
          label="Contact"
          value={caseData.contactName || 'Not linked'}
          colors={colors}
          testID="support-case-contact-highlight"
        />
        <SummaryCell
          label="Owner"
          value={caseData.assignee || 'Unassigned'}
          colors={colors}
          testID="support-case-owner-highlight"
        />
      </View>
    </View>
  );
}

function StatusPathSection({ status, colors }: { status: string; colors: ThemeColors }) {
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Case Status</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="support-case-status-path-card">
        <CaseStatusPath status={status} knownStatuses={CASE_STATUS_ORDER} testIDPrefix="support-case-status-path" />
      </View>
    </View>
  );
}

function SlaSection({ slaDeadline, colors }: { slaDeadline?: string; colors: ThemeColors }) {
  if (!slaDeadline) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>SLA Deadline</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="support-case-sla-deadline">
        <Text style={{ color: colors.onSurface }}>{slaDeadline}</Text>
      </View>
    </View>
  );
}

function HandoffSection({ handoffStatus, colors }: { handoffStatus?: string; colors: ThemeColors }) {
  if (!handoffStatus) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Handoff Status</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="support-case-handoff-status">
        <Text style={{ color: colors.onSurface }}>{handoffStatus}</Text>
      </View>
    </View>
  );
}

function ActiveRunBadge({ caseId, colors }: { caseId: string; colors: ThemeColors }) {
  const { data } = useAgentRuns({ status: 'awaiting_approval' });
  const runs = data?.data ?? [];
  const active = runs.find((run: AgentRun) => run.entity_type === 'case' && run.entity_id === caseId) ?? runs[0];
  if (!active) return null;
  return (
    <View style={[styles.card, { backgroundColor: colors.surface }]} testID="support-active-run-status">
      <Text style={{ color: colors.onSurface }}>Agent run: {active.status}</Text>
    </View>
  );
}

function TriggerSection({
  caseData,
  colors,
  triggerAgent,
  triggerKBIsPending,
  onTriggerKB,
}: {
  caseData: SupportCaseDetailData;
  colors: ThemeColors;
  triggerAgent: { mutate: (context: { caseId: string; customerQuery: string }) => void; isPending: boolean };
  triggerKBIsPending: boolean;
  onTriggerKB: () => void;
}) {
  return (
    <>
      <View style={styles.section}>
        <Button
          mode="contained"
          testID="support-trigger-agent-button"
          disabled={triggerAgent.isPending}
          onPress={() => triggerAgent.mutate({ caseId: caseData.id, customerQuery: caseData.subject ?? '' })}
        >
          {triggerAgent.isPending ? 'Running…' : 'Run Support Agent'}
        </Button>
        <ActiveRunBadge caseId={caseData.id} colors={colors} />
      </View>

      {caseData.status === 'resolved' ? (
        <View style={styles.section}>
          <Button mode="outlined" testID="kb-trigger-button" disabled={triggerKBIsPending} onPress={onTriggerKB}>
            {triggerKBIsPending ? 'Running...' : 'Generate KB Article'}
          </Button>
        </View>
      ) : null}
    </>
  );
}

function SignalsHeader({
  colors,
  signalSummary,
}: {
  colors: ThemeColors;
  signalSummary: string | null;
}) {
  return (
    <View style={styles.section}>
      <View style={styles.sectionHeaderRow}>
        <Text style={[styles.sectionTitle, styles.sectionTitleNoMargin, { color: colors.onSurface }]}>Signals</Text>
        {signalSummary ? (
          <View style={[styles.signalSummaryChip, { backgroundColor: colors.surface }]} testID="support-case-signal-summary">
            <Text style={[styles.signalSummaryText, { color: colors.error }]}>{signalSummary}</Text>
          </View>
        ) : null}
      </View>
    </View>
  );
}

export function SupportCaseSummaryTab({
  caseData,
  colors,
  router,
  signalSummary,
  triggerAgent,
  triggerKBIsPending,
  onTriggerKB,
}: {
  caseData: SupportCaseDetailData;
  colors: ThemeColors;
  router: ReturnType<typeof useRouter>;
  signalSummary: string | null;
  triggerAgent: { mutate: (context: { caseId: string; customerQuery: string }) => void; isPending: boolean };
  triggerKBIsPending: boolean;
  onTriggerKB: () => void;
}) {
  return (
    <>
      <HighlightsSection caseData={caseData} colors={colors} router={router} />
      <StatusPathSection status={caseData.status} colors={colors} />
      <SlaSection slaDeadline={caseData.slaDeadline} colors={colors} />
      <HandoffSection handoffStatus={caseData.handoffStatus} colors={colors} />
      <SignalsHeader colors={colors} signalSummary={signalSummary} />
      <TriggerSection
        caseData={caseData}
        colors={colors}
        triggerAgent={triggerAgent}
        triggerKBIsPending={triggerKBIsPending}
        onTriggerKB={onTriggerKB}
      />
      <AgentActivitySection entityType="case" entityId={caseData.id} testIDPrefix="support-case-agent-activity" />
      <EntitySignalsSection entityType="case" entityId={caseData.id} testIDPrefix="support-case-signals" />
    </>
  );
}

export const supportCaseDetailStyles = StyleSheet.create({
  section: { padding: spacing.base },
  sectionTitle: { ...typography.headingMD, marginBottom: spacing.md },
  sectionTitleNoMargin: { marginBottom: 0 },
  sectionHeaderRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md },
  card: { padding: spacing.base, borderRadius: radius.md },
  signalSummaryChip: { borderRadius: radius.full, paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  signalSummaryText: { fontSize: 12, fontWeight: '700' },
  statusChip: { borderRadius: radius.full, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  statusChipText: { color: brandColors.onError, ...typography.eyebrow },
  summaryGrid: { marginTop: spacing.md, gap: spacing.md },
  summaryCell: { borderRadius: radius.md, padding: spacing.base },
  summaryLabel: { ...typography.eyebrow, marginBottom: spacing.xs },
  summaryValue: { fontSize: 14, fontWeight: '600' },
});

const styles = supportCaseDetailStyles;

export { getPriorityColor };
