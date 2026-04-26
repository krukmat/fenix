// Support wedge — case detail with agent trigger flow (W3-T2, W3-T3)
// No edit button — edit removed from wedge. Copilot: /support/[id]/copilot.
import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme, Button } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { AgentActivitySection } from '../../../src/components/agents/AgentActivitySection';
import { CRMDetailHeader } from '../../../src/components/crm';
import { useCase } from '../../../src/hooks/useCRM';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import { useTriggerSupportAgent, useTriggerKBAgent, useAgentRuns } from '../../../src/hooks/useWedge';
import { wedgeHref, wedgeHrefObject } from '../../../src/utils/navigation';
import { brandColors, semanticColors } from '../../../src/theme/colors';
import { radius, spacing } from '../../../src/theme/spacing';
import { typography } from '../../../src/theme/typography';
import type { ThemeColors } from '../../../src/theme/types';
import type { AgentRun } from '../../../src/services/api';

// ─── Types ────────────────────────────────────────────────────────────────────

interface CaseDetailData {
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

// ─── Helpers ──────────────────────────────────────────────────────────────────

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function getPriorityColor(priority: string): string {
  if (priority === 'high') return brandColors.error;
  if (priority === 'medium') return semanticColors.warning;
  return semanticColors.success;
}

function getMetadata(c: CaseDetailData) {
  return [
    { label: 'Status', value: c.status },
    { label: 'Priority', value: c.priority },
    { label: 'Assignee', value: c.assignee || 'Unassigned' },
    { label: 'SLA Deadline', value: c.slaDeadline || 'Not set' },
  ];
}

function s(o: Record<string, unknown> | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

type R = Record<string, unknown>;

function parseCaseCore(c: R, handoff: R | undefined): Omit<CaseDetailData, 'accountName' | 'activeSignalCount'> {
  return {
    id: String(c.id ?? ''),
    subject: s(c, 'subject'),
    status: s(c, 'status') ?? 'open',
    priority: (s(c, 'priority') as CaseDetailData['priority'] | undefined) ?? 'medium',
    description: s(c, 'description'),
    accountId: s(c, 'accountId') ?? s(c, 'account_id'),
    slaDeadline: s(c, 'slaDeadline') ?? s(c, 'sla_deadline'),
    handoffStatus: s(handoff, 'status') ?? s(c, 'handoffStatus'),
    assignee: s(c, 'assignee'),
  };
}

function parseCasePayload(data: unknown): CaseDetailData | undefined {
  const payload = (data ?? null) as R | null;
  const c = (payload?.case as R | undefined) ?? payload ?? undefined;
  if (!c) return undefined;
  const acct = payload?.account as R | undefined;
  const handoff = payload?.handoff as R | undefined;
  const signalCount = payload?.active_signal_count;
  return {
    ...parseCaseCore(c, handoff),
    accountName: s(acct, 'name'),
    activeSignalCount: typeof signalCount === 'number' ? signalCount : 0,
  };
}

function formatSignalSummary(activeSignalCount?: number): string | null {
  if (!activeSignalCount || activeSignalCount <= 0) {
    return null;
  }

  return activeSignalCount === 1 ? '1 active signal' : `${activeSignalCount} active signals`;
}

// ─── Section components ───────────────────────────────────────────────────────

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
  accountId, accountName, router, colors,
}: { accountId?: string; accountName?: string; router: ReturnType<typeof useRouter>; colors: ThemeColors }) {
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
  caseData: CaseDetailData;
  colors: ThemeColors;
  triggerAgent: { mutate: (context: { entityType: string; entityId: string }) => void; isPending: boolean };
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
          onPress={() => triggerAgent.mutate({ entityType: 'case', entityId: caseData.id })}
        >
          {triggerAgent.isPending ? 'Running…' : 'Run Support Agent'}
        </Button>
        <ActiveRunBadge caseId={caseData.id} colors={colors} />
      </View>

      {caseData.status === 'resolved' && (
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
      )}
    </>
  );
}

function supportDetailHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Support Case',
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

function SupportCaseLoading({ colors }: { colors: ThemeColors }) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]} testID="support-case-detail-loading">
      <ActivityIndicator size="large" color={colors.primary} />
      <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading case...</Text>
    </View>
  );
}

function SupportCaseError({ colors, message }: { colors: ThemeColors; message: string }) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]} testID="support-case-detail-error">
      <Text style={{ color: colors.error, fontSize: 16 }}>{message}</Text>
    </View>
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function SupportCaseDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useCase(id);
  const caseData = parseCasePayload(data);
  const triggerAgent = useTriggerSupportAgent();
  const triggerKB = useTriggerKBAgent();
  const signalSummary = caseData ? formatSignalSummary(caseData.activeSignalCount) : null;

  const handleKBTrigger = () => {
    if (!caseData) return;
    triggerKB.mutate({ caseId: caseData.id }, {
      onSuccess: (result) => {
        if (result?.runId) router.push(wedgeHref(`/activity/${result.runId}`));
      },
    });
  };

  if (isLoading) return <SupportCaseLoading colors={colors} />;
  if (error || !caseData) return <SupportCaseError colors={colors} message={error?.message || 'Case not found'} />;

  return (
    <>
      <Stack.Screen options={supportDetailHeaderOptions(colors)} />
      <View
        testID="support-case-detail-screen"
        style={[styles.container, { backgroundColor: colors.background }]}
      >
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

          <View style={styles.section}>
            <View style={styles.sectionHeaderRow}>
              <Text style={[styles.sectionTitle, styles.sectionTitleNoMargin, { color: colors.onSurface }]}>Signals</Text>
              {signalSummary ? (
                <View
                  style={[styles.signalSummaryChip, { backgroundColor: colors.surface }]}
                  testID="support-case-signal-summary"
                >
                  <Text style={[styles.signalSummaryText, { color: colors.error }]}>
                    {signalSummary}
                  </Text>
                </View>
              ) : null}
            </View>
          </View>

          <TriggerSection caseData={caseData} colors={colors} triggerAgent={triggerAgent} triggerKBIsPending={triggerKB.isPending} onTriggerKB={handleKBTrigger} />

          <AgentActivitySection entityType="case" entityId={caseData.id} testIDPrefix="support-case-agent-activity" />
          <EntitySignalsSection entityType="case" entityId={caseData.id} testIDPrefix="support-case-signals" />

          <View style={styles.section}>
            <Button
              mode="outlined"
              testID="support-copilot-button"
              onPress={() => router.push(wedgeHrefObject(`/support/${caseData.id}/copilot`, { entity_type: 'case', entity_id: caseData.id }))}
            >
              Open Copilot
            </Button>
          </View>
        </ScrollView>
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
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
