// Wave 1 — governance_mobile_enhancement_plan: Stack navigator for governance sub-screens.
// Prerequisite for governance/audit.tsx (Wave 2) and governance/usage.tsx (Wave 3).
// Follows the exact pattern of app/(tabs)/activity/_layout.tsx.

import React from 'react';
import { Stack } from 'expo-router';

export default function GovernanceLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="audit" options={{ title: 'Audit Trail', headerShown: true }} />
      <Stack.Screen name="usage" options={{ title: 'Usage Events', headerShown: true }} />
    </Stack>
  );
}
