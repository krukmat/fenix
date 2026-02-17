// Task 4.3 — Case Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader } from '../../../src/components/crm';
import { useCase } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface CaseDetailData {
  id: string;
  subject?: string;
  status: string;
  priority: 'low' | 'medium' | 'high';
  description?: string;
  accountId?: string;
  accountName?: string;
  slaDeadline?: string;
  handoffStatus?: string;
  assignee?: string;
}

function getPriorityColor(priority: string): string {
  if (priority === 'high') return '#EF4444';
  if (priority === 'medium') return '#F59E0B';
  return '#10B981';
}

function getMetadata(caseData: CaseDetailData) {
  return [
    { label: 'Status', value: caseData.status },
    { label: 'Priority', value: caseData.priority },
    { label: 'Assignee', value: caseData.assignee || 'Unassigned' },
    { label: 'SLA Deadline', value: caseData.slaDeadline || 'Not set' },
  ];
}

function renderHandoffSection(handoffStatus: string | undefined, colors: ThemeColors) {
  if (!handoffStatus) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.title, { color: colors.onSurface }]}>Handoff Status</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="case-handoff-status">
        <Text style={{ color: colors.onSurface }}>{handoffStatus}</Text>
      </View>
    </View>
  );
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

// FIX-9: Export for tests
export function renderCaseContent(caseData: CaseDetailData, colors: ThemeColors, router: ReturnType<typeof useRouter>) {
  const metadata = getMetadata(caseData);
  return (
    <>
      <View style={[styles.priorityBanner, { backgroundColor: getPriorityColor(caseData.priority) }]}>
        <Text style={styles.priorityText}>PRIORITY: {caseData.priority.toUpperCase()}</Text>
      </View>
      <CRMDetailHeader title={caseData.subject || 'No Subject'} subtitle={caseData.description} metadata={metadata} testIDPrefix="case-detail" />
      {caseData.slaDeadline && (
        <View style={styles.section}>
          <Text style={[styles.title, { color: colors.onSurface }]}>SLA Deadline</Text>
          <View style={[styles.card, { backgroundColor: colors.surface }]} testID="case-sla-deadline">
            <Text style={{ color: colors.onSurface }}>{caseData.slaDeadline}</Text>
          </View>
        </View>
      )}
      {renderHandoffSection(caseData.handoffStatus, colors)}
      {renderAccountSection(caseData.accountId, caseData.accountName, router, colors)}
    </>
  );
}

// eslint-disable-next-line complexity
export default function CaseDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  // FIX-4: Runtime guard for id param
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useCase(id);
  const caseData: CaseDetailData | undefined = data?.data;

  // FIX-1: Removed useMemo wrapping JSX
  const content = caseData ? renderCaseContent(caseData, colors, router) : null;

  return (
    <>
      <Stack.Screen options={{ title: caseData?.subject || 'Case' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]}>
        {isLoading ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading case...</Text>
          </View>
        ) : error || !caseData ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <Text style={{ color: colors.error, fontSize: 16 }}>{error?.message || 'Case not found'}</Text>
          </View>
        ) : content}
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  priorityBanner: { padding: 8, alignItems: 'center' },
  priorityText: { color: '#FFF', fontWeight: '600', fontSize: 14 },
  section: { padding: 16 },
  title: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 16, borderRadius: 8 },
});
