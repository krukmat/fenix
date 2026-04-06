import React, { useEffect, useRef } from 'react';
import { ActivityIndicator, View } from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';

import { useAuthStore } from '../src/stores/authStore';

function readParam(value: string | string[] | undefined): string | null {
  if (Array.isArray(value)) return value[0] ?? null;
  return value ?? null;
}

export default function E2EBootstrapRoute() {
  const router = useRouter();
  const login = useAuthStore((state) => state.login);
  const hasBootstrapped = useRef(false);
  const params = useLocalSearchParams<{
    token?: string | string[];
    userId?: string | string[];
    workspaceId?: string | string[];
    redirect?: string | string[];
  }>();

  useEffect(() => {
    if (hasBootstrapped.current) return;
    hasBootstrapped.current = true;

    const token = readParam(params.token);
    const userId = readParam(params.userId);
    const workspaceId = readParam(params.workspaceId);
    const redirect = readParam(params.redirect) ?? '/home';

    async function bootstrap() {
      if (!token || !userId || !workspaceId) {
        router.replace('/login');
        return;
      }

      await login({ token, userId, workspaceId });
      router.replace(redirect as '/home');
    }

    bootstrap().catch(() => {
      router.replace('/login');
    });
  }, [login, params.redirect, params.token, params.userId, params.workspaceId, router]);

  return (
    <View
      testID="e2e-bootstrap-screen"
      style={{ flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: '#1565C0' }}
    >
      <ActivityIndicator size="large" color="#FFFFFF" />
    </View>
  );
}
