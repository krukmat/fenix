import React, { useState } from 'react';
import { ScrollView, StyleSheet, View, Text, Alert } from 'react-native';
import { Button, TextInput, useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { useCreateCase } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

type CaseCreateForm = {
  ownerId: string;
  subject: string;
  description: string;
};

export function validateNewCaseForm(form: CaseCreateForm) {
  return {
    ownerId: !form.ownerId.trim(),
    subject: !form.subject.trim(),
  };
}

function CaseNewFormFields({
  form,
  showValidation,
  validation,
  onChange,
}: {
  form: CaseCreateForm;
  showValidation: boolean;
  validation: ReturnType<typeof validateNewCaseForm>;
  onChange: (key: keyof CaseCreateForm, value: string) => void;
}) {
  return (
    <>
      <TextInput
        testID="case-new-subject-input"
        label="Subject"
        mode="outlined"
        value={form.subject}
        onChangeText={(value) => onChange('subject', value)}
        error={showValidation && validation.subject}
        style={styles.input}
      />
      <TextInput
        testID="case-new-owner-id-input"
        label="Owner ID"
        mode="outlined"
        value={form.ownerId}
        onChangeText={(value) => onChange('ownerId', value)}
        error={showValidation && validation.ownerId}
        style={styles.input}
      />
      <TextInput
        testID="case-new-description-input"
        label="Description"
        mode="outlined"
        value={form.description}
        onChangeText={(value) => onChange('description', value)}
        multiline
        style={styles.input}
      />
    </>
  );
}

export default function NewCaseScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const createCase = useCreateCase();
  const [form, setForm] = useState<CaseCreateForm>({
    ownerId: '',
    subject: '',
    description: '',
  });
  const [showValidation, setShowValidation] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const validation = validateNewCaseForm(form);
  const hasErrors = Object.values(validation).some(Boolean);

  const onChange = (key: keyof CaseCreateForm, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setSubmitError(null);
  };

  const handleSubmit = async () => {
    setShowValidation(true);
    setSubmitError(null);
    if (hasErrors) return;

    try {
      await createCase.mutateAsync({
        ownerId: form.ownerId.trim(),
        subject: form.subject.trim(),
        description: form.description.trim() || undefined,
      });
      Alert.alert('Case created', 'The case was created successfully.');
      router.back();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : 'Failed to create case.');
    }
  };

  return (
    <>
      <Stack.Screen options={{ title: 'New Case' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]} contentContainerStyle={styles.content} testID="case-new-screen">
        <CaseNewFormFields form={form} showValidation={showValidation} validation={validation} onChange={onChange} />

        {submitError && (
          <View style={styles.errorContainer}>
            <Text style={[styles.errorText, { color: colors.error }]}>{submitError}</Text>
          </View>
        )}

        <Button
          testID="case-new-submit"
          mode="contained"
          onPress={handleSubmit}
          loading={createCase.isPending}
          disabled={createCase.isPending}
          style={styles.button}
        >
          Create Case
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
