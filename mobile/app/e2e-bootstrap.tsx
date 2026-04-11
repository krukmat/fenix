// docs/plans/maestro-screenshot-auth-bypass-plan.md — Task 2
// Runtime-only governance gate: the route only accepts auth-injection params
// when EXPO_PUBLIC_E2E_MODE === '1'. In any other build it immediately
// redirects to /login without calling login() or mutating auth state.
// This keeps production builds free of an unconditional auth-injection
// surface, per the "Tools, not mutations" principle in CLAUDE.md.
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

    // Governance gate — must be evaluated before touching any params.
    if (process.env.EXPO_PUBLIC_E2E_MODE !== '1') {
      router.replace('/login');
      return;
    }

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
