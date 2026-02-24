import React, { useState } from 'react';
import { ScrollView, StyleSheet, View, Text, Alert } from 'react-native';
import { Button, TextInput, useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { useCreateDeal } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

type DealCreateForm = {
  accountId: string;
  pipelineId: string;
  stageId: string;
  ownerId: string;
  title: string;
};

export function validateNewDealForm(form: DealCreateForm) {
  return {
    accountId: !form.accountId.trim(),
    pipelineId: !form.pipelineId.trim(),
    stageId: !form.stageId.trim(),
    ownerId: !form.ownerId.trim(),
    title: !form.title.trim(),
  };
}

function DealNewFormFields({
  form,
  showValidation,
  validation,
  onChange,
}: {
  form: DealCreateForm;
  showValidation: boolean;
  validation: ReturnType<typeof validateNewDealForm>;
  onChange: (key: keyof DealCreateForm, value: string) => void;
}) {
  return (
    <>
      <TextInput
        testID="deal-new-title-input"
        label="Title"
        mode="outlined"
        value={form.title}
        onChangeText={(value) => onChange('title', value)}
        error={showValidation && validation.title}
        style={styles.input}
      />
      <TextInput
        testID="deal-new-account-id-input"
        label="Account ID"
        mode="outlined"
        value={form.accountId}
        onChangeText={(value) => onChange('accountId', value)}
        error={showValidation && validation.accountId}
        style={styles.input}
      />
      <TextInput
        testID="deal-new-pipeline-id-input"
        label="Pipeline ID"
        mode="outlined"
        value={form.pipelineId}
        onChangeText={(value) => onChange('pipelineId', value)}
        error={showValidation && validation.pipelineId}
        style={styles.input}
      />
      <TextInput
        testID="deal-new-stage-id-input"
        label="Stage ID"
        mode="outlined"
        value={form.stageId}
        onChangeText={(value) => onChange('stageId', value)}
        error={showValidation && validation.stageId}
        style={styles.input}
      />
      <TextInput
        testID="deal-new-owner-id-input"
        label="Owner ID"
        mode="outlined"
        value={form.ownerId}
        onChangeText={(value) => onChange('ownerId', value)}
        error={showValidation && validation.ownerId}
        style={styles.input}
      />
    </>
  );
}

export default function NewDealScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const createDeal = useCreateDeal();
  const [form, setForm] = useState<DealCreateForm>({
    accountId: '',
    pipelineId: '',
    stageId: '',
    ownerId: '',
    title: '',
  });
  const [showValidation, setShowValidation] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const validation = validateNewDealForm(form);
  const hasErrors = Object.values(validation).some(Boolean);

  const onChange = (key: keyof DealCreateForm, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setSubmitError(null);
  };

  const handleSubmit = async () => {
    setShowValidation(true);
    setSubmitError(null);
    if (hasErrors) return;

    try {
      await createDeal.mutateAsync({
        accountId: form.accountId.trim(),
        pipelineId: form.pipelineId.trim(),
        stageId: form.stageId.trim(),
        ownerId: form.ownerId.trim(),
        title: form.title.trim(),
      });
      Alert.alert('Deal created', 'The deal was created successfully.');
      router.back();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : 'Failed to create deal.');
    }
  };

  return (
    <>
      <Stack.Screen options={{ title: 'New Deal' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]} contentContainerStyle={styles.content} testID="deal-new-screen">
        <DealNewFormFields form={form} showValidation={showValidation} validation={validation} onChange={onChange} />

        {submitError && (
          <View style={styles.errorContainer}>
            <Text style={[styles.errorText, { color: colors.error }]}>{submitError}</Text>
          </View>
        )}

        <Button
          testID="deal-new-submit"
          mode="contained"
          onPress={handleSubmit}
          loading={createDeal.isPending}
          disabled={createDeal.isPending}
          style={styles.button}
        >
          Create Deal
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
