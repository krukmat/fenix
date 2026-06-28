import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'expo-router';
import { normalizeCRMAccount, normalizeCRMCase, normalizeCRMContact } from '../../services/api';
import type { CRMCase, CRMContact } from '../../services/api';
import { useAccounts, useCase, useContacts, useCreateCase, useUpdateCase } from '../../hooks/useCRM';
import { useAuthStore } from '../../stores/authStore';
import {
  Field,
  FormScreen,
  FormSectionLabel,
  FormErrorText,
  LoadingView,
  OptionButtonItem,
  OptionButtonList,
  SubmitButton,
  baseFormStyles,
  listItems,
  record,
  useCRMColors,
} from './CRMFormBase';

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

function caseOptionItems<T extends { id: string }>(items: T[], label: (item: T) => string): OptionButtonItem[] {
  return items.map((item) => ({ id: item.id, label: label(item) }));
}

function CaseFields({
  values,
  accounts,
  contacts,
  onSelect,
}: {
  values: CaseFormValues;
  accounts: OptionButtonItem[];
  contacts: OptionButtonItem[];
  onSelect: (field: FieldName, value: string) => void;
}) {
  return (
    <>
      <Field label="Subject" value={values.subject} onChangeText={(value) => onSelect('subject', value)} testID="crm-case-form-subject" />
      <Field label="Description" value={values.description} onChangeText={(value) => onSelect('description', value)} testID="crm-case-form-description" multiline />
      <Field label="Priority" value={values.priority} onChangeText={(value) => onSelect('priority', value)} testID="crm-case-form-priority" />
      <Field label="Status" value={values.status} onChangeText={(value) => onSelect('status', value)} testID="crm-case-form-status" />
      <Field label="Channel" value={values.channel} onChangeText={(value) => onSelect('channel', value)} testID="crm-case-form-channel" />
      <FormSectionLabel>Account</FormSectionLabel>
      <OptionButtonList items={accounts} selectedId={values.accountId} testIDPrefix="crm-case-form-account" onSelect={(id) => onSelect('accountId', id)} noneLabel="None" />
      <FormSectionLabel>Contact</FormSectionLabel>
      <OptionButtonList items={contacts} selectedId={values.contactId} testIDPrefix="crm-case-form-contact" onSelect={(id) => onSelect('contactId', id)} noneLabel="None" />
    </>
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
  const accountOptions = useMemo(() => caseOptionItems(accounts, (account) => account.name), [accounts]);
  const contactOptions = useMemo(
    () => caseOptionItems(contacts, (contact) => [contact.firstName, contact.lastName].filter(Boolean).join(' ') || contact.email || contact.id),
    [contacts],
  );
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
    return <LoadingView testID="crm-case-form-loading" colors={colors} />;
  }

  return (
    <FormScreen testID="crm-case-form-screen" colors={colors}>
      <CaseFields values={values} accounts={accountOptions} contacts={contactOptions} onSelect={setField} />
      <FormErrorText error={error} style={[baseFormStyles.error, { color: colors.error }]} />
      <SubmitButton
        testID="crm-case-form-submit"
        onPress={onSubmit}
        disabled={submitting}
        label={mode === 'edit' ? 'Save Case' : 'Create Case'}
        colors={colors}
      />
    </FormScreen>
  );
}
