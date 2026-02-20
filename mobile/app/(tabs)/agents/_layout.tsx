// Task 4.5 — Agent Runs Stack Layout

import React from 'react';
import { Stack } from 'expo-router';

export default function AgentRunsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Agent Run Details' }} />
    </Stack>
  );
}
