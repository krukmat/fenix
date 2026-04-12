// UC-A5/B4: Signal detail screen
// FR-200: signal context forwarding to Copilot

import React, { useCallback } from 'react';
import { View, ActivityIndicator, StyleSheet } from 'react-native';
import { Text, Button, useTheme } from 'react-native-paper';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { SignalDetailView } from '../../../../src/components/signals/SignalDetailView';
import { useSignalsByEntity, useDismissSignal } from '../../../../src/hooks/useAgentSpec';
import type { Signal } from '../../../../src/services/api';

type Params = { id: string | string[]; entity_type?: string; entity_id?: string };

function resolveParams(params: Params): { id: string; entityType: string; entityId: string } {
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const entityType = params.entity_type ?? '';
  const entityId = params.entity_id ?? id;
  return { id, entityType, entityId };
}

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

function useSignalHandlers(id: string, signal: Signal | undefined) {
  const router = useRouter();
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

  return { handleDismiss, handleAskCopilot, dismissPending: dismissMutation.isPending };
}

function selectSignal(signals: Signal[] | undefined, id: string): Signal | undefined {
  if (!Array.isArray(signals) || signals.length === 0) {
    return undefined;
  }
  return signals.find((item) => item.id === id) ?? signals[0];
}

export default function SignalDetailScreen() {
  const theme = useTheme();
  const rawParams = useLocalSearchParams<Params>();
  const { id, entityType, entityId } = resolveParams(rawParams);

  const { data: signals, isLoading, error } = useSignalsByEntity(entityType || 'signal', entityId);
  const signal = selectSignal(signals, id);
  const { handleDismiss, handleAskCopilot, dismissPending } = useSignalHandlers(id, signal);

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
          dismissPending={dismissPending} />
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
