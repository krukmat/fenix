// Task 4.4 — FR-200: Copilot Chat Screen
// Task Mobile P1.7 — FR-200/UC-A5: signal-aware context from route params
import React from 'react';
import { Stack, useLocalSearchParams } from 'expo-router';
import { CopilotPanel } from '../../../src/components/copilot';
import type { CopilotInitialContext } from '../../../src/components/copilot/CopilotPanel';

export default function CopilotScreen() {
  const params = useLocalSearchParams<{
    entity_type?: string;
    entity_id?: string;
    signal_id?: string;
    signal_type?: string;
  }>();

  const hasContext = params.entity_type || params.entity_id || params.signal_id || params.signal_type;

  const initialContext: CopilotInitialContext | undefined = hasContext
    ? {
        entityType: params.entity_type,
        entityId: params.entity_id,
        signalId: params.signal_id,
        signalType: params.signal_type,
      }
    : undefined;

  return (
    <>
      <Stack.Screen options={{ title: 'Copilot' }} />
      <CopilotPanel initialContext={initialContext} />
    </>
  );
}
