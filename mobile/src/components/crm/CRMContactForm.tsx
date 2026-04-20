import React, { useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useRouter } from 'expo-router';
import {
  normalizeCRMAccount,
  normalizeCRMContact,
} from '../../services/api';
import type { CRMAccount, CRMContact } from '../../services/api';
import { useAccounts, useContact, useCreateContact, useUpdateContact } from '../../hooks/useCRM';
import {
  Field,
  FormErrorText,
  LoadingView,
  SubmitButton,
  baseFormStyles,
  listItems,
  record,
  useCRMColors,
} from './CRMFormBase';

type ContactFormValues = {
  accountId: string;
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  title: string;
};

type FieldName = keyof ContactFormValues;
type ContactFormMode = 'create' | 'edit';

const emptyValues: ContactFormValues = {
  accountId: '',
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  title: '',
};

function listAccounts(data: ReturnType<typeof useAccounts>['data']): CRMAccount[] {
  return listItems(data, normalizeCRMAccount);
}

function contactValues(contact: CRMContact): ContactFormValues {
  return {
    accountId: contact.accountId ?? '',
    firstName: contact.firstName ?? '',
    lastName: contact.lastName ?? '',
    email: contact.email ?? '',
    phone: contact.phone ?? '',
    title: contact.title ?? '',
  };
}

function validate(values: ContactFormValues): string | null {
  if (!values.accountId.trim()) return 'Account is required';
  if (!values.firstName.trim() && !values.lastName.trim() && !values.email.trim()) {
    return 'Add a name or email';
  }
  if (values.email.trim() && !values.email.includes('@')) return 'Enter a valid email';
  return null;
}

function payload(values: ContactFormValues) {
  return Object.fromEntries(
    Object.entries(values)
      .map(([key, value]) => [key, value.trim()])
      .filter(([, value]) => value !== ''),
  ) as Partial<ContactFormValues>;
}

function AccountPicker({
  accounts,
  selectedId,
  onSelect,
}: {
  accounts: CRMAccount[];
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  const colors = useCRMColors();
  if (accounts.length === 0) {
    return <Text style={[styles.help, { color: colors.onSurfaceVariant }]}>No accounts loaded</Text>;
  }
  return (
    <View style={styles.accountList}>
      {accounts.map((account) => {
        const selected = account.id === selectedId;
        return (
          <TouchableOpacity
            key={account.id}
            testID={`crm-contact-form-account-${account.id}`}
            style={[styles.accountOption, { borderColor: selected ? colors.primary : colors.outline }]}
            onPress={() => onSelect(account.id)}
          >
            <Text style={[styles.accountName, { color: colors.onSurface }]}>{account.name}</Text>
          </TouchableOpacity>
        );
      })}
    </View>
  );
}

export function CRMContactForm({ mode, contactId }: { mode: ContactFormMode; contactId?: string }) {
  const router = useRouter();
  const colors = useCRMColors();
  const contactQuery = useContact(contactId ?? '');
  const accountsQuery = useAccounts();
  const createContact = useCreateContact();
  const updateContact = useUpdateContact();
  const [values, setValues] = useState<ContactFormValues>(emptyValues);
  const [error, setError] = useState<string | null>(null);
  const contact = useMemo(() => {
    const source = record(contactQuery.data)?.contact ?? contactQuery.data;
    return normalizeCRMContact(source);
  }, [contactQuery.data]);
  const accounts = useMemo(() => listAccounts(accountsQuery.data), [accountsQuery.data]);
  const loading = (mode === 'edit' && contactQuery.isLoading) || accountsQuery.isLoading;
  const submitting = createContact.isPending || updateContact.isPending;

  useEffect(() => {
    if (mode === 'edit' && contact.id) setValues(contactValues(contact));
  }, [contact, mode]);

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
    if (mode === 'edit' && contactId) {
      await updateContact.mutateAsync({ id: contactId, data });
      router.replace(`/crm/contacts/${contactId}`);
      return;
    }
    await createContact.mutateAsync(data);
    router.replace('/crm/contacts');
  };

  if (loading) {
    return <LoadingView testID="crm-contact-form-loading" colors={colors} />;
  }

  return (
    <ScrollView style={[baseFormStyles.container, { backgroundColor: colors.background }]} testID="crm-contact-form-screen">
      <View style={[baseFormStyles.card, { backgroundColor: colors.surface }]}>
        <Text style={[baseFormStyles.label, { color: colors.onSurfaceVariant }]}>Account</Text>
        <AccountPicker accounts={accounts} selectedId={values.accountId} onSelect={(id) => setField('accountId', id)} />
        <Field label="First name" value={values.firstName} onChangeText={(value) => setField('firstName', value)} testID="crm-contact-form-first-name" />
        <Field label="Last name" value={values.lastName} onChangeText={(value) => setField('lastName', value)} testID="crm-contact-form-last-name" />
        <Field label="Email" value={values.email} onChangeText={(value) => setField('email', value)} testID="crm-contact-form-email" />
        <Field label="Phone" value={values.phone} onChangeText={(value) => setField('phone', value)} testID="crm-contact-form-phone" />
        <Field label="Title" value={values.title} onChangeText={(value) => setField('title', value)} testID="crm-contact-form-title" />
        <FormErrorText error={error} style={[baseFormStyles.error, { color: colors.error }]} />
        <SubmitButton
          testID="crm-contact-form-submit"
          onPress={onSubmit}
          disabled={submitting}
          label={mode === 'edit' ? 'Save Contact' : 'Create Contact'}
          colors={colors}
        />
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  accountList: { gap: 8, marginBottom: 14 },
  accountOption: { borderWidth: 1, borderRadius: 8, padding: 12 },
  accountName: { fontSize: 15, fontWeight: '600' },
  help: { fontSize: 13, marginBottom: 14 },
});
