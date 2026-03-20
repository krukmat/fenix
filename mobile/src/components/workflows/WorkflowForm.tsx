import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Text, TextInput, useTheme } from 'react-native-paper';

type WorkflowFormValue = {
  name: string;
  description: string;
  dslSource: string;
};

type WorkflowFormValidation = {
  name: boolean;
  dslSource: boolean;
};

interface WorkflowFormProps {
  value: WorkflowFormValue;
  validation: WorkflowFormValidation;
  showValidation: boolean;
  readOnlyName?: boolean;
  submitError?: string | null;
  onChange: (field: keyof WorkflowFormValue, nextValue: string) => void;
}

export function WorkflowForm({
  value,
  validation,
  showValidation,
  readOnlyName = false,
  submitError,
  onChange,
}: WorkflowFormProps) {
  const theme = useTheme();

  return (
    <>
      <TextInput
        testID="workflow-form-name-input"
        label="Name"
        mode="outlined"
        value={value.name}
        onChangeText={(nextValue) => onChange('name', nextValue)}
        error={showValidation && validation.name}
        disabled={readOnlyName}
        style={styles.input}
      />
      <TextInput
        testID="workflow-form-description-input"
        label="Description"
        mode="outlined"
        value={value.description}
        onChangeText={(nextValue) => onChange('description', nextValue)}
        multiline
        style={styles.input}
      />
      <TextInput
        testID="workflow-form-dsl-input"
        label="DSL Source"
        mode="outlined"
        value={value.dslSource}
        onChangeText={(nextValue) => onChange('dslSource', nextValue)}
        error={showValidation && validation.dslSource}
        multiline
        numberOfLines={8}
        style={styles.input}
      />

      {submitError ? (
        <View style={styles.errorContainer}>
          <Text style={[styles.errorText, { color: theme.colors.error }]}>{submitError}</Text>
        </View>
      ) : null}
    </>
  );
}

export function validateWorkflowForm(value: WorkflowFormValue, requireName = true): WorkflowFormValidation {
  return {
    name: requireName ? !value.name.trim() : false,
    dslSource: !value.dslSource.trim(),
  };
}

const styles = StyleSheet.create({
  input: { marginBottom: 12 },
  errorContainer: { marginBottom: 12 },
  errorText: { fontSize: 14 },
});
