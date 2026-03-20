// Task Mobile P1.6 — UC-A5/B4: Signals section for entity detail screens
// B4.1: dismiss from entity detail
// B4.3: graceful degradation when evidence not available

import React, { useCallback } from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, Button, useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { SignalCard } from './SignalCard';
import { useSignalsByEntity, useDismissSignal } from '../../hooks/useAgentSpec';
import type { Signal } from '../../services/api';

interface EntitySignalsSectionProps {
  entityType: string;
  entityId: string;
  testIDPrefix?: string;
}

export function EntitySignalsSection({
  entityType,
  entityId,
  testIDPrefix = 'entity-signals',
}: EntitySignalsSectionProps) {
  const theme = useTheme();
  const router = useRouter();
  const { data: signals, isLoading } = useSignalsByEntity(entityType, entityId);
  const dismissMutation = useDismissSignal();

  const activeSignals = (signals ?? []).filter((s: Signal) => s.status === 'active');

  const handleDismiss = useCallback(
    (id: string) => {
      dismissMutation.mutate(id);
    },
    [dismissMutation]
  );

  const handleAskCopilot = useCallback(() => {
    router.push({
      pathname: '/(tabs)/copilot',
      params: { entity_type: entityType, entity_id: entityId },
    });
  }, [router, entityType, entityId]);

  // Do not render section header if loading or no active signals
  if (isLoading || activeSignals.length === 0) return null;

  return (
    <View style={styles.section} testID={testIDPrefix}>
      <Text
        variant="titleSmall"
        style={[styles.heading, { color: theme.colors.onSurface }]}
        testID={`${testIDPrefix}-heading`}
      >
        {`Signals (${activeSignals.length})`}
      </Text>

      {activeSignals.map((signal: Signal) => (
        <SignalCard
          key={signal.id}
          signal={signal}
          onDismiss={handleDismiss}
          testIDPrefix={`${testIDPrefix}-card-${signal.id}`}
        />
      ))}

      <Button
        mode="outlined"
        onPress={handleAskCopilot}
        style={styles.copilotBtn}
        testID={`${testIDPrefix}-ask-copilot`}
      >
        {`Ask Copilot about this ${entityType}`}
      </Button>
    </View>
  );
}

const styles = StyleSheet.create({
  section: { paddingTop: 8, paddingBottom: 8 },
  heading: { paddingHorizontal: 16, marginBottom: 8, fontWeight: '600' },
  copilotBtn: { marginHorizontal: 16, marginTop: 12 },
});
