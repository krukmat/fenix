// Task Mobile P1.4 — CRM hub stack navigator

import React from 'react';
import { Stack } from 'expo-router';
import { darkStackScreenOptions } from '../../../src/navigation/darkStackOptions';

export default function CRMLayout() {
  return (
    <Stack
      screenOptions={darkStackScreenOptions}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="accounts/index" options={{ title: 'Accounts', headerShown: true }} />
      <Stack.Screen name="accounts/new" options={{ title: 'New Account', headerShown: true }} />
      <Stack.Screen name="accounts/[id]" options={{ title: 'Account', headerShown: true }} />
      <Stack.Screen name="accounts/edit/[id]" options={{ title: 'Edit Account', headerShown: true }} />
      <Stack.Screen name="contacts/index" options={{ title: 'Contacts', headerShown: true }} />
      <Stack.Screen name="contacts/new" options={{ title: 'New Contact', headerShown: true }} />
      <Stack.Screen name="contacts/[id]" options={{ title: 'Contact', headerShown: true }} />
      <Stack.Screen name="contacts/edit/[id]" options={{ title: 'Edit Contact', headerShown: true }} />
      <Stack.Screen name="leads/index" options={{ title: 'Leads', headerShown: true }} />
      <Stack.Screen name="leads/new" options={{ title: 'New Lead', headerShown: true }} />
      <Stack.Screen name="leads/[id]" options={{ title: 'Lead', headerShown: true }} />
      <Stack.Screen name="leads/edit/[id]" options={{ title: 'Edit Lead', headerShown: true }} />
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
