import React from 'react';
import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { Button } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { AgentActivitySection } from '../agents/AgentActivitySection';
import { CRMDetailHeader } from '../crm';
import { EntitySignalsSection } from '../signals/EntitySignalsSection';
import { useAgentRuns } from '../../hooks/useWedge';
import { wedgeHref, wedgeHrefObject } from '../../utils/navigation';
import { brandColors, semanticColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { AgentRun } from '../../services/api';
import type { ThemeColors } from '../../theme/types';

export interface SupportCaseDetailData {
  id: string;
  subject?: string;
  status: string;
  priority: 'low' | 'medium' | 'high';
  description?: string;
  accountId?: string;
  accountName?: string;
  slaDeadline?: string;
  handoffStatus?: string;
  assignee?: string;
  activeSignalCount?: number;
}

function getPriorityColor(priority: string): string {
  if (priority === 'high') return brandColors.error;
  if (priority === 'medium') return semanticColors.warning;
  return semanticColors.success;
}

function getMetadata(c: SupportCaseDetailData) {
  return [
    { label: 'Status', value: c.status },
    { label: 'Priority', value: c.priority },
    { label: 'Assignee', value: c.assignee || 'Unassigned' },
    { label: 'SLA Deadline', value: c.slaDeadline || 'Not set' },
  ];
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

function AccountSection({
  accountId,
  accountName,
  router,
  colors,
}: {
  accountId?: string;
  accountName?: string;
  router: ReturnType<typeof useRouter>;
  colors: ThemeColors;
}) {
  if (!accountId) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Account</Text>
      <TouchableOpacity
        style={[styles.card, { backgroundColor: colors.surface }]}
        onPress={() => router.push(wedgeHref(`/sales/${accountId}`))}
      >
        <Text style={{ color: colors.onSurface, fontWeight: '500' }}>{accountName || 'View Account'}</Text>
      </TouchableOpacity>
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
          <Button
            mode="outlined"
            testID="kb-trigger-button"
            disabled={triggerKBIsPending}
            onPress={onTriggerKB}
          >
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
          <View
            style={[styles.signalSummaryChip, { backgroundColor: colors.surface }]}
            testID="support-case-signal-summary"
          >
            <Text style={[styles.signalSummaryText, { color: colors.error }]}>{signalSummary}</Text>
          </View>
        ) : null}
      </View>
    </View>
  );
}

export function SupportCaseDetailContent({
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
    <View testID="support-case-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <ScrollView style={styles.container}>
        <View style={[styles.priorityBanner, { backgroundColor: getPriorityColor(caseData.priority) }]}>
          <Text style={styles.priorityText}>PRIORITY: {caseData.priority.toUpperCase()}</Text>
        </View>

        <CRMDetailHeader
          title={caseData.subject || 'No Subject'}
          subtitle={caseData.description}
          metadata={getMetadata(caseData)}
          testIDPrefix="support-case-detail"
        />

        <SlaSection slaDeadline={caseData.slaDeadline} colors={colors} />
        <HandoffSection handoffStatus={caseData.handoffStatus} colors={colors} />
        <AccountSection accountId={caseData.accountId} accountName={caseData.accountName} router={router} colors={colors} />
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

        <View style={styles.section}>
          <Button
            mode="outlined"
            testID="support-copilot-button"
            onPress={() =>
              router.push(
                wedgeHrefObject(`/support/${caseData.id}/copilot`, { entity_type: 'case', entity_id: caseData.id })
              )
            }
          >
            Open Copilot
          </Button>
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  priorityBanner: { padding: spacing.sm, alignItems: 'center' },
  priorityText: { color: brandColors.onError, fontWeight: '600', fontSize: 14 },
  section: { padding: spacing.base },
  sectionTitle: { ...typography.headingMD, marginBottom: spacing.md },
  sectionTitleNoMargin: { marginBottom: 0 },
  sectionHeaderRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md },
  signalSummaryChip: {
    borderRadius: radius.full,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
  },
  signalSummaryText: {
    fontSize: 12,
    fontWeight: '700',
  },
  card: { padding: spacing.base, borderRadius: radius.md },
});
