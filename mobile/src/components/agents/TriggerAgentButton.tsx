// Task 4.5 — FR-230: Trigger Agent Button Component
// Allows manual triggering of agent runs

import React, { useState } from 'react';
import { View, Text, StyleSheet, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { Button, Dialog, Portal, Paragraph, RadioButton } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useAgentDefinitions } from '../../hooks/useCRM';
import { agentApi } from '../../services/api';
import type { ThemeColors } from '../../theme/types';

interface AgentDefinition {
  id: string;
  name: string;
  description?: string;
}

function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

export default function TriggerAgentButton() {
  const colors = useThemeColors();
  const router = useRouter();
  const [showDialog, setShowDialog] = useState(false);
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null);

  const { data: definitionsData, isLoading: isLoadingDefinitions } = useAgentDefinitions();
  const definitions = definitionsData?.data as AgentDefinition[] | undefined;

  const handleTriggerPress = () => {
    setShowDialog(true);
  };

  const handleCloseDialog = () => {
    setShowDialog(false);
    setSelectedAgentId(null);
  };

  const handleAgentSelect = (agentId: string) => {
    setSelectedAgentId(agentId);
  };

  const handleConfirmTrigger = async () => {
    if (!selectedAgentId) return;

    try {
      const result = await agentApi.triggerRun(selectedAgentId, {});
      // Navigate to the new agent run detail screen
      router.push(`/agents/${result.id}`);
      handleCloseDialog();
    } catch (error) {
      console.error('Failed to trigger agent:', error);
      // In a production app, show an error message to the user
    }
  };

  if (isLoadingDefinitions) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="small" color={colors.primary} />
        <Text style={[styles.loadingText, { color: colors.onSurfaceVariant }]}>Loading...</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Button
        mode="contained"
        onPress={handleTriggerPress}
        style={[styles.triggerButton, { backgroundColor: colors.primary }]}
        contentStyle={styles.buttonContent}
        testID="trigger-agent-btn"
      >
        <Text style={styles.triggerButtonText}>Trigger Agent</Text>
      </Button>

      <Portal>
        <Dialog
          visible={showDialog}
          onDismiss={handleCloseDialog}
          testID="trigger-agent-dialog"
        >
          <Dialog.Title>Select Agent</Dialog.Title>
          <Dialog.Content>
            {definitions && definitions.length > 0 ? (
              <View>
                {definitions.map((agent) => (
                  <RadioButton.Item
                    key={agent.id}
                    label={agent.name}
                    value={agent.id}
                    status={selectedAgentId === agent.id ? 'checked' : 'unchecked'}
                    onPress={() => handleAgentSelect(agent.id)}
                    testID={`agent-option-${agent.id}`}
                  />
                ))}
              </View>
            ) : (
              <Paragraph>
                No agent definitions available. Please contact administrator.
              </Paragraph>
            )}
          </Dialog.Content>
          <Dialog.Actions>
            <Button mode="text" onPress={handleCloseDialog} testID="trigger-agent-cancel-btn">
              Cancel
            </Button>
            <Button
              mode="text"
              onPress={handleConfirmTrigger}
              disabled={!selectedAgentId}
              testID="trigger-agent-confirm-btn"
            >
              Trigger
            </Button>
          </Dialog.Actions>
        </Dialog>
      </Portal>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
  },
  loadingContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 16,
  },
  loadingText: {
    marginLeft: 8,
    fontSize: 14,
  },
  triggerButton: {
    borderRadius: 8,
    elevation: 2,
  },
  buttonContent: {
    paddingVertical: 8,
    paddingHorizontal: 16,
  },
  triggerButtonText: {
    fontSize: 16,
    fontWeight: '600',
  },
});
