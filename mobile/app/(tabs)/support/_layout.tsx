// mobile_tab_bar_overflow_fix_plan: Stack navigator for support feature routes.
// Prevents support/[id].tsx from leaking into the parent Tabs navigator as a
// ghost tab.

import React from 'react';
import { Stack } from 'expo-router';

export default function SupportLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Case Detail', headerShown: true }} />
    </Stack>
  );
}
