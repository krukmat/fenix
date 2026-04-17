// Task Mobile P1.4 — T1: Contact detail screen
import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack } from 'expo-router';
import { CRMDetailHeader } from '../../../src/components/crm';
import { useContact } from '../../../src/hooks/useCRM';
import { AgentActivitySection } from '../../../src/components/agents/AgentActivitySection';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import type { ThemeColors } from '../../../src/theme/types';

interface ContactDetailData {
  id: string;
  name?: string;
  email?: string;
  phone?: string;
  title?: string;
  accountId?: string;
  accountName?: string;
}

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function parseContact(data: unknown): ContactDetailData | undefined {
  const d = (data ?? null) as Record<string, unknown> | null;
  if (!d) return undefined;
  const c = (d.contact as Record<string, unknown> | undefined) ?? d;
  if (!c?.id) return undefined;
  return {
    id: String(c.id),
    name: c.name as string | undefined,
    email: c.email as string | undefined,
    phone: c.phone as string | undefined,
    title: c.title as string | undefined,
    accountId: (c.accountId ?? c.account_id) as string | undefined,
    accountName: c.accountName as string | undefined,
  };
}

function getMetadata(c: ContactDetailData) {
  return [
    { label: 'Title', value: c.title || 'N/A' },
    { label: 'Email', value: c.email || 'N/A' },
    { label: 'Phone', value: c.phone || 'N/A' },
  ];
}

function contactHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Contact',
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

export default function ContactDetailScreen() {
  const colors = useColors();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useContact(id);
  const contact = parseContact(data);

  if (isLoading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="contact-detail-loading">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading contact...</Text>
      </View>
    );
  }

  if (error || !contact) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID="contact-detail-error">
        <Text style={{ color: colors.error, fontSize: 16 }}>
          {error?.message || 'Contact not found'}
        </Text>
      </View>
    );
  }

  return (
    <>
      <Stack.Screen options={contactHeaderOptions(colors)} />
      <View
        testID="contact-detail-screen"
        style={[styles.container, { backgroundColor: colors.background }]}
      >
        <ScrollView style={styles.container}>
          <CRMDetailHeader
            title={contact.name || 'Unknown Contact'}
            subtitle={contact.title}
            metadata={getMetadata(contact)}
            testIDPrefix="contact-detail"
          />
          <AgentActivitySection
            entityType="contact"
            entityId={contact.id}
            testIDPrefix="contact-agent-activity"
          />
          <EntitySignalsSection
            entityType="contact"
            entityId={contact.id}
            testIDPrefix="contact-signals"
          />
        </ScrollView>
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
});
