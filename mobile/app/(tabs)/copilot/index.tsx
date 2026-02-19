// Task 4.4 — FR-200: Copilot Chat Screen (reemplaza placeholder)
import React from 'react';
import { Stack } from 'expo-router';
import { CopilotPanel } from '../../../src/components/copilot';

export default function CopilotScreen() {
  return (
    <>
      <Stack.Screen options={{ title: 'Copilot' }} />
      <CopilotPanel />
    </>
  );
}
