// Support copilot — contextual copilot route under support case detail (W3-T4)
// Route: /support/[id]/copilot  params: entity_type, entity_id
import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Stack, useLocalSearchParams } from 'expo-router';
import { CopilotPanel } from '../../../../src/components/copilot';

export default function SupportCopilotScreen() {
  const params = useLocalSearchParams<{ id: string | string[]; entity_type?: string; entity_id?: string }>();
  const caseId = Array.isArray(params.id) ? params.id[0] : params.id;
  const entityType = params.entity_type ?? 'case';
  const entityId = params.entity_id ?? caseId;

  return (
    <>
      <Stack.Screen options={{ title: 'Support Copilot' }} />
      <View style={styles.container}>
        <CopilotPanel
          testIDPrefix="support-copilot"
          entityType={entityType}
          entityId={entityId}
        />
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
});
