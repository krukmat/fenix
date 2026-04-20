import React, { useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { useRouter } from 'expo-router';
import { useTheme } from 'react-native-paper';
import { normalizeCRMAccount, normalizeCRMCase, normalizeCRMContact } from '../../services/api';
import type { CRMCase, CRMContact } from '../../services/api';
import { useAccounts, useCase, useContacts, useCreateCase, useUpdateCase } from '../../hooks/useCRM';
import { useAuthStore } from '../../stores/authStore';
import type { ThemeColors } from '../../theme/types';

type CaseFormValues = {
  accountId: string;
  contactId: string;
  subject: string;
  description: string;
  priority: string;
  status: string;
  channel: string;
};

type FieldName = keyof CaseFormValues;
type CaseFormMode = 'create' | 'edit';

const emptyValues: CaseFormValues = {
  accountId: '',
  contactId: '',
  subject: '',
  description: '',
  priority: 'medium',
  status: 'open',
  channel: '',
};

function useCRMColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function record(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function unwrapDataArray<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[];
  const payload = record(value);
  return Array.isArray(payload?.data) ? (payload.data as T[]) : [];
}

function listItems<T>(data: { pages?: unknown[] } | undefined, normalize: (raw: unknown) => T): T[] {
  return (data?.pages ?? []).flatMap((page) => unwrapDataArray<unknown>(page).map(normalize));
}

function caseValues(caseData: CRMCase): CaseFormValues {
  return {
    accountId: caseData.accountId ?? '',
    contactId: caseData.contactId ?? '',
    subject: caseData.subject,
    description: caseData.description ?? '',
    priority: caseData.priority ?? 'medium',
    status: caseData.status ?? 'open',
    channel: caseData.channel ?? '',
  };
}

function selectedContact(contacts: CRMContact[], contactId: string): CRMContact | undefined {
  return contacts.find((contact) => contact.id === contactId);
}

function validate(values: CaseFormValues, ownerId: string | null, contacts: CRMContact[]): string | null {
  if (!ownerId) return 'Signed-in user is required';
  if (!values.subject.trim()) return 'Case subject is required';
  const contact = selectedContact(contacts, values.contactId);
  if (values.accountId && contact?.accountId && contact.accountId !== values.accountId) {
    return 'Selected contact belongs to another account';
  }
  return null;
}

function payload(values: CaseFormValues, ownerId: string) {
  return {
    ownerId,
    ...Object.fromEntries(
      Object.entries(values)
        .map(([key, value]) => [key, value.trim()])
        .filter(([, value]) => value !== ''),
    ),
  };
}

function Field(props: { label: string; value: string; onChangeText: (value: string) => void; testID: string; multiline?: boolean }) {
  const colors = useCRMColors();
  return (
    <View style={styles.field}>
      <Text style={[styles.label, { color: colors.onSurfaceVariant }]}>{props.label}</Text>
      <TextInput
        testID={props.testID}
        value={props.value}
        onChangeText={props.onChangeText}
        multiline={props.multiline}
        style={[
          styles.input,
          props.multiline ? styles.multiline : null,
          { borderColor: colors.outline, color: colors.onSurface, backgroundColor: colors.surface },
        ]}
      />
    </View>
  );
}

function OptionList<T extends { id: string }>({
  items,
  selectedId,
  label,
  testIDPrefix,
  onSelect,
}: {
  items: T[];
  selectedId: string;
  label: (item: T) => string;
  testIDPrefix: string;
  onSelect: (id: string) => void;
}) {
  const colors = useCRMColors();
  return (
    <View style={styles.optionList}>
      <TouchableOpacity testID={`${testIDPrefix}-none`} style={[styles.option, { borderColor: selectedId ? colors.outline : colors.primary }]} onPress={() => onSelect('')}>
        <Text style={[styles.optionText, { color: colors.onSurface }]}>None</Text>
      </TouchableOpacity>
      {items.map((item) => (
        <TouchableOpacity
          key={item.id}
          testID={`${testIDPrefix}-${item.id}`}
          style={[styles.option, { borderColor: item.id === selectedId ? colors.primary : colors.outline }]}
          onPress={() => onSelect(item.id)}
        >
          <Text style={[styles.optionText, { color: colors.onSurface }]}>{label(item)}</Text>
        </TouchableOpacity>
      ))}
    </View>
  );
}

export function CRMCaseForm({ mode, caseId }: { mode: CaseFormMode; caseId?: string }) {
  const router = useRouter();
  const colors = useCRMColors();
  const ownerId = useAuthStore((state) => state.userId);
  const caseQuery = useCase(caseId ?? '');
  const accountsQuery = useAccounts();
  const contactsQuery = useContacts();
  const createCase = useCreateCase();
  const updateCase = useUpdateCase();
  const [values, setValues] = useState<CaseFormValues>(emptyValues);
  const [error, setError] = useState<string | null>(null);
  const caseData = useMemo(() => normalizeCRMCase(record(caseQuery.data)?.case ?? caseQuery.data), [caseQuery.data]);
  const accounts = useMemo(() => listItems(accountsQuery.data, normalizeCRMAccount), [accountsQuery.data]);
  const contacts = useMemo(() => listItems(contactsQuery.data, normalizeCRMContact), [contactsQuery.data]);
  const loading = (mode === 'edit' && caseQuery.isLoading) || accountsQuery.isLoading || contactsQuery.isLoading;
  const submitting = createCase.isPending || updateCase.isPending;

  useEffect(() => {
    if (mode === 'edit' && caseData.id) setValues(caseValues(caseData));
  }, [caseData, mode]);

  const setField = (field: FieldName, value: string) => {
    setError(null);
    setValues((current) => ({ ...current, [field]: value }));
  };

  const onSubmit = async () => {
    const validationError = validate(values, ownerId, contacts);
    if (validationError || !ownerId) {
      setError(validationError);
      return;
    }
    const data = payload(values, ownerId);
    if (mode === 'edit' && caseId) {
      await updateCase.mutateAsync({ id: caseId, data });
      router.replace(`/crm/cases/${caseId}`);
      return;
    }
    await createCase.mutateAsync(data);
    router.replace('/crm/cases');
  };

  if (loading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="crm-case-form-loading">
        <Text style={{ color: colors.onSurfaceVariant }}>Loading...</Text>
      </View>
    );
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor: colors.background }]} testID="crm-case-form-screen">
      <View style={[styles.card, { backgroundColor: colors.surface }]}>
        <Field label="Subject" value={values.subject} onChangeText={(value) => setField('subject', value)} testID="crm-case-form-subject" />
        <Field label="Description" value={values.description} onChangeText={(value) => setField('description', value)} testID="crm-case-form-description" multiline />
        <Field label="Priority" value={values.priority} onChangeText={(value) => setField('priority', value)} testID="crm-case-form-priority" />
        <Field label="Status" value={values.status} onChangeText={(value) => setField('status', value)} testID="crm-case-form-status" />
        <Field label="Channel" value={values.channel} onChangeText={(value) => setField('channel', value)} testID="crm-case-form-channel" />
        <Text style={[styles.label, { color: colors.onSurfaceVariant }]}>Account</Text>
        <OptionList items={accounts} selectedId={values.accountId} label={(account) => account.name} testIDPrefix="crm-case-form-account" onSelect={(id) => setField('accountId', id)} />
        <Text style={[styles.label, { color: colors.onSurfaceVariant }]}>Contact</Text>
        <OptionList items={contacts} selectedId={values.contactId} label={(contact) => [contact.firstName, contact.lastName].filter(Boolean).join(' ') || contact.email || contact.id} testIDPrefix="crm-case-form-contact" onSelect={(id) => setField('contactId', id)} />
        {error ? <Text style={[styles.error, { color: colors.error }]}>{error}</Text> : null}
        <TouchableOpacity testID="crm-case-form-submit" style={[styles.submit, { backgroundColor: colors.primary }, submitting ? styles.disabled : null]} onPress={onSubmit} disabled={submitting}>
          <Text style={[styles.submitText, { color: colors.onPrimary }]}>{mode === 'edit' ? 'Save Case' : 'Create Case'}</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  card: { margin: 16, padding: 16, borderRadius: 8 },
  field: { marginBottom: 14 },
  label: { fontSize: 13, fontWeight: '600', marginBottom: 6 },
  input: { borderWidth: 1, borderRadius: 8, minHeight: 44, paddingHorizontal: 12, fontSize: 16 },
  multiline: { minHeight: 96, paddingTop: 10, textAlignVertical: 'top' },
  optionList: { gap: 8, marginBottom: 14 },
  option: { borderWidth: 1, borderRadius: 8, padding: 12 },
  optionText: { fontSize: 15, fontWeight: '600' },
  error: { fontSize: 14, marginBottom: 12 },
  submit: { minHeight: 48, borderRadius: 8, alignItems: 'center', justifyContent: 'center' },
  disabled: { opacity: 0.7 },
  submitText: { fontSize: 16, fontWeight: '700' },
});
