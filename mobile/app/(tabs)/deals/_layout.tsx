// mobile_tab_bar_overflow_fix_plan: Stack navigator for deals feature routes.
// Prevents deals/[id].tsx and deals/new.tsx from leaking into the parent
// Tabs navigator as ghost tabs.

import React from 'react';
import { Stack } from 'expo-router';

export default function DealsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Deal Detail', headerShown: true }} />
      <Stack.Screen name="new" options={{ title: 'New Deal', headerShown: true }} />
    </Stack>
  );
}
