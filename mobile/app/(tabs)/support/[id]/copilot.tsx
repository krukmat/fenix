// Support copilot — contextual copilot route under support case detail (W3-T4)
// F9.A5: wired to canonical support trigger { case_id, customer_query }
// Route: /support/[id]/copilot  params: entity_type, entity_id
import React, { useCallback } from 'react';
import { View, StyleSheet } from 'react-native';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { CopilotPanel } from '../../../../src/components/copilot';
import { useTriggerSupportAgent } from '../../../../src/hooks/useWedge';

export default function SupportCopilotScreen() {
  const params = useLocalSearchParams<{ id: string | string[]; entity_type?: string; entity_id?: string }>();
  const router = useRouter();
  const caseId = Array.isArray(params.id) ? params.id[0] : params.id;
  const entityType = params.entity_type ?? 'case';
  const entityId = params.entity_id ?? caseId;

  const { mutateAsync } = useTriggerSupportAgent();

  // F9.A5: fires when operator submits a query in Copilot — triggers governed support run
  const onSupportTrigger = useCallback(
    async (customerQuery: string) => {
      const run = await mutateAsync({ caseId, customerQuery, language: undefined, priority: undefined });
      router.push(`/activity/${run.runId}`);
    },
    [caseId, mutateAsync, router],
  );

  return (
    <>
      <Stack.Screen options={{ title: 'Support Copilot' }} />
      <View style={styles.container}>
        <CopilotPanel
          initialContext={{ entityType, entityId }}
          onSupportTrigger={onSupportTrigger}
        />
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
});
