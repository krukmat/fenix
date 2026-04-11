// mobile_tab_bar_overflow_fix_plan: Stack navigator for accounts feature routes.
// Prevents accounts/[id].tsx and accounts/new.tsx from leaking into the parent
// Tabs navigator as ghost tabs.

import React from 'react';
import { Stack } from 'expo-router';

export default function AccountsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Account Detail', headerShown: true }} />
      <Stack.Screen name="new" options={{ title: 'New Account', headerShown: true }} />
    </Stack>
  );
}
