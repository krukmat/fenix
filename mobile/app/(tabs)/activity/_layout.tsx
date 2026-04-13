// Task Mobile P1.5 — Activity Log stack navigator (renamed from agents)

import React from 'react';
import { Stack } from 'expo-router';

export default function ActivityLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="insights" />
      <Stack.Screen name="[id]" options={{ title: 'Run Detail', headerShown: true }} />
    </Stack>
  );
}
