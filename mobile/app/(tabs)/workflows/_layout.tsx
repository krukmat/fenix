// Task Mobile P1.4 — Workflows stack navigator

import React from 'react';
import { Stack } from 'expo-router';

export default function WorkflowsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Workflow', headerShown: true }} />
    </Stack>
  );
}
