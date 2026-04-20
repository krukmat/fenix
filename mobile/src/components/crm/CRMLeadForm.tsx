import React, { useEffect, useMemo, useState } from 'react';
import { ScrollView, View } from 'react-native';
import { useRouter } from 'expo-router';
import { normalizeCRMLead } from '../../services/api';
import type { CRMLead } from '../../services/api';
import { useCreateLead, useLead, useUpdateLead } from '../../hooks/useCRM';
import {
  Field,
  FormErrorText,
  LoadingView,
  SubmitButton,
  baseFormStyles,
  record,
  useCRMColors,
} from './CRMFormBase';

type LeadFormValues = {
  name: string;
  email: string;
  company: string;
  source: string;
  status: string;
  score: string;
};

type FieldName = keyof LeadFormValues;
type LeadFormMode = 'create' | 'edit';

const emptyValues: LeadFormValues = {
  name: '',
  email: '',
  company: '',
  source: '',
  status: 'new',
  score: '',
};

function metaText(lead: CRMLead, key: string): string {
  const value = lead.metadata[key];
  return typeof value === 'string' ? value : '';
}

function leadValues(lead: CRMLead): LeadFormValues {
  return {
    name: metaText(lead, 'name'),
    email: metaText(lead, 'email'),
    company: metaText(lead, 'company'),
    source: lead.source ?? '',
    status: lead.status ?? 'new',
    score: lead.score === undefined ? '' : String(lead.score),
  };
}

function validate(values: LeadFormValues): string | null {
  if (!values.name.trim()) return 'Lead name is required';
  if (values.email.trim() && !values.email.includes('@')) return 'Enter a valid email';
  if (values.score.trim() && Number.isNaN(Number(values.score))) return 'Score must be a number';
  return null;
}

function payload(values: LeadFormValues) {
  const metadata = Object.fromEntries(
    [
      ['name', values.name],
      ['email', values.email],
      ['company', values.company],
    ]
      .map(([key, value]) => [key, value.trim()])
      .filter(([, value]) => value !== ''),
  );
  return {
    ...(values.source.trim() ? { source: values.source.trim() } : {}),
    ...(values.status.trim() ? { status: values.status.trim() } : {}),
    ...(values.score.trim() ? { score: Number(values.score) } : {}),
    metadata,
  };
}

export function CRMLeadForm({ mode, leadId }: { mode: LeadFormMode; leadId?: string }) {
  const router = useRouter();
  const colors = useCRMColors();
  const leadQuery = useLead(leadId ?? '');
  const createLead = useCreateLead();
  const updateLead = useUpdateLead();
  const [values, setValues] = useState<LeadFormValues>(emptyValues);
  const [error, setError] = useState<string | null>(null);
  const lead = useMemo(() => {
    const source = record(leadQuery.data)?.lead ?? leadQuery.data;
    return normalizeCRMLead(source);
  }, [leadQuery.data]);
  const loading = mode === 'edit' && leadQuery.isLoading;
  const submitting = createLead.isPending || updateLead.isPending;

  useEffect(() => {
    if (mode === 'edit' && lead.id) setValues(leadValues(lead));
  }, [lead, mode]);

  const setField = (field: FieldName, value: string) => {
    setError(null);
    setValues((current) => ({ ...current, [field]: value }));
  };

  const onSubmit = async () => {
    const validationError = validate(values);
    if (validationError) {
      setError(validationError);
      return;
    }
    const data = payload(values);
    if (mode === 'edit' && leadId) {
      await updateLead.mutateAsync({ id: leadId, data });
      router.replace(`/crm/leads/${leadId}`);
      return;
    }
    await createLead.mutateAsync(data);
    router.replace('/crm/leads');
  };

  if (loading) {
    return <LoadingView testID="crm-lead-form-loading" colors={colors} />;
  }

  return (
    <ScrollView style={[baseFormStyles.container, { backgroundColor: colors.background }]} testID="crm-lead-form-screen">
      <View style={[baseFormStyles.card, { backgroundColor: colors.surface }]}>
        <Field label="Name" value={values.name} onChangeText={(value) => setField('name', value)} testID="crm-lead-form-name" />
        <Field label="Email" value={values.email} onChangeText={(value) => setField('email', value)} testID="crm-lead-form-email" />
        <Field label="Company" value={values.company} onChangeText={(value) => setField('company', value)} testID="crm-lead-form-company" />
        <Field label="Source" value={values.source} onChangeText={(value) => setField('source', value)} testID="crm-lead-form-source" />
        <Field label="Status" value={values.status} onChangeText={(value) => setField('status', value)} testID="crm-lead-form-status" />
        <Field label="Score" value={values.score} onChangeText={(value) => setField('score', value)} testID="crm-lead-form-score" />
        <FormErrorText error={error} style={[baseFormStyles.error, { color: colors.error }]} />
        <SubmitButton
          testID="crm-lead-form-submit"
          onPress={onSubmit}
          disabled={submitting}
          label={mode === 'edit' ? 'Save Lead' : 'Create Lead'}
          colors={colors}
        />
      </View>
    </ScrollView>
  );
}
