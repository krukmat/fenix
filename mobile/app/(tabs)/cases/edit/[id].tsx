import React, { useMemo, useState } from 'react';
import { ScrollView, StyleSheet, View, Text, Alert } from 'react-native';
import { Button, TextInput, useTheme } from 'react-native-paper';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { useCase, useUpdateCase } from '../../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../../src/theme/types';

type CaseUpdateForm = {
  status: string;
  priority: string;
  stageId: string;
  ownerId: string;
  description: string;
  metadata: string;
};

function pickString(payload: Record<string, unknown> | undefined, keys: string[]): string {
  for (const key of keys) {
    const value = payload?.[key];
    if (value !== undefined && value !== null) return String(value);
  }
  return '';
}

function initialFormFromPayload(payload: Record<string, unknown> | undefined): CaseUpdateForm {
  return {
    status: pickString(payload, ['status']),
    priority: pickString(payload, ['priority']),
    stageId: pickString(payload, ['stageId', 'stage_id']),
    ownerId: pickString(payload, ['ownerId', 'owner_id']),
    description: pickString(payload, ['description']),
    metadata: pickString(payload, ['metadata']),
  };
}

function CaseEditFormFields({
  form,
  onChange,
}: {
  form: CaseUpdateForm;
  onChange: (key: keyof CaseUpdateForm, value: string) => void;
}) {
  return (
    <>
      <TextInput
        testID="case-edit-status-input"
        label="Status"
        mode="outlined"
        value={form.status}
        onChangeText={(value) => onChange('status', value)}
        style={styles.input}
      />
      <TextInput
        testID="case-edit-priority-input"
        label="Priority"
        mode="outlined"
        value={form.priority}
        onChangeText={(value) => onChange('priority', value)}
        style={styles.input}
      />
      <TextInput
        testID="case-edit-stage-id-input"
        label="Stage ID"
        mode="outlined"
        value={form.stageId}
        onChangeText={(value) => onChange('stageId', value)}
        style={styles.input}
      />
      <TextInput
        testID="case-edit-owner-id-input"
        label="Owner ID"
        mode="outlined"
        value={form.ownerId}
        onChangeText={(value) => onChange('ownerId', value)}
        style={styles.input}
      />
      <TextInput
        testID="case-edit-description-input"
        label="Description"
        mode="outlined"
        value={form.description}
        onChangeText={(value) => onChange('description', value)}
        style={styles.input}
        multiline
      />
      <TextInput
        testID="case-edit-metadata-input"
        label="Metadata (JSON string)"
        mode="outlined"
        value={form.metadata}
        onChangeText={(value) => onChange('metadata', value)}
        style={styles.input}
        multiline
      />
    </>
  );
}

// eslint-disable-next-line complexity
export default function EditCaseScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading } = useCase(id);
  const updateCase = useUpdateCase();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const payload = (data?.data ?? data ?? null) as Record<string, unknown> | null;
  const caseObj = ((payload?.case as Record<string, unknown> | undefined) ?? payload ?? undefined);
  const initial = useMemo(() => initialFormFromPayload(caseObj), [caseObj]);
  const [form, setForm] = useState<CaseUpdateForm>(initial);

  React.useEffect(() => {
    setForm(initial);
  }, [initial]);

  const onChange = (key: keyof CaseUpdateForm, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setSubmitError(null);
  };

  const handleSubmit = async () => {
    if (!id) return;
    setSubmitError(null);
    try {
      await updateCase.mutateAsync({
        id,
        data: {
          status: form.status.trim() || undefined,
          priority: form.priority.trim() || undefined,
          stageId: form.stageId.trim() || undefined,
          ownerId: form.ownerId.trim() || undefined,
          description: form.description.trim() || undefined,
          metadata: form.metadata.trim() || undefined,
        },
      });
      Alert.alert('Case updated', 'The case was updated successfully.');
      router.back();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : 'Failed to update case.');
    }
  };

  return (
    <>
      <Stack.Screen options={{ title: 'Edit Case' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]} contentContainerStyle={styles.content} testID="case-edit-screen">
        <CaseEditFormFields form={form} onChange={onChange} />

        {submitError && (
          <View style={styles.errorContainer}>
            <Text style={[styles.errorText, { color: colors.error }]}>{submitError}</Text>
          </View>
        )}

        <Button
          testID="case-edit-submit"
          mode="contained"
          onPress={handleSubmit}
          loading={isLoading || updateCase.isPending}
          disabled={isLoading || updateCase.isPending || !id}
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
  input: { marginBottom: 12 },
  errorContainer: { marginBottom: 12 },
  errorText: { fontSize: 14 },
  button: { marginTop: 8 },
});
