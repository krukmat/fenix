// Task 4.5 — FR-230: Trigger Agent Button Component
// Allows manual triggering of agent runs

import React, { useState } from 'react';
import { View, Text, StyleSheet, ActivityIndicator } from 'react-native';
import { useTheme, Button, Dialog, Portal, Paragraph, RadioButton } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useAgentDefinitions } from '../../hooks/useCRM';
import { agentApi } from '../../services/api';
import type { ThemeColors } from '../../theme/types';

interface AgentDefinition {
  id: string;
  name: string;
  description?: string;
}

interface TriggerAgentDialogProps {
  visible: boolean;
  definitions: AgentDefinition[] | undefined;
  selectedAgentId: string | null;
  onDismiss: () => void;
  onSelect: (agentId: string) => void;
  onConfirm: () => void;
}

function TriggerAgentDialog({
  visible,
  definitions,
  selectedAgentId,
  onDismiss,
  onSelect,
  onConfirm,
}: TriggerAgentDialogProps) {
  return (
    <Portal>
      <Dialog visible={visible} onDismiss={onDismiss} testID="trigger-agent-dialog">
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
                  onPress={() => onSelect(agent.id)}
                  testID={`agent-option-${agent.id}`}
                />
              ))}
            </View>
          ) : (
            <Paragraph>No agent definitions available. Please contact administrator.</Paragraph>
          )}
        </Dialog.Content>
        <Dialog.Actions>
          <Button mode="text" onPress={onDismiss} testID="trigger-agent-cancel-btn">
            Cancel
          </Button>
          <Button
            mode="text"
            onPress={onConfirm}
            disabled={!selectedAgentId}
            testID="trigger-agent-confirm-btn"
          >
            Trigger
          </Button>
        </Dialog.Actions>
      </Dialog>
    </Portal>
  );
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

  const handleCloseDialog = () => {
    setShowDialog(false);
    setSelectedAgentId(null);
  };

  const handleConfirmTrigger = async () => {
    if (!selectedAgentId) return;
    try {
      const result = await agentApi.triggerRun(selectedAgentId, {});
      router.push(`/agents/${result.id}`);
      handleCloseDialog();
    } catch (error) {
      console.error('Failed to trigger agent:', error);
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
        onPress={() => setShowDialog(true)}
        style={[styles.triggerButton, { backgroundColor: colors.primary }]}
        contentStyle={styles.buttonContent}
        testID="trigger-agent-btn"
      >
        <Text style={styles.triggerButtonText}>Trigger Agent</Text>
      </Button>
      <TriggerAgentDialog
        visible={showDialog}
        definitions={definitions}
        selectedAgentId={selectedAgentId}
        onDismiss={handleCloseDialog}
        onSelect={setSelectedAgentId}
        onConfirm={handleConfirmTrigger}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { padding: 16 },
  loadingContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 16,
  },
  loadingText: { marginLeft: 8, fontSize: 14 },
  triggerButton: { borderRadius: 8, elevation: 2 },
  buttonContent: { paddingVertical: 8, paddingHorizontal: 16 },
  triggerButtonText: { fontSize: 16, fontWeight: '600' },
});
