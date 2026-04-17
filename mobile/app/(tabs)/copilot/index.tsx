// Task Mobile P1.4 — Copilot screen: reads route params and passes initialContext
// FR-200 (Copilot embedded), UC-A5: signal-aware context from route params
import React from 'react';
import { Stack, useLocalSearchParams } from 'expo-router';
import { CopilotPanel } from '../../../src/components/copilot';

type CopilotInitialContext = {
  entityType?: string;
  entityId?: string;
  signalId?: string;
  signalType?: string;
};

function buildInitialContext(params: Record<string, string | string[]>): CopilotInitialContext | undefined {
  const entityType = params.entity_type as string | undefined;
  const entityId = params.entity_id as string | undefined;
  const signalId = params.signal_id as string | undefined;
  const signalType = params.signal_type as string | undefined;

  if (!entityType && !entityId && !signalId && !signalType) return undefined;

  return {
    ...(entityType ? { entityType } : {}),
    ...(entityId ? { entityId } : {}),
    ...(signalId ? { signalId } : {}),
    ...(signalType ? { signalType } : {}),
  };
}

export default function CopilotScreen() {
  const params = useLocalSearchParams();
  const initialContext = buildInitialContext(params as Record<string, string | string[]>);

  return (
    <>
      <Stack.Screen options={{ headerShown: false }} />
      <CopilotPanel initialContext={initialContext} />
    </>
  );
}
