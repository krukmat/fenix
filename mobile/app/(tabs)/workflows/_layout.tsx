// Task Mobile P1.4 — Workflows stack navigator

import React from 'react';
import { Stack } from 'expo-router';
import { darkStackScreenOptions } from '../../../src/navigation/darkStackOptions';

export default function WorkflowsLayout() {
  return (
    <Stack
      screenOptions={darkStackScreenOptions}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="new" options={{ title: 'New Workflow', headerShown: true }} />
      <Stack.Screen name="[id]" options={{ title: 'Workflow', headerShown: true }} />
      <Stack.Screen name="edit/[id]" options={{ title: 'Edit Workflow', headerShown: true }} />
    </Stack>
  );
}
