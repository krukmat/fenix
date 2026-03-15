// Task Mobile P1.4 — Home stack navigator

import React from 'react';
import { Stack } from 'expo-router';

export default function HomeLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: 'slide_from_right',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="signal/[id]" options={{ title: 'Signal Detail', headerShown: true }} />
    </Stack>
  );
}
