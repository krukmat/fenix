import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'expo-router';
import {
  normalizeCRMAccount,
  normalizeCRMContact,
} from '../../services/api';
import type { CRMAccount, CRMContact } from '../../services/api';
import { useAccounts, useContact, useCreateContact, useUpdateContact } from '../../hooks/useCRM';
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

function accountOptions(accounts: CRMAccount[]): OptionButtonItem[] {
  return accounts.map((account) => ({ id: account.id, label: account.name }));
}

function ContactFields({
  values,
  accounts,
  setField,
}: {
  values: ContactFormValues;
  accounts: OptionButtonItem[];
  setField: (field: FieldName, value: string) => void;
}) {
  return (
    <>
      <FormSectionLabel>Account</FormSectionLabel>
      <OptionButtonList
        items={accounts}
        selectedId={values.accountId}
        testIDPrefix="crm-contact-form-account"
        onSelect={(id) => setField('accountId', id)}
        emptyLabel="No accounts loaded"
      />
      <Field label="First name" value={values.firstName} onChangeText={(value) => setField('firstName', value)} testID="crm-contact-form-first-name" />
      <Field label="Last name" value={values.lastName} onChangeText={(value) => setField('lastName', value)} testID="crm-contact-form-last-name" />
      <Field label="Email" value={values.email} onChangeText={(value) => setField('email', value)} testID="crm-contact-form-email" />
      <Field label="Phone" value={values.phone} onChangeText={(value) => setField('phone', value)} testID="crm-contact-form-phone" />
      <Field label="Title" value={values.title} onChangeText={(value) => setField('title', value)} testID="crm-contact-form-title" />
    </>
  );
}

export function CRMContactForm({ mode, contactId }: { mode: ContactFormMode; contactId?: string }) {
  const router = useRouter();
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
  const accountChoices = useMemo(() => accountOptions(accounts), [accounts]);
  const colors = useCRMColors();
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
    <FormScreen testID="crm-contact-form-screen" colors={colors}>
      <ContactFields values={values} accounts={accountChoices} setField={setField} />
      <FormErrorText error={error} style={[baseFormStyles.error, { color: colors.error }]} />
      <SubmitButton
        testID="crm-contact-form-submit"
        onPress={onSubmit}
        disabled={submitting}
        label={mode === 'edit' ? 'Save Contact' : 'Create Contact'}
        colors={colors}
      />
    </FormScreen>
  );
}
