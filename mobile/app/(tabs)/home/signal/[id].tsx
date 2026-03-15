// UC-A5/B4: Signal detail screen
// FR-200: signal context forwarding to Copilot

import React, { useCallback } from 'react';
import { View, ActivityIndicator, StyleSheet } from 'react-native';
import { Text, Button, useTheme } from 'react-native-paper';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { SignalDetailView } from '../../../../src/components/signals/SignalDetailView';
import { useSignalsByEntity, useDismissSignal } from '../../../../src/hooks/useAgentSpec';

function SignalActions({
  onAskCopilot, onDismiss, dismissPending,
}: {
  onAskCopilot: () => void;
  onDismiss: () => void;
  dismissPending: boolean;
}) {
  return (
    <View style={styles.actions}>
      <Button mode="outlined" onPress={onAskCopilot} style={styles.btn} testID="signal-detail-ask-copilot">
        Ask Copilot
      </Button>
      <Button mode="contained" onPress={onDismiss} loading={dismissPending} style={styles.btn}
        testID="signal-detail-dismiss">
        Dismiss
      </Button>
    </View>
  );
}

export default function SignalDetailScreen() {
  const theme = useTheme();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[]; entity_type?: string; entity_id?: string }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const entityType = params.entity_type ?? '';
  const entityId = params.entity_id ?? id;

  const { data: signals, isLoading, error } = useSignalsByEntity(entityType || 'signal', entityId);
  const signal = signals?.find((s) => s.id === id) ?? signals?.[0];
  const dismissMutation = useDismissSignal();

  const handleDismiss = useCallback(() => {
    if (!id) return;
    dismissMutation.mutate(id, { onSuccess: () => router.back() });
  }, [id, dismissMutation, router]);

  const handleAskCopilot = useCallback(() => {
    if (!signal) return;
    router.push({
      pathname: '/(tabs)/copilot',
      params: { entity_type: signal.entity_type, entity_id: signal.entity_id,
        signal_id: signal.id, signal_type: signal.signal_type },
    });
  }, [signal, router]);

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: theme.colors.background }]}>
        <ActivityIndicator size="large" color={theme.colors.primary} />
      </View>
    );
  }

  if (error || !signal) {
    return (
      <View style={[styles.centered, { backgroundColor: theme.colors.background }]}>
        <Text style={{ color: theme.colors.error }}>{error?.message ?? 'Signal not found'}</Text>
      </View>
    );
  }

  return (
    <>
      <Stack.Screen options={{ title: signal.signal_type }} />
      <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
        <SignalDetailView signal={signal} testIDPrefix="signal-detail" />
        <SignalActions onAskCopilot={handleAskCopilot} onDismiss={handleDismiss}
          dismissPending={dismissMutation.isPending} />
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  actions: { flexDirection: 'row', gap: 12, padding: 16, borderTopWidth: StyleSheet.hairlineWidth, borderTopColor: '#ccc' },
  btn: { flex: 1 },
});
