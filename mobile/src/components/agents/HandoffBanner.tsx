// Task Mobile P1.8 — FR-232/UC-A7: HandoffBanner shown when agent run status = "escalated"

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Banner, Text, ActivityIndicator } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useHandoffPackage } from '../../hooks/useAgentSpec';
import { resolveWedgeHandoffPackageDestination, wedgeHref } from '../../utils/navigation';

interface HandoffBannerProps {
  runId: string;
  caseId?: string;
  testIDPrefix?: string;
}

export function HandoffBanner({ runId, caseId, testIDPrefix = 'handoff-banner' }: HandoffBannerProps) {
  const router = useRouter();
  const { data: handoff, isLoading } = useHandoffPackage(runId, caseId, true);

  if (isLoading) {
    return (
      <View style={styles.loading} testID={`${testIDPrefix}-loading`}>
        <ActivityIndicator size="small" />
      </View>
    );
  }

  if (!handoff) return null;

  const handleAccept = () => {
    router.push(wedgeHref(resolveWedgeHandoffPackageDestination(handoff, runId)));
  };

  return (
    <Banner
      visible
      testID={testIDPrefix}
      icon="account-alert-outline"
      actions={[
        {
          label: 'Accept Handoff',
          onPress: handleAccept,
          testID: `${testIDPrefix}-accept`,
        },
      ]}
    >
      <View>
        <Text style={styles.title} testID={`${testIDPrefix}-reason`}>
          {handoff.reason}
        </Text>
        {handoff.conversation_context ? (
          <Text style={styles.context} testID={`${testIDPrefix}-context`} numberOfLines={2}>
            {handoff.conversation_context}
          </Text>
        ) : null}
        <Text style={styles.evidence} testID={`${testIDPrefix}-evidence-count`}>
          {handoff.evidence_count} evidence item{handoff.evidence_count !== 1 ? 's' : ''} preserved
        </Text>
      </View>
    </Banner>
  );
}

const styles = StyleSheet.create({
  loading: { padding: 12, alignItems: 'center' },
  title: { fontWeight: '600', fontSize: 14, marginBottom: 4 },
  context: { fontSize: 12, color: '#555', marginBottom: 4 },
  evidence: { fontSize: 11, color: '#888' },
});
