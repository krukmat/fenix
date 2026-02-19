import React, { useState } from 'react';
import { Button, Dialog, Portal, Text } from 'react-native-paper';

export interface SuggestedAction {
  label: string;
  tool: string;
  params: Record<string, unknown>;
}

interface ActionButtonProps {
  action: SuggestedAction;
  onExecute: (action: SuggestedAction) => Promise<void>;
  testIDPrefix?: string;
}

export function ActionButton({ action, onExecute, testIDPrefix = 'action' }: ActionButtonProps) {
  const [visible, setVisible] = useState(false);
  const [loading, setLoading] = useState(false);

  const openDialog = () => setVisible(true);
  const closeDialog = () => {
    if (!loading) setVisible(false);
  };

  const confirm = async () => {
    setLoading(true);
    try {
      await onExecute(action);
      setVisible(false);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Button
        mode="outlined"
        onPress={openDialog}
        loading={loading}
        disabled={loading}
        testID={`${testIDPrefix}-btn`}
      >
        {action.label}
      </Button>

      <Portal>
        <Dialog visible={visible} onDismiss={closeDialog} testID={`${testIDPrefix}-dialog`}>
          <Dialog.Title>Confirm action</Dialog.Title>
          <Dialog.Content>
            <Text>{`Execute: ${action.label}?`}</Text>
          </Dialog.Content>
          <Dialog.Actions>
            <Button onPress={closeDialog} disabled={loading} testID={`${testIDPrefix}-cancel`}>
              Cancel
            </Button>
            <Button onPress={confirm} loading={loading} disabled={loading} testID={`${testIDPrefix}-confirm`}>
              Confirm
            </Button>
          </Dialog.Actions>
        </Dialog>
      </Portal>
    </>
  );
}
