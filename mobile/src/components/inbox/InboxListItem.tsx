import React from 'react';
import { View } from 'react-native';
import { useRouter } from 'expo-router';
import { ApprovalCard } from '../approvals/ApprovalCard';
import { SignalCard } from '../signals/SignalCard';
import { resolveWedgeHandoffPackageDestination, wedgeHref, wedgeHrefObject } from '../../utils/navigation';
import type { AgentRun, ApprovalRequest, HandoffPackage, Signal } from '../../services/api';
import { InboxHandoffCard, InboxRejectedCard } from './InboxStateBlocks';
import { styles } from './InboxStyles';

export type InboxRenderableItem =
  | { type: 'approval'; id: string; approval: ApprovalRequest }
  | { type: 'handoff'; id: string; runId: string; handoff: HandoffPackage }
  | { type: 'signal'; id: string; signal: Signal }
  | { type: 'rejected'; id: string; run: AgentRun };

export function InboxListItem({
  item,
  index,
  onApprove,
  onReject,
  approvalsPending,
}: {
  item: InboxRenderableItem;
  index: number;
  onApprove: (id: string) => void;
  onReject: (id: string, reason: string) => void;
  approvalsPending: boolean;
}) {
  const router = useRouter();

  return (
    <View
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
