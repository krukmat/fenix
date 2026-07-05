import React from 'react';
import { View } from 'react-native';
import { useRouter } from 'expo-router';
import { ApprovalCard, type ApprovalDecisionFeedback } from '../approvals/ApprovalCard';
import { APPROVAL_STEP_COLORS, normalizeApprovalStatus } from '../approvals/ApprovalPath';
import { SignalCard } from '../signals/SignalCard';
import { resolveWedgeHandoffPackageDestination, wedgeHref, wedgeHrefObject } from '../../utils/navigation';
import type { AgentRun, ApprovalRequest, HandoffPackage, Signal } from '../../services/api';
import { getAgentStatusColor, getConfidenceColor } from '../../theme/semantic';
import { InboxHandoffCard, InboxRejectedCard } from './InboxStateBlocks';
import { styles } from './InboxStyles';

export type InboxRenderableItem =
  | { type: 'approval'; id: string; approval: ApprovalRequest }
  | { type: 'handoff'; id: string; runId: string; handoff: HandoffPackage }
  | { type: 'signal'; id: string; signal: Signal }
  | { type: 'rejected'; id: string; run: AgentRun };

function getInboxItemAccentColor(
  item: InboxRenderableItem,
  approvalDecisionFeedback: ApprovalDecisionFeedback | undefined,
): string {
  if (item.type === 'approval') {
    const hasUnresolvedConflict = approvalDecisionFeedback?.kind === 'conflict' && !approvalDecisionFeedback.visibleStatus;
    if (hasUnresolvedConflict) return getAgentStatusColor('denied_by_policy');
    const status = approvalDecisionFeedback?.visibleStatus ?? item.approval.status;
    return APPROVAL_STEP_COLORS[normalizeApprovalStatus(status)];
  }
  if (item.type === 'signal') return getConfidenceColor(item.signal.confidence);
  if (item.type === 'rejected') return getAgentStatusColor(item.run.status);
  return getAgentStatusColor('handed_off');
}

export function InboxListItem({
  item,
  index,
  approvalDecisionFeedback,
  onApprove,
  onReject,
  approvalsPending,
}: {
  item: InboxRenderableItem;
  index: number;
  approvalDecisionFeedback?: ApprovalDecisionFeedback;
  onApprove: (id: string, comment?: string) => void;
  onReject: (id: string, reason: string) => void;
  approvalsPending: boolean;
}) {
  const router = useRouter();

  return (
    <View
      style={[
        styles.item,
        styles.itemAccent,
        { borderLeftColor: getInboxItemAccentColor(item, approvalDecisionFeedback) },
      ]}
      testID={`inbox-item-${index}`}
      accessibilityLabel={`${item.type}:${item.id}`}
    >
      {item.type === 'approval' ? (
        <ApprovalCard
          approval={item.approval}
          onApprove={onApprove}
          onReject={onReject}
          decisionFeedback={approvalDecisionFeedback}
          testIDPrefix={`inbox-approval-${item.approval.id}`}
          disabled={approvalsPending}
        />
      ) : null}
      {item.type === 'handoff' ? (
        <InboxHandoffCard
          handoff={item.handoff}
          runId={item.runId}
          onPress={() => router.push(wedgeHref(resolveWedgeHandoffPackageDestination(item.handoff, item.runId)))}
        />
      ) : null}
      {item.type === 'signal' ? (
        <SignalCard
          signal={item.signal}
          onDismiss={() => {}}
          onPress={(signal) =>
            router.push(
              wedgeHrefObject('/(tabs)/home/signal/[id]', {
                id: signal.id,
                entity_type: signal.entity_type,
                entity_id: signal.entity_id,
              })
            )
          }
          testIDPrefix={`inbox-signal-${item.signal.id}`}
        />
      ) : null}
      {item.type === 'rejected' ? (
        <InboxRejectedCard
          run={item.run}
          onPress={() => router.push(wedgeHref(`/activity/${item.run.id}`))}
        />
      ) : null}
    </View>
  );
}
