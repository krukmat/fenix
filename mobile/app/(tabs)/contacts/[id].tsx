// Task 4.3 — Contact Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader } from '../../../src/components/crm';
import { useContact } from '../../../src/hooks/useCRM';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface ContactDetailData {
  id: string;
  name?: string;
  email?: string;
  phone?: string;
  accountId?: string;
  accountName?: string;
  title?: string;
  department?: string;
}

function getMetadata(contact: ContactDetailData) {
  return [
    { label: 'Email', value: contact.email || 'Not available' },
    { label: 'Phone', value: contact.phone || 'Not available' },
    { label: 'Title', value: contact.title || 'Not specified' },
    { label: 'Department', value: contact.department || 'Not specified' },
  ];
}

function renderAccountSection(accountId: string | undefined, accountName: string | undefined, router: ReturnType<typeof useRouter>, colors: ThemeColors) {
  if (!accountId) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.title, { color: colors.onSurface }]}>Account</Text>
      <TouchableOpacity
        style={[styles.card, { backgroundColor: colors.surface }]}
        onPress={() => router.push(`/accounts/${accountId}`)}
      >
        <Text style={{ color: colors.onSurface, fontWeight: '500' }}>{accountName || 'View Account'}</Text>
      </TouchableOpacity>
    </View>
  );
}

function renderContent(contact: ContactDetailData, router: ReturnType<typeof useRouter>, colors: ThemeColors) {
  const metadata = getMetadata(contact);
  return (
    <>
      <CRMDetailHeader
        title={contact.name || 'Unnamed Contact'}
        subtitle={contact.accountName ? `Works at ${contact.accountName}` : undefined}
        metadata={metadata}
        testIDPrefix="contact-detail"
      />
      {renderAccountSection(contact.accountId, contact.accountName, router, colors)}
      <EntitySignalsSection entityType="contact" entityId={contact.id} testIDPrefix="contact-signals" />
    </>
  );
}

function s(o: Record<string, unknown> | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

function parseContactPayload(data: unknown): ContactDetailData | undefined {
  const r = (data ?? null) as Record<string, unknown> | null;
  if (!r) return undefined;
  const fullName = [r.firstName, r.lastName].filter(Boolean).join(' ') || undefined;
  return {
    id: String(r.id ?? ''),
    name: s(r, 'name') ?? fullName,
    email: s(r, 'email'),
    phone: s(r, 'phone'),
    accountId: s(r, 'accountId') ?? s(r, 'account_id'),
    accountName: s(r, 'accountName'),
    title: s(r, 'title'),
    department: s(r, 'department'),
  };
}

export default function ContactDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  // FIX-4: Runtime guard for id param
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useContact(id);
  const contact = parseContactPayload(data);

  // FIX-1: Removed useMemo wrapping JSX
  const content = contact ? renderContent(contact, router, colors) : null;

  return (
    <>
      <Stack.Screen options={{ title: contact?.name || 'Contact' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]}>
        {isLoading ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading contact...</Text>
          </View>
        ) : error || !contact ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Contact not found'}</Text>
          </View>
        ) : content}
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  section: { padding: 16 },
  title: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 16, borderRadius: 8 },
});
