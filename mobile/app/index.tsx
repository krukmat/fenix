import React from 'react';
import { Redirect } from 'expo-router';

import { useAuthStore } from '../src/stores/authStore';

export default function IndexRoute() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  if (isAuthenticated) {
    return <Redirect href="/accounts" />;
  }

  return <Redirect href="/login" />;
}
