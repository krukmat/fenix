// Task 4.2 — FR-300: Root Layout

import React, { useEffect, useState } from 'react';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import * as SplashScreen from 'expo-splash-screen';
import { View, ActivityIndicator } from 'react-native';
import { PaperProvider } from 'react-native-paper';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import * as Sentry from '@sentry/react-native';

import { fenixTheme } from '../src/theme';
import { brandColors } from '../src/theme/colors';
import { useAuthStore } from '../src/stores/authStore';

const isE2E = process.env.EXPO_PUBLIC_E2E_MODE === '1';

// Task 4.9 — NFR-030: Sentry crash reporting
Sentry.init({
  dsn: process.env.EXPO_PUBLIC_SENTRY_DSN ?? '',
  enabled: !isE2E && !!process.env.EXPO_PUBLIC_SENTRY_DSN,
  tracesSampleRate: 0.2,
  debug: false,
});

// Task 4.8 — E2E mode: disable automatic queries so Detox/Espresso sees the app as "idle"
if (!isE2E) {
  SplashScreen.preventAutoHideAsync();
}

// Create QueryClient
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: isE2E ? Infinity : 30_000,
      gcTime: 5 * 60_000,
      retry: isE2E ? 0 : 1,
      refetchOnWindowFocus: false,
      refetchOnMount: isE2E ? false : true,
      refetchOnReconnect: isE2E ? false : true,
    },
  },
});

function RootLayout() {
  const { isLoading, loadStoredToken } = useAuthStore();
  const [isReady, setIsReady] = useState(isE2E);

  useEffect(() => {
    if (isE2E) {
      return;
    }

    async function prepare() {
      try {
        // Load stored token from SecureStore
        await loadStoredToken();
      } catch {
        // loadStoredToken handles its own error state
      } finally {
        setIsReady(true);
        // Hide splash screen after loading
        await SplashScreen.hideAsync();
      }
    }

    prepare();
  }, [loadStoredToken]);

  if (!isReady || isLoading) {
    return (
      <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: brandColors.background }}>
        <ActivityIndicator size="large" color={brandColors.onBackground} />
      </View>
    );
  }

  // Always render ONE Stack with all groups - auth guard is in child layouts
  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <SafeAreaProvider>
        <QueryClientProvider client={queryClient}>
          <PaperProvider theme={fenixTheme}>
            <Stack
              screenOptions={{
                headerShown: false,
              }}
            >
              {/* Auth screens group */}
              <Stack.Screen
                name="(auth)"
                options={{ animation: 'fade' }}
              />
              {/* Main app screens - auth guard is in (tabs)/_layout.tsx */}
              <Stack.Screen
                name="(tabs)"
                options={{ headerShown: false }}
              />
              <Stack.Screen
                name="modal"
                options={{
                  presentation: 'modal',
                  title: 'Modal'
                }}
              />
            </Stack>
            <StatusBar style="light" />
          </PaperProvider>
        </QueryClientProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}

// Task 4.9 — NFR-030: wrap with Sentry for crash capture
export default Sentry.wrap(RootLayout);
