import React, { useState } from 'react';
import { Alert, ScrollView, StyleSheet } from 'react-native';
import { Button, useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { WorkflowForm, validateWorkflowForm } from '../../../src/components/workflows/WorkflowForm';
import { useCreateWorkflow } from '../../../src/hooks/useAgentSpec';
import type { ThemeColors } from '../../../src/theme/types';

type WorkflowCreateForm = {
  name: string;
  description: string;
  dslSource: string;
};

export default function WorkflowNewScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const createWorkflow = useCreateWorkflow();
  const [form, setForm] = useState<WorkflowCreateForm>({
    name: '',
    description: '',
    dslSource: '',
  });
  const [showValidation, setShowValidation] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const validation = validateWorkflowForm(form);
  const hasErrors = Object.values(validation).some(Boolean);

  const onChange = (field: keyof WorkflowCreateForm, nextValue: string) => {
    setForm((prev) => ({ ...prev, [field]: nextValue }));
    setSubmitError(null);
  };

  const handleSubmit = async () => {
    setShowValidation(true);
    setSubmitError(null);
    if (hasErrors) return;

    try {
      const result = await createWorkflow.mutateAsync({
        name: form.name.trim(),
        description: form.description.trim() || undefined,
        dsl_source: form.dslSource.trim(),
      });
      Alert.alert('Workflow created', 'The workflow draft was created successfully.');
      router.replace(`/workflows/${result.id}`);
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to create workflow.');
    }
  };

  return (
    <>
      <Stack.Screen options={{ title: 'New Workflow', headerShown: true }} />
      <ScrollView
        style={[styles.container, { backgroundColor: colors.background }]}
        contentContainerStyle={styles.content}
        testID="workflow-new-screen"
      >
        <WorkflowForm
          value={form}
          validation={validation}
          showValidation={showValidation}
          submitError={submitError}
          onChange={onChange}
        />

        <Button
          testID="workflow-new-submit"
          mode="contained"
          onPress={handleSubmit}
          loading={createWorkflow.isPending}
          disabled={createWorkflow.isPending}
          style={styles.button}
        >
          Create Workflow
        </Button>
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: 16 },
  button: { marginTop: 8 },
});
