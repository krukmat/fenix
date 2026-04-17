// Task Mobile P1.4 — T4: WorkflowForm component + validateWorkflowForm
import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, TextInput, useTheme } from 'react-native-paper';

type WorkflowFormValue = {
  name: string;
  description: string;
  dslSource: string;
};

type WorkflowFormValidation = {
  name?: string;
  dslSource?: string;
};

interface WorkflowFormProps {
  value: WorkflowFormValue;
  validation: WorkflowFormValidation;
  showValidation: boolean;
  submitError?: string | null;
  readOnlyName?: boolean;
  onChange: (field: keyof WorkflowFormValue, value: string) => void;
}

export function validateWorkflowForm(
  form: WorkflowFormValue,
  requireName = true,
): WorkflowFormValidation {
  const errors: WorkflowFormValidation = {};
  if (requireName && !form.name.trim()) {
    errors.name = 'Name is required';
  }
  if (!form.dslSource.trim()) {
    errors.dslSource = 'DSL source is required';
  }
  return errors;
}

export function WorkflowForm({
  value,
  validation,
  showValidation,
  submitError,
  readOnlyName = false,
  onChange,
}: WorkflowFormProps) {
  const theme = useTheme();

  return (
    <View style={styles.container}>
      {!readOnlyName && (
        <View style={styles.field}>
          <TextInput
            label="Name"
            value={value.name}
            onChangeText={(v) => onChange('name', v)}
            mode="outlined"
            testID="workflow-form-name-input"
            error={showValidation && !!validation.name}
          />
          {showValidation && validation.name ? (
            <Text style={[styles.error, { color: theme.colors.error }]} testID="workflow-form-name-error">
              {validation.name}
            </Text>
          ) : null}
        </View>
      )}

      <View style={styles.field}>
        <TextInput
          label="Description (optional)"
          value={value.description}
          onChangeText={(v) => onChange('description', v)}
          mode="outlined"
          multiline
          numberOfLines={2}
          testID="workflow-form-description-input"
        />
      </View>

      <View style={styles.field}>
        <TextInput
          label="DSL Source"
          value={value.dslSource}
          onChangeText={(v) => onChange('dslSource', v)}
          mode="outlined"
          multiline
          numberOfLines={8}
          testID="workflow-form-dsl-input"
          error={showValidation && !!validation.dslSource}
          style={styles.dslInput}
        />
        {showValidation && validation.dslSource ? (
          <Text style={[styles.error, { color: theme.colors.error }]} testID="workflow-form-dsl-error">
            {validation.dslSource}
          </Text>
        ) : null}
      </View>

      {submitError ? (
        <Text style={[styles.error, styles.submitError, { color: theme.colors.error }]} testID="workflow-form-submit-error">
          {submitError}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { gap: 4 },
  field: { marginBottom: 12 },
  error: { fontSize: 12, marginTop: 4 },
  submitError: { marginTop: 8, textAlign: 'center' },
  dslInput: { fontFamily: 'monospace' },
});
