import React, { useState } from 'react';
import { Alert } from 'react-native';
import { useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { WorkflowEditorScreen } from '../../../src/components/workflows/WorkflowEditorScreen';
import { validateWorkflowForm } from '../../../src/components/workflows/WorkflowForm';
import type { WorkflowFormValue } from '../../../src/components/workflows/WorkflowForm';
import { useCreateWorkflow } from '../../../src/hooks/useAgentSpec';
import type { ThemeColors } from '../../../src/theme/types';

export default function WorkflowNewScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const createWorkflow = useCreateWorkflow();
  const [form, setForm] = useState<WorkflowFormValue>({
    name: '',
    description: '',
    dslSource: '',
  });
  const [showValidation, setShowValidation] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const validation = validateWorkflowForm(form);
  const hasErrors = Object.values(validation).some(Boolean);

  const onChange = (field: keyof WorkflowFormValue, nextValue: string) => {
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
      <WorkflowEditorScreen
        backgroundColor={colors.background}
        testID="workflow-new-screen"
        submitTestID="workflow-new-submit"
        submitLabel="Create Workflow"
        value={form}
        validation={validation}
        showValidation={showValidation}
        submitError={submitError}
        submitPending={createWorkflow.isPending}
        onChange={onChange}
        onSubmit={handleSubmit}
      />
    </>
  );
}
