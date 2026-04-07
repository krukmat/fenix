// Sales wedge — copilot route (W4-T4)
// Route: /sales/[id]/copilot  params: entity_type, entity_id
import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Stack, useLocalSearchParams } from 'expo-router';
import { CopilotPanel } from '../../../../src/components/copilot';

export default function SalesCopilotScreen() {
  const params = useLocalSearchParams<{
    id: string | string[];
    entity_type?: string;
    entity_id?: string;
  }>();
  const rawId = Array.isArray(params.id) ? params.id[0] : params.id;
  const entityType = params.entity_type ?? 'account';
  const entityId = params.entity_id ?? (rawId.startsWith('deal-') ? rawId.slice(5) : rawId);

  return (
    <>
      <Stack.Screen options={{ title: 'Sales Copilot' }} />
      <View style={styles.container}>
        <CopilotPanel initialContext={{ entityType, entityId }} />
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
});
