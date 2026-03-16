import React, { useEffect, useMemo, useState } from 'react';
import { Alert, ScrollView, StyleSheet, View } from 'react-native';
import { Button, Text, useTheme } from 'react-native-paper';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { WorkflowForm, validateWorkflowForm } from '../../../../src/components/workflows/WorkflowForm';
import { useUpdateWorkflow, useWorkflow } from '../../../../src/hooks/useAgentSpec';
import type { ThemeColors } from '../../../../src/theme/types';

type WorkflowEditForm = {
  name: string;
  description: string;
  dslSource: string;
};

function initialFormFromWorkflow(workflow?: { name?: string; description?: string; dsl_source?: string }): WorkflowEditForm {
  return {
    name: workflow?.name ?? '',
    description: workflow?.description ?? '',
    dslSource: workflow?.dsl_source ?? '',
  };
}

function WorkflowEditUnavailable({
  colors,
  onBack,
}: {
  colors: ThemeColors;
  onBack: () => void;
}) {
  return (
    <>
      <Stack.Screen options={{ title: 'Edit Workflow', headerShown: true }} />
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="workflow-edit-disabled">
        <Text style={{ color: colors.onSurfaceVariant, textAlign: 'center' }}>
          Only draft workflows can be edited on mobile.
        </Text>
        <Button mode="contained" style={styles.button} testID="workflow-edit-back" onPress={onBack}>
          Back to Workflow
        </Button>
      </View>
    </>
  );
}

function getWorkflowEditState(workflow: { status?: string } | undefined, isLoading: boolean): 'missing' | 'disabled' | 'ready' {
  if (!workflow && !isLoading) return 'missing';
  if (workflow && workflow.status !== 'draft') return 'disabled';
  return 'ready';
}

async function submitWorkflowEdit(
  mutateAsync: (input: { id: string; data: { description?: string; dsl_source: string } }) => Promise<unknown>,
  id: string,
  form: WorkflowEditForm
) {
  await mutateAsync({
    id,
    data: {
      description: form.description.trim() || undefined,
      dsl_source: form.dslSource.trim(),
    },
  });
}

function WorkflowEditMissing({ colors }: { colors: ThemeColors }) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]} testID="workflow-edit-missing">
      <Text style={{ color: colors.error }}>Workflow not found</Text>
    </View>
  );
}

export default function WorkflowEditScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;

  const { data: workflow, isLoading } = useWorkflow(id);
  const updateWorkflow = useUpdateWorkflow();

  const initialForm = useMemo(() => initialFormFromWorkflow(workflow), [workflow]);
  const [form, setForm] = useState<WorkflowEditForm>(initialForm);
  const [showValidation, setShowValidation] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    setForm(initialForm);
  }, [initialForm]);

  const validation = validateWorkflowForm(form, false);
  const hasErrors = Object.values(validation).some(Boolean);
  const viewState = getWorkflowEditState(workflow, isLoading);
  const canEdit = viewState === 'ready' && workflow?.status === 'draft';

  const onChange = (field: keyof WorkflowEditForm, nextValue: string) => {
    setForm((prev) => ({ ...prev, [field]: nextValue }));
    setSubmitError(null);
  };

  const handleSubmit = async () => {
    if (!id || !canEdit) return;
    setShowValidation(true);
    setSubmitError(null);
    if (hasErrors) return;

    try {
      await submitWorkflowEdit(updateWorkflow.mutateAsync, id, form);
      Alert.alert('Workflow updated', 'The workflow draft was updated successfully.');
      router.replace(`/workflows/${id}`);
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to update workflow.');
    }
  };

  if (viewState === 'missing') return <WorkflowEditMissing colors={colors} />;
  if (viewState === 'disabled') return <WorkflowEditUnavailable colors={colors} onBack={() => router.replace(`/workflows/${id}`)} />;

  return (
    <>
      <Stack.Screen options={{ title: 'Edit Workflow', headerShown: true }} />
      <ScrollView
        style={[styles.container, { backgroundColor: colors.background }]}
        contentContainerStyle={styles.content}
        testID="workflow-edit-screen"
      >
        <WorkflowForm
          value={form}
          validation={validation}
          showValidation={showValidation}
          readOnlyName
          submitError={submitError}
          onChange={onChange}
        />

        <Button
          testID="workflow-edit-submit"
          mode="contained"
          onPress={handleSubmit}
          loading={isLoading || updateWorkflow.isPending}
          disabled={isLoading || updateWorkflow.isPending || !id || !canEdit}
          style={styles.button}
        >
          Save Changes
        </Button>
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: 16 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  button: { marginTop: 12 },
});
