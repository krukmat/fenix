import React from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { useTheme } from 'react-native-paper';
import type { ThemeColors } from '../../theme/types';
import { ApprovalLinkageSection, HandoffSection, HighlightsSection, RejectionSection } from './RunInspectorOverview';
import { EvidenceSection, OutputSection, ReasoningTraceSection, ToolActivitySection, UsageSection } from './RunInspectorDetails';
import {
  normalizeEvidenceItems,
  normalizeReasoningTrace,
  normalizeToolCalls,
  type RunInspectorProps,
} from './runInspector.model';

export function RunInspector(props: RunInspectorProps) {
  const colors = useTheme().colors as ThemeColors;
  const { run, usage, handoff, handoffLoading, onOpenAuditTrail, onOpenInbox, onOpenHandoffDestination, onViewFullUsage } = props;
  const evidenceItems = normalizeEvidenceItems(run);
  const toolCalls = normalizeToolCalls(run);
  const trace = normalizeReasoningTrace(run);

  return (
    <ScrollView testID="activity-run-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <HighlightsSection run={run} onOpenAuditTrail={onOpenAuditTrail} />
      <RejectionSection reason={run.status === 'denied_by_policy' ? run.rejection_reason : undefined} />
      <EvidenceSection run={run} items={evidenceItems} />
      <ToolActivitySection calls={toolCalls} />
      <ApprovalLinkageSection run={run} onOpenInbox={onOpenInbox} />
      <HandoffSection
        run={run}
        handoff={handoff}
        handoffLoading={handoffLoading}
        onOpenHandoffDestination={onOpenHandoffDestination}
      />
      <ReasoningTraceSection trace={trace} />
      <OutputSection output={run.output} />
      <UsageSection runId={run.id} events={usage} onViewFullUsage={onViewFullUsage} />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
