import React, { useMemo, useState } from 'react';
import { ScrollView, StyleSheet, View, Text, Alert } from 'react-native';
import { Button, TextInput, useTheme } from 'react-native-paper';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { useDeal, useUpdateDeal } from '../../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../../src/theme/types';

type DealUpdateForm = {
  status: string;
  stageId: string;
  ownerId: string;
  amount: string;
  expectedClose: string;
  metadata: string;
};

function pickString(payload: Record<string, unknown> | undefined, keys: string[]): string {
  for (const key of keys) {
    const value = payload?.[key];
    if (value !== undefined && value !== null) return String(value);
  }
  return '';
}

function initialFormFromPayload(payload: Record<string, unknown> | undefined): DealUpdateForm {
  return {
    status: pickString(payload, ['status']),
    stageId: pickString(payload, ['stageId', 'stage_id']),
    ownerId: pickString(payload, ['ownerId', 'owner_id']),
    amount: pickString(payload, ['amount']),
    expectedClose: pickString(payload, ['expectedClose', 'expected_close']),
    metadata: pickString(payload, ['metadata']),
  };
}

function parseOptionalNumber(value: string): number | undefined {
  if (!value.trim()) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function DealEditFormFields({
  form,
  onChange,
}: {
  form: DealUpdateForm;
  onChange: (key: keyof DealUpdateForm, value: string) => void;
}) {
  return (
    <>
      <TextInput
        testID="deal-edit-status-input"
        label="Status"
        mode="outlined"
        value={form.status}
        onChangeText={(value) => onChange('status', value)}
        style={styles.input}
      />
      <TextInput
        testID="deal-edit-stage-id-input"
        label="Stage ID"
        mode="outlined"
        value={form.stageId}
        onChangeText={(value) => onChange('stageId', value)}
        style={styles.input}
      />
      <TextInput
        testID="deal-edit-owner-id-input"
        label="Owner ID"
        mode="outlined"
        value={form.ownerId}
        onChangeText={(value) => onChange('ownerId', value)}
        style={styles.input}
      />
      <TextInput
        testID="deal-edit-amount-input"
        label="Amount"
        keyboardType="numeric"
        mode="outlined"
        value={form.amount}
        onChangeText={(value) => onChange('amount', value)}
        style={styles.input}
      />
      <TextInput
        testID="deal-edit-expected-close-input"
        label="Expected Close (RFC3339)"
        mode="outlined"
        value={form.expectedClose}
        onChangeText={(value) => onChange('expectedClose', value)}
        style={styles.input}
      />
      <TextInput
        testID="deal-edit-metadata-input"
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
export default function EditDealScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading } = useDeal(id);
  const updateDeal = useUpdateDeal();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const payload = (data?.data ?? data ?? null) as Record<string, unknown> | null;
  const dealObj = ((payload?.deal as Record<string, unknown> | undefined) ?? payload ?? undefined);
  const initial = useMemo(() => initialFormFromPayload(dealObj), [dealObj]);
  const [form, setForm] = useState<DealUpdateForm>(initial);

  React.useEffect(() => {
    setForm(initial);
  }, [initial]);

  const onChange = (key: keyof DealUpdateForm, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setSubmitError(null);
  };

  const handleSubmit = async () => {
    if (!id) return;
    setSubmitError(null);
    try {
      await updateDeal.mutateAsync({
        id,
        data: {
          status: form.status.trim() || undefined,
          stageId: form.stageId.trim() || undefined,
          ownerId: form.ownerId.trim() || undefined,
          amount: parseOptionalNumber(form.amount),
          expectedClose: form.expectedClose.trim() || undefined,
          metadata: form.metadata.trim() || undefined,
        },
      });
      Alert.alert('Deal updated', 'The deal was updated successfully.');
      router.back();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : 'Failed to update deal.');
    }
  };

  return (
    <>
      <Stack.Screen options={{ title: 'Edit Deal' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]} contentContainerStyle={styles.content} testID="deal-edit-screen">
        <DealEditFormFields form={form} onChange={onChange} />

        {submitError && (
          <View style={styles.errorContainer}>
            <Text style={[styles.errorText, { color: colors.error }]}>{submitError}</Text>
          </View>
        )}

        <Button
          testID="deal-edit-submit"
          mode="contained"
          onPress={handleSubmit}
          loading={isLoading || updateDeal.isPending}
          disabled={isLoading || updateDeal.isPending || !id}
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
