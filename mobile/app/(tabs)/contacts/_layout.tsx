// crm-dentro-governance: Stack navigator for hidden Contacts routes.
// Keeps contacts/[id].tsx from leaking into the parent Tabs navigator.

import React from 'react';
import { Stack } from 'expo-router';

export default function ContactsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="[id]" options={{ title: 'Contact', headerShown: true }} />
    </Stack>
  );
}
