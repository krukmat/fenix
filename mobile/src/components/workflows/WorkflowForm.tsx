// Task Mobile P1.4 — T4: WorkflowForm component + validateWorkflowForm
import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, TextInput, useTheme } from 'react-native-paper';
import { spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

export type WorkflowFormValue = {
  name: string;
  description: string;
  dslSource: string;
};

export type WorkflowFormValidation = {
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

function WorkflowError({
  visible,
  message,
  testID,
}: {
  visible: boolean;
  message?: string;
  testID: string;
}) {
  const theme = useTheme();
  if (!visible || !message) {
    return null;
  }
  return (
    <Text style={[styles.error, { color: theme.colors.error }]} testID={testID}>
      {message}
    </Text>
  );
}

function WorkflowTextField({
  label,
  value,
  onChangeText,
  testID,
  error,
  multiline = false,
  numberOfLines,
  style,
}: {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  testID: string;
  error?: boolean;
  multiline?: boolean;
  numberOfLines?: number;
  style?: object;
}) {
  return (
    <TextInput
      label={label}
      value={value}
      onChangeText={onChangeText}
      mode="outlined"
      multiline={multiline}
      numberOfLines={numberOfLines}
      testID={testID}
      error={error}
      style={style}
    />
  );
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
  return (
    <View style={styles.container}>
      {!readOnlyName && (
        <View style={styles.field}>
          <WorkflowTextField
            label="Name"
            value={value.name}
            onChangeText={(v) => onChange('name', v)}
            testID="workflow-form-name-input"
            error={showValidation && !!validation.name}
          />
          <WorkflowError visible={showValidation} message={validation.name} testID="workflow-form-name-error" />
        </View>
      )}

      <View style={styles.field}>
        <WorkflowTextField
          label="Description (optional)"
          value={value.description}
          onChangeText={(v) => onChange('description', v)}
          multiline
          numberOfLines={2}
          testID="workflow-form-description-input"
        />
      </View>

      <View style={styles.field}>
        <WorkflowTextField
          label="DSL Source"
          value={value.dslSource}
          onChangeText={(v) => onChange('dslSource', v)}
          multiline
          numberOfLines={8}
          testID="workflow-form-dsl-input"
          error={showValidation && !!validation.dslSource}
          style={styles.dslInput}
        />
        <WorkflowError visible={showValidation} message={validation.dslSource} testID="workflow-form-dsl-error" />
      </View>

      {submitError ? (
        <Text style={[styles.error, styles.submitError]} testID="workflow-form-submit-error">
          {submitError}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { gap: spacing.xs },
  field: { marginBottom: spacing.md },
  error: { fontSize: 12, marginTop: spacing.xs, color: '#B3261E' },
  submitError: { marginTop: spacing.sm, textAlign: 'center' },
  dslInput: typography.mono,
});
