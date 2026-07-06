import React from 'react';
import { Button, Text } from 'react-native-paper';
import { View } from 'react-native';
import { useRouter } from 'expo-router';
import { AgentActivitySection } from '../agents/AgentActivitySection';
import { EntitySignalsSection } from '../signals/EntitySignalsSection';
import { useAgentRuns } from '../../hooks/useWedge';
import type { AgentRun } from '../../services/api';
import type { ThemeColors } from '../../theme/types';
import {
  HandoffSection,
  HighlightsSection,
  SignalsHeader,
  SlaSection,
  StatusPathSection,
  supportCaseDetailStyles as styles,
} from './SupportCaseDetailMeta';
import type { SupportCaseDetailData } from './supportCaseDetail.types';

function Section({ children }: { children: React.ReactNode }) {
  return <View style={styles.section}>{children}</View>;
}

function ActiveRunBadge({ caseId, colors }: { caseId: string; colors: ThemeColors }) {
  const { data } = useAgentRuns({ status: 'awaiting_approval' });
  const runs = data?.data ?? [];
  const active = runs.find((run: AgentRun) => run.entity_type === 'case' && run.entity_id === caseId) ?? runs[0];
  return active ? <View style={[styles.card, { backgroundColor: colors.surface }]} testID="support-active-run-status"><Text style={{ color: colors.onSurface }}>Agent run: {active.status}</Text></View> : null;
}

function TriggerSection({ caseData, colors, triggerAgent, triggerKBIsPending, onTriggerKB }: { caseData: SupportCaseDetailData; colors: ThemeColors; triggerAgent: { mutate: (context: { caseId: string; customerQuery: string }) => void; isPending: boolean }; triggerKBIsPending: boolean; onTriggerKB: () => void }) {
  return <>
    <Section>
      <Button mode="contained" testID="support-trigger-agent-button" disabled={triggerAgent.isPending} onPress={() => triggerAgent.mutate({ caseId: caseData.id, customerQuery: caseData.subject ?? '' })}>
        {triggerAgent.isPending ? 'Running…' : 'Run Support Agent'}
      </Button>
      <ActiveRunBadge caseId={caseData.id} colors={colors} />
    </Section>
    {caseData.status === 'resolved' ? <Section>
      <Button mode="outlined" testID="kb-trigger-button" disabled={triggerKBIsPending} onPress={onTriggerKB}>
        {triggerKBIsPending ? 'Running...' : 'Generate KB Article'}
      </Button>
    </Section> : null}
  </>;
}

export function SupportCaseSummaryTab({ caseData, colors, router, signalSummary, triggerAgent, triggerKBIsPending, onTriggerKB }: { caseData: SupportCaseDetailData; colors: ThemeColors; router: ReturnType<typeof useRouter>; signalSummary: string | null; triggerAgent: { mutate: (context: { caseId: string; customerQuery: string }) => void; isPending: boolean }; triggerKBIsPending: boolean; onTriggerKB: () => void }) {
  return <>
    <Section><HighlightsSection caseData={caseData} colors={colors} router={router} /></Section>
    <Section><StatusPathSection status={caseData.status} colors={colors} /></Section>
    <Section><SlaSection slaDeadline={caseData.slaDeadline} colors={colors} /></Section>
    <Section><HandoffSection handoffStatus={caseData.handoffStatus} colors={colors} /></Section>
    <Section><SignalsHeader colors={colors} signalSummary={signalSummary} /></Section>
    <TriggerSection caseData={caseData} colors={colors} triggerAgent={triggerAgent} triggerKBIsPending={triggerKBIsPending} onTriggerKB={onTriggerKB} />
    <AgentActivitySection entityType="case" entityId={caseData.id} testIDPrefix="support-case-agent-activity" />
    <EntitySignalsSection entityType="case" entityId={caseData.id} testIDPrefix="support-case-signals" />
  </>;
}
