// Task 4.3 — Account Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme, Button } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader, EntityTimeline } from '../../../src/components/crm';
import { AgentActivitySection } from '../../../src/components/agents/AgentActivitySection';
import { SignalCountBadge } from '../../../src/components/signals/SignalCountBadge';
import { useAccount } from '../../../src/hooks/useCRM';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface ContactItem { id: string; name: string; email?: string; phone?: string; title?: string; }
interface DealItem { id: string; name: string; value?: number; status: string; }
interface TimelineItem { id: string; type: 'note' | 'activity' | 'status_change' | 'created' | 'updated'; title: string; description?: string; timestamp: string; userName?: string; }
interface AccountData { id: string; name?: string; industry?: string; phone?: string; email?: string; website?: string; description?: string; contacts?: ContactItem[]; deals?: DealItem[]; timeline?: TimelineItem[]; activeSignalCount?: number; }

function getMetadata(account: AccountData) {
  return [
    { label: 'Industry', value: account.industry || 'Not specified' },
    { label: 'Phone', value: account.phone || 'Not available' },
    { label: 'Email', value: account.email || 'Not available' },
    { label: 'Website', value: account.website || 'Not available' },
  ];
}

function contactSubline(contact: ContactItem): string {
  if (contact.title && contact.email) return `${contact.title} · ${contact.email}`;
  if (contact.email) return contact.email;
  if (contact.phone) return contact.phone;
  if (contact.title) return contact.title;
  return 'Tap to view details';
}

function renderContactsSection(contacts: ContactItem[], colors: ThemeColors, onOpenContact: (id: string) => void) {
  return (
    <View style={styles.section}>
      <View style={styles.sectionHeader}>
        <Text style={[styles.title, { color: colors.onSurface }]}>Related Contacts</Text>
        <Text style={[styles.sectionCount, { color: colors.onSurfaceVariant }]}>{contacts.length}</Text>
      </View>
      {contacts.length === 0 ? (
        <View style={[styles.emptyCard, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurfaceVariant }}>No related contacts yet</Text>
        </View>
      ) : (
        contacts.map((contact) => (
          <TouchableOpacity
            key={contact.id}
            style={[styles.contactCard, { backgroundColor: colors.surface }]}
            onPress={() => onOpenContact(contact.id)}
            testID={`account-related-contact-${contact.id}`}
          >
            <View style={styles.contactHeader}>
              <Text style={[styles.contactName, { color: colors.onSurface }]}>{contact.name}</Text>
              <Text style={[styles.contactCta, { color: colors.primary }]}>View</Text>
            </View>
            <Text style={[styles.contactSubline, { color: colors.onSurfaceVariant }]}>{contactSubline(contact)}</Text>
          </TouchableOpacity>
        ))
      )}
    </View>
  );
}

function renderDealsSection(deals: DealItem[], colors: ThemeColors) {
  return (
    <View style={styles.section}>
      <Text style={[styles.title, { color: colors.onSurface }]}>Related Deals</Text>
      {deals.map(d => {
        const statusColor = d.status === 'won' ? '#10B981' : d.status === 'lost' ? '#EF4444' : colors.primary;
        return (
          <View key={d.id} style={[styles.card, { backgroundColor: colors.surface }]}>
            <View style={styles.row}>
              <Text style={{ color: colors.onSurface, fontWeight: '500', flex: 1 }}>{d.name}</Text>
              <View style={[styles.badge, { backgroundColor: statusColor }]}>
                <Text style={styles.badgeText}>{d.status}</Text>
              </View>
            </View>
            {d.value !== undefined && <Text style={{ color: colors.onSurfaceVariant, fontSize: 12, marginTop: 4 }}>${d.value.toLocaleString()}</Text>}
          </View>
        );
      })}
    </View>
  );
}

// Task 4.8 — GAP 3: Added testIDs for E2E tests
function renderTimelineSection(timeline: TimelineItem[], colors: ThemeColors) {
  return (
    <View style={styles.section}>
      <Text
        testID="account-timeline-tab"
        style={[styles.title, { color: colors.onSurface }]}
      >
        Activity
      </Text>
      <View testID="account-timeline-list">
        <EntityTimeline
          events={timeline}
          testIDPrefix="account-timeline"
          emptyMessage="No activity yet"
        />
      </View>
    </View>
  );
}

function renderContent(account: AccountData, colors: ThemeColors, onOpenContact: (id: string) => void, onOpenCopilot: () => void) {
  const metadata = getMetadata(account);
  return (
    <>
      <CRMDetailHeader title={account.name || 'Unnamed Account'} subtitle={account.description} metadata={metadata} testIDPrefix="account-detail" />
      <View style={styles.section}>
        <Text style={[styles.title, { color: colors.onSurface }]}>Signals</Text>
        <SignalCountBadge count={account.activeSignalCount} testID="account-detail-signal-badge" />
      </View>
      {renderContactsSection(account.contacts || [], colors, onOpenContact)}
      {account.deals && account.deals.length > 0 && renderDealsSection(account.deals, colors)}
      {renderTimelineSection(account.timeline || [], colors)}
      <AgentActivitySection entityType="account" entityId={account.id} testIDPrefix="account-agent-activity" />
      <EntitySignalsSection entityType="account" entityId={account.id} testIDPrefix="account-signals" />
      <View style={styles.section}>
        <Button mode="contained" onPress={onOpenCopilot} testID="account-copilot-open-button">
          Open Copilot
        </Button>
      </View>
    </>
  );
}

// eslint-disable-next-line complexity
export default function AccountDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  // FIX-4: Runtime guard for id param
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useAccount(id);
  const payload = (data ?? null) as Record<string, unknown> | null;
  const accountObj = (payload?.account as Record<string, unknown> | undefined) ?? payload ?? undefined;
  const contactsData = payload?.contacts as { data?: ContactItem[] } | ContactItem[] | undefined;
  const dealsData = payload?.deals as { data?: DealItem[] } | DealItem[] | undefined;
  const timelineData = payload?.timeline as { data?: TimelineItem[] } | TimelineItem[] | undefined;
  const account: AccountData | undefined = accountObj
    ? {
        id: String(accountObj.id ?? ''),
        name: accountObj.name as string | undefined,
        industry: accountObj.industry as string | undefined,
        phone: accountObj.phone as string | undefined,
        email: accountObj.email as string | undefined,
        website: accountObj.website as string | undefined,
        description: accountObj.description as string | undefined,
        contacts: (Array.isArray(contactsData) ? contactsData : contactsData?.data)?.map((contact) => {
          const raw = contact as unknown as Record<string, unknown>;
          const fullName = [raw.firstName, raw.lastName].filter(Boolean).join(' ').trim();
          return {
            id: String(raw.id ?? ''),
            name: (raw.name as string | undefined) ?? (fullName || 'Unnamed Contact'),
            email: raw.email as string | undefined,
            phone: raw.phone as string | undefined,
            title: raw.title as string | undefined,
          };
        }),
        deals: Array.isArray(dealsData) ? dealsData : dealsData?.data,
        timeline: Array.isArray(timelineData) ? timelineData : timelineData?.data,
        activeSignalCount: typeof payload?.active_signal_count === 'number' ? payload.active_signal_count : 0,
      }
    : undefined;

  // FIX-1: Removed useMemo wrapping JSX
  const content = account
    ? renderContent(
        account,
        colors,
        (contactId) => router.push(`/contacts/${contactId}`),
        () => router.push({ pathname: '/copilot', params: { entity_type: 'account', entity_id: account.id } }),
      )
    : null;
  const title = account?.name || 'Account';

  return (
    <>
      <Stack.Screen options={{ title }} />
      <ScrollView testID="account-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
        {isLoading ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading account...</Text>
          </View>
        ) : error || !account ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Account not found'}</Text>
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
  sectionHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  sectionCount: {
    fontSize: 12,
    fontWeight: '600',
  },
  title: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 12, borderRadius: 8, marginBottom: 8 },
  contactCard: {
    padding: 14,
    borderRadius: 10,
    marginBottom: 10,
    elevation: 1,
  },
  emptyCard: {
    padding: 14,
    borderRadius: 10,
  },
  contactHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 4,
  },
  contactName: {
    fontSize: 15,
    fontWeight: '600',
    flex: 1,
  },
  contactCta: {
    fontSize: 12,
    fontWeight: '700',
  },
  contactSubline: {
    fontSize: 12,
  },
  row: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  badge: { paddingHorizontal: 8, paddingVertical: 4, borderRadius: 12 },
  badgeText: { color: '#FFF', fontSize: 12, fontWeight: '500' },
});
