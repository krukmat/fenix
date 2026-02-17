// Task 4.2 — FR-300: Auth Layout con redirect cuando ya autenticado

import React from 'react';
import { Redirect, Stack } from 'expo-router';
import { useAuthStore } from '../../src/stores/authStore';

export default function AuthLayout() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  // Redirect to tabs if already authenticated
  if (isAuthenticated) {
    return <Redirect href="/(tabs)/accounts" />;
  }

  return (
    <Stack
      screenOptions={{
        headerShown: false,
      }}
    >
      <Stack.Screen name="login" />
      <Stack.Screen name="register" />
    </Stack>
  );
}
