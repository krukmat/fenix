// mobile_tab_bar_overflow_fix_plan: Stack navigator for sales feature routes.
// Prevents sales/[id].tsx and sales/deal-[id].tsx from leaking into the parent
// Tabs navigator as ghost tabs.

import React from 'react';
import { Stack } from 'expo-router';

export default function SalesLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Deal Detail', headerShown: true }} />
      <Stack.Screen name="deal-[id]" options={{ title: 'Deal', headerShown: true }} />
    </Stack>
  );
}
