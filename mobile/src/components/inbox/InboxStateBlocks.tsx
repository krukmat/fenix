import React from 'react';
import { ActivityIndicator, Pressable, Text, View } from 'react-native';
import { resolveHandoffEntityContext } from '../../utils/navigation';
import type { AgentRun, HandoffPackage } from '../../services/api';
import { styles } from './InboxStyles';

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

export function InboxHandoffCard({
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

export function InboxRejectedCard({ run, onPress }: { run: AgentRun; onPress: () => void }) {
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
