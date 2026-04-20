import React, { useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { useRouter } from 'expo-router';
import { useTheme } from 'react-native-paper';
import { normalizeCRMAccount } from '../../services/api';
import type { CRMAccount } from '../../services/api';
import type { ThemeColors } from '../../theme/types';
import { useAccount, useCreateAccount, useUpdateAccount } from '../../hooks/useCRM';

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

function useCRMColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

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

function record(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' ? (value as Record<string, unknown>) : null;
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

function Field({
  label,
  value,
  onChangeText,
  testID,
  multiline,
}: {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  testID: string;
  multiline?: boolean;
}) {
  const colors = useCRMColors();
  return (
    <View style={styles.field}>
      <Text style={[styles.label, { color: colors.onSurfaceVariant }]}>{label}</Text>
      <TextInput
        testID={testID}
        value={value}
        onChangeText={onChangeText}
        multiline={multiline}
        style={[
          styles.input,
          multiline ? styles.multiline : null,
          { borderColor: colors.outline, color: colors.onSurface, backgroundColor: colors.surface },
        ]}
      />
    </View>
  );
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
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="crm-account-form-loading">
        <Text style={{ color: colors.onSurfaceVariant }}>Loading...</Text>
      </View>
    );
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor: colors.background }]} testID="crm-account-form-screen">
      <View style={[styles.card, { backgroundColor: colors.surface }]}>
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
        {error ? <Text style={[styles.error, { color: colors.error }]}>{error}</Text> : null}
        <TouchableOpacity
          testID="crm-account-form-submit"
          style={[styles.submit, { backgroundColor: colors.primary }, submitting ? styles.disabled : null]}
          onPress={onSubmit}
          disabled={submitting}
        >
          <Text style={[styles.submitText, { color: colors.onPrimary }]}>{mode === 'edit' ? 'Save Account' : 'Create Account'}</Text>
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
  error: { fontSize: 14, marginBottom: 12 },
  submit: { minHeight: 48, borderRadius: 8, alignItems: 'center', justifyContent: 'center' },
  disabled: { opacity: 0.7 },
  submitText: { fontSize: 16, fontWeight: '700' },
});
