import React, { useEffect, useMemo, useState } from 'react';
import { ScrollView, View } from 'react-native';
import { useRouter } from 'expo-router';
import { normalizeCRMAccount } from '../../services/api';
import type { CRMAccount } from '../../services/api';
import { useAccount, useCreateAccount, useUpdateAccount } from '../../hooks/useCRM';
import {
  Field,
  FormErrorText,
  LoadingView,
  SubmitButton,
  baseFormStyles,
  record,
  useCRMColors,
} from './CRMFormBase';

type AccountFormValues = {
  name: string;
  industry: string;
  website: string;
  phone: string;
  email: string;
  description: string;
};

type FieldName = keyof AccountFormValues;
type AccountFormMode = 'create' | 'edit';

const emptyValues: AccountFormValues = {
  name: '',
  industry: '',
  website: '',
  phone: '',
  email: '',
  description: '',
};

function accountValues(account: CRMAccount): AccountFormValues {
  return {
    name: account.name,
    industry: account.industry ?? '',
    website: account.website ?? '',
    phone: account.phone ?? '',
    email: account.email ?? '',
    description: account.description ?? '',
  };
}

function payload(values: AccountFormValues): AccountFormValues & { name: string } {
  const optional = Object.fromEntries(
    Object.entries(values)
      .filter(([key]) => key !== 'name')
      .map(([key, value]) => [key, value.trim()])
      .filter(([, value]) => value !== ''),
  ) as Partial<AccountFormValues>;
  return { ...optional, name: values.name.trim() } as AccountFormValues & { name: string };
}

function validate(values: AccountFormValues): string | null {
  if (!values.name.trim()) return 'Account name is required';
  if (values.email.trim() && !values.email.includes('@')) return 'Enter a valid email';
  return null;
}

export function CRMAccountForm({ mode, accountId }: { mode: AccountFormMode; accountId?: string }) {
  const router = useRouter();
  const colors = useCRMColors();
  const accountQuery = useAccount(accountId ?? '');
  const createAccount = useCreateAccount();
  const updateAccount = useUpdateAccount();
  const [values, setValues] = useState<AccountFormValues>(emptyValues);
  const [error, setError] = useState<string | null>(null);
  const account = useMemo(() => {
    const source = record(accountQuery.data)?.account ?? accountQuery.data;
    return normalizeCRMAccount(source);
  }, [accountQuery.data]);
  const loading = mode === 'edit' && accountQuery.isLoading;
  const submitting = createAccount.isPending || updateAccount.isPending;

  useEffect(() => {
    if (mode === 'edit' && account.id) setValues(accountValues(account));
  }, [account, mode]);

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
    if (mode === 'edit' && accountId) {
      await updateAccount.mutateAsync({ id: accountId, data });
      router.replace(`/crm/accounts/${accountId}`);
      return;
    }
    await createAccount.mutateAsync(data);
    router.replace('/crm/accounts');
  };

  if (loading) {
    return <LoadingView testID="crm-account-form-loading" colors={colors} />;
  }

  return (
    <ScrollView style={[baseFormStyles.container, { backgroundColor: colors.background }]} testID="crm-account-form-screen">
      <View style={[baseFormStyles.card, { backgroundColor: colors.surface }]}>
        <Field label="Name" value={values.name} onChangeText={(value) => setField('name', value)} testID="crm-account-form-name" />
        <Field label="Industry" value={values.industry} onChangeText={(value) => setField('industry', value)} testID="crm-account-form-industry" />
        <Field label="Website" value={values.website} onChangeText={(value) => setField('website', value)} testID="crm-account-form-website" />
        <Field label="Phone" value={values.phone} onChangeText={(value) => setField('phone', value)} testID="crm-account-form-phone" />
        <Field label="Email" value={values.email} onChangeText={(value) => setField('email', value)} testID="crm-account-form-email" />
        <Field
          label="Description"
          value={values.description}
          onChangeText={(value) => setField('description', value)}
          testID="crm-account-form-description"
          multiline
        />
        <FormErrorText error={error} style={[baseFormStyles.error, { color: colors.error }]} />
        <SubmitButton
          testID="crm-account-form-submit"
          onPress={onSubmit}
          disabled={submitting}
          label={mode === 'edit' ? 'Save Account' : 'Create Account'}
          colors={colors}
        />
      </View>
    </ScrollView>
  );
}
