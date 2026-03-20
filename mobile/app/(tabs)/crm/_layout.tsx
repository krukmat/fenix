// Task Mobile P1.4 — CRM hub stack navigator

import React from 'react';
import { Stack } from 'expo-router';

export default function CRMLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="accounts/index" options={{ title: 'Accounts', headerShown: true }} />
      <Stack.Screen name="accounts/new" options={{ title: 'New Account', headerShown: true }} />
      <Stack.Screen name="accounts/[id]" options={{ title: 'Account', headerShown: true }} />
      <Stack.Screen name="contacts/index" options={{ title: 'Contacts', headerShown: true }} />
      <Stack.Screen name="contacts/[id]" options={{ title: 'Contact', headerShown: true }} />
      <Stack.Screen name="deals/index" options={{ title: 'Deals', headerShown: true }} />
      <Stack.Screen name="deals/new" options={{ title: 'New Deal', headerShown: true }} />
      <Stack.Screen name="deals/[id]" options={{ title: 'Deal', headerShown: true }} />
      <Stack.Screen name="deals/edit/[id]" options={{ title: 'Edit Deal', headerShown: true }} />
      <Stack.Screen name="cases/index" options={{ title: 'Cases', headerShown: true }} />
      <Stack.Screen name="cases/new" options={{ title: 'New Case', headerShown: true }} />
      <Stack.Screen name="cases/[id]" options={{ title: 'Case', headerShown: true }} />
      <Stack.Screen name="cases/edit/[id]" options={{ title: 'Edit Case', headerShown: true }} />
    </Stack>
  );
}
