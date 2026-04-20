import React, { useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { useRouter } from 'expo-router';
import { useTheme } from 'react-native-paper';
import { useCreateDeal, useDeal, useUpdateDeal } from '../../hooks/useCRM';
import { useAuthStore } from '../../stores/authStore';
import type { ThemeColors } from '../../theme/types';
import { normalizeCRMDeal } from '../../services/api';
import type { CRMDeal } from '../../services/api';
import {
  CRMDealSelectors,
  emptyDealSelectorValues,
  validateDealSelectors,
  type CRMDealSelectorData,
  type CRMDealSelectorValues,
} from './CRMDealSelectors';

type DealCreateValues = CRMDealSelectorValues & {
  title: string;
  amount: string;
  currency: string;
  expectedClose: string;
  status: string;
};

type DealCreateField = keyof DealCreateValues;
type DealFormMode = 'create' | 'edit';

const emptyData: CRMDealSelectorData = {
  accounts: [],
  contacts: [],
  pipelines: [],
  stages: [],
};

const emptyValues: DealCreateValues = {
  ...emptyDealSelectorValues,
  title: '',
  amount: '',
  currency: 'USD',
  expectedClose: '',
  status: 'open',
};

function useCRMColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function selectorValues(values: DealCreateValues): CRMDealSelectorValues {
  return {
    accountId: values.accountId,
    contactId: values.contactId,
    pipelineId: values.pipelineId,
    stageId: values.stageId,
  };
}

function dealValues(deal: CRMDeal): DealCreateValues {
  return {
    accountId: deal.accountId,
    contactId: deal.contactId ?? '',
    pipelineId: deal.pipelineId,
    stageId: deal.stageId,
    title: deal.title,
    amount: deal.amount === undefined ? '' : String(deal.amount),
    currency: deal.currency ?? 'USD',
    expectedClose: deal.expectedClose ?? '',
    status: deal.status ?? 'open',
  };
}

function validate(values: DealCreateValues, ownerId: string | null, data: CRMDealSelectorData): string | null {
  if (!ownerId) return 'Signed-in user is required';
  if (!values.title.trim()) return 'Deal title is required';
  if (values.amount.trim() && Number.isNaN(Number(values.amount))) return 'Amount must be a number';
  return validateDealSelectors(selectorValues(values), data.contacts, data.stages);
}

function record(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function payload(values: DealCreateValues, ownerId: string) {
  return {
    ownerId,
    accountId: values.accountId,
    pipelineId: values.pipelineId,
    stageId: values.stageId,
    title: values.title.trim(),
    ...(values.contactId ? { contactId: values.contactId } : {}),
    ...(values.amount.trim() ? { amount: Number(values.amount) } : {}),
    ...(values.currency.trim() ? { currency: values.currency.trim() } : {}),
    ...(values.expectedClose.trim() ? { expectedClose: values.expectedClose.trim() } : {}),
    ...(values.status.trim() ? { status: values.status.trim() } : {}),
  };
}

function Field(props: { label: string; value: string; onChangeText: (value: string) => void; testID: string }) {
  const colors = useCRMColors();
  return (
    <View style={styles.field}>
      <Text style={[styles.label, { color: colors.onSurfaceVariant }]}>{props.label}</Text>
      <TextInput
        testID={props.testID}
        value={props.value}
        onChangeText={props.onChangeText}
        style={[styles.input, { borderColor: colors.outline, color: colors.onSurface, backgroundColor: colors.surface }]}
      />
    </View>
  );
}

function CRMDealForm({ mode, dealId }: { mode: DealFormMode; dealId?: string }) {
  const router = useRouter();
  const colors = useCRMColors();
  const ownerId = useAuthStore((state) => state.userId);
  const dealQuery = useDeal(dealId ?? '');
  const createDeal = useCreateDeal();
  const updateDeal = useUpdateDeal();
  const [values, setValues] = useState<DealCreateValues>(emptyValues);
  const [selectorData, setSelectorData] = useState<CRMDealSelectorData>(emptyData);
  const [error, setError] = useState<string | null>(null);
  const deal = useMemo(() => normalizeCRMDeal(record(dealQuery.data)?.deal ?? dealQuery.data), [dealQuery.data]);
  const loading = mode === 'edit' && dealQuery.isLoading;
  const submitting = createDeal.isPending || updateDeal.isPending;

  useEffect(() => {
    if (mode === 'edit' && deal.id) setValues(dealValues(deal));
  }, [deal, mode]);

  const setField = (field: DealCreateField, value: string) => {
    setError(null);
    setValues((current) => ({
      ...current,
      [field]: value,
      ...(field === 'pipelineId' ? { stageId: '' } : {}),
    }));
  };

  const onSubmit = async () => {
    const validationError = validate(values, ownerId, selectorData);
    if (validationError || !ownerId) {
      setError(validationError);
      return;
    }
    const data = payload(values, ownerId);
    if (mode === 'edit' && dealId) {
      await updateDeal.mutateAsync({ id: dealId, data });
      router.replace(`/crm/deals/${dealId}`);
      return;
    }
    await createDeal.mutateAsync(data);
    router.replace('/crm/deals');
  };

  if (loading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="crm-deal-form-loading">
        <Text style={{ color: colors.onSurfaceVariant }}>Loading...</Text>
      </View>
    );
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor: colors.background }]} testID={`crm-deal-${mode}-form-screen`}>
      <View style={[styles.card, { backgroundColor: colors.surface }]}>
        <Field label="Title" value={values.title} onChangeText={(value) => setField('title', value)} testID="crm-deal-form-title" />
        <Field label="Amount" value={values.amount} onChangeText={(value) => setField('amount', value)} testID="crm-deal-form-amount" />
        <Field label="Currency" value={values.currency} onChangeText={(value) => setField('currency', value)} testID="crm-deal-form-currency" />
        <Field label="Expected close" value={values.expectedClose} onChangeText={(value) => setField('expectedClose', value)} testID="crm-deal-form-expected-close" />
        <Field label="Status" value={values.status} onChangeText={(value) => setField('status', value)} testID="crm-deal-form-status" />
        <CRMDealSelectors values={selectorValues(values)} onChange={setField} onDataChange={setSelectorData} />
        {error ? <Text style={[styles.error, { color: colors.error }]}>{error}</Text> : null}
        <TouchableOpacity
          testID="crm-deal-form-submit"
          style={[styles.submit, { backgroundColor: colors.primary }, submitting ? styles.disabled : null]}
          onPress={onSubmit}
          disabled={submitting}
        >
          <Text style={[styles.submitText, { color: colors.onPrimary }]}>{mode === 'edit' ? 'Save Deal' : 'Create Deal'}</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

export function CRMDealCreateForm() {
  return <CRMDealForm mode="create" />;
}

export function CRMDealEditForm({ dealId }: { dealId: string }) {
  return <CRMDealForm mode="edit" dealId={dealId} />;
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  card: { margin: 16, padding: 16, borderRadius: 8 },
  field: { marginBottom: 14 },
  label: { fontSize: 13, fontWeight: '600', marginBottom: 6 },
  input: { borderWidth: 1, borderRadius: 8, minHeight: 44, paddingHorizontal: 12, fontSize: 16 },
  error: { fontSize: 14, marginBottom: 12 },
  submit: { minHeight: 48, borderRadius: 8, alignItems: 'center', justifyContent: 'center' },
  disabled: { opacity: 0.7 },
  submitText: { fontSize: 16, fontWeight: '700' },
});
