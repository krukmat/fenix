import React from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { Button } from 'react-native-paper';
import { spacing } from '../../theme/spacing';
import { WorkflowForm } from './WorkflowForm';
import type { WorkflowFormValidation, WorkflowFormValue } from './WorkflowForm';

interface WorkflowEditorScreenProps {
  backgroundColor: string;
  testID: string;
  submitTestID: string;
  submitLabel: string;
  value: WorkflowFormValue;
  validation: WorkflowFormValidation;
  showValidation: boolean;
  submitError?: string | null;
  readOnlyName?: boolean;
  submitPending: boolean;
  submitDisabled?: boolean;
  onChange: (field: keyof WorkflowFormValue, value: string) => void;
  onSubmit: () => void;
}

export function WorkflowEditorScreen({
  backgroundColor,
  testID,
  submitTestID,
  submitLabel,
  value,
  validation,
  showValidation,
  submitError,
  readOnlyName = false,
  submitPending,
  submitDisabled = false,
  onChange,
  onSubmit,
}: WorkflowEditorScreenProps) {
  return (
    <ScrollView
      style={[styles.container, { backgroundColor }]}
      contentContainerStyle={styles.content}
      testID={testID}
    >
      <WorkflowForm
        value={value}
        validation={validation}
        showValidation={showValidation}
        submitError={submitError}
        readOnlyName={readOnlyName}
        onChange={onChange}
      />

      <Button
        testID={submitTestID}
        mode="contained"
        onPress={onSubmit}
        loading={submitPending}
        disabled={submitPending || submitDisabled}
        style={styles.button}
      >
        {submitLabel}
      </Button>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: spacing.base },
  button: { marginTop: spacing.xs },
});
