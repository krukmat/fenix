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

import { fenixTheme } from '../src/theme';
import { useAuthStore } from '../src/stores/authStore';

// Keep splash screen visible while loading
SplashScreen.preventAutoHideAsync();

// Create QueryClient
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export default function RootLayout() {
  const { isLoading, loadStoredToken } = useAuthStore();
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    async function prepare() {
      try {
        // Load stored token from SecureStore
        await loadStoredToken();
      } catch (e) {
        console.warn('Error loading stored token:', e);
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
      <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: '#1565C0' }}>
        <ActivityIndicator size="large" color="#FFFFFF" />
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
                name="(auth)/login" 
                options={{ animation: 'fade' }}
              />
              <Stack.Screen 
                name="(auth)/register" 
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
            <StatusBar style="auto" />
          </PaperProvider>
        </QueryClientProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}
