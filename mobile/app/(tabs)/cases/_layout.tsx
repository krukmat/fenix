// mobile_tab_bar_overflow_fix_plan: Stack navigator for cases feature routes.
// Prevents cases/[id].tsx and cases/new.tsx from leaking into the parent
// Tabs navigator as ghost tabs.

import React from 'react';
import { Stack } from 'expo-router';

export default function CasesLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Case Detail', headerShown: true }} />
      <Stack.Screen name="new" options={{ title: 'New Case', headerShown: true }} />
    </Stack>
  );
}
