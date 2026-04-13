import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator } from 'react-native';
import { useTheme, Button } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { CRMDetailHeader } from '../../../../src/components/crm';
import { AgentActivitySection } from '../../../../src/components/agents/AgentActivitySection';
import { useLead } from '../../../../src/hooks/useCRM';
import { useTriggerProspectingAgent } from '../../../../src/hooks/useWedge';
import { wedgeHref } from '../../../../src/utils/navigation';
import type { ThemeColors } from '../../../../src/theme/types';

interface LeadDetailData {
  id: string;
  name?: string;
  email?: string;
  company?: string;
  source?: string;
  status: string;
  ownerId?: string;
  score?: number;
  metadata?: string;
}

type MetadataRecord = Record<string, unknown>;
type LeadRecord = Record<string, unknown>;

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function s(o: LeadRecord | null | undefined, key: string): string | undefined {
  return o?.[key] as string | undefined;
}

function n(o: LeadRecord | null | undefined, key: string): number | undefined {
  const value = o?.[key];
  return typeof value === 'number' ? value : undefined;
}

function parseMetadata(raw: unknown): MetadataRecord | undefined {
  if (typeof raw !== 'string' || raw.trim() === '') return undefined;
  try {
    const parsed = JSON.parse(raw) as unknown;
    return parsed && typeof parsed === 'object' ? (parsed as MetadataRecord) : undefined;
  } catch {
    return undefined;
  }
}

function getMetadataField(metadata: MetadataRecord | undefined, key: string): string | undefined {
  return typeof metadata?.[key] === 'string' ? (metadata[key] as string) : undefined;
}

function getOwnerId(lead: LeadRecord): string | undefined {
  return s(lead, 'ownerId') ?? s(lead, 'owner_id');
}

function parseLeadPayload(data: unknown): LeadDetailData | undefined {
  const lead = (data ?? null) as LeadRecord | null;
  if (!lead?.id) return undefined;
  const metadata = parseMetadata(lead.metadata);

  return {
    id: String(lead.id),
    name: getMetadataField(metadata, 'name'),
    email: getMetadataField(metadata, 'email'),
    company: getMetadataField(metadata, 'company'),
    source: s(lead, 'source'),
    status: s(lead, 'status') ?? 'new',
    ownerId: getOwnerId(lead),
    score: n(lead, 'score'),
    metadata: s(lead, 'metadata'),
  };
}

function getMetadata(lead: LeadDetailData) {
  return [
    { label: 'Source', value: lead.source || 'Unknown' },
    { label: 'Status', value: lead.status },
    { label: 'Owner', value: lead.ownerId || 'Unassigned' },
    { label: 'Score', value: lead.score !== undefined ? String(lead.score) : 'N/A' },
  ];
}

function LeadMetaSection({ lead, colors }: { lead: LeadDetailData; colors: ThemeColors }) {
  const hasTopline = lead.email || lead.company;
  const hasMetadata = typeof lead.metadata === 'string' && lead.metadata.trim() !== '';
  if (!hasTopline && !hasMetadata) return null;

  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Lead Context</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]}>
        {lead.email ? <Text style={{ color: colors.onSurface }}>Email: {lead.email}</Text> : null}
        {lead.company ? <Text style={{ color: colors.onSurface, marginTop: lead.email ? 6 : 0 }}>Company: {lead.company}</Text> : null}
        {hasMetadata ? (
          <Text style={{ color: colors.onSurfaceVariant, marginTop: hasTopline ? 10 : 0, fontSize: 12 }}>
            Metadata: {lead.metadata}
          </Text>
        ) : null}
      </View>
    </View>
  );
}

function LeadLoading({ colors }: { colors: ThemeColors }) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-lead-detail-loading">
      <ActivityIndicator size="large" color={colors.primary} />
      <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>Loading lead...</Text>
    </View>
  );
}

function LeadError({ colors, message }: { colors: ThemeColors; message: string }) {
  return (
    <View style={[styles.centered, { backgroundColor: colors.background }]} testID="sales-lead-detail-error">
      <Text style={{ color: colors.error, fontSize: 16 }}>{message}</Text>
    </View>
  );
}

function leadDetailHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Lead',
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

function LeadDetailContent({
  leadData,
  colors,
  isPending,
  onTrigger,
}: {
  leadData: LeadDetailData;
  colors: ThemeColors;
  isPending: boolean;
  onTrigger: () => void;
}) {
  return (
    <ScrollView testID="sales-lead-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <CRMDetailHeader
        title={leadData.name || `Lead ${leadData.id}`}
        subtitle={leadData.email || leadData.company || leadData.source}
        metadata={getMetadata(leadData)}
        testIDPrefix="sales-lead-detail"
      />
      <View style={styles.section}>
        <Button
          mode="contained"
          testID="prospecting-trigger-button"
          disabled={isPending}
          onPress={onTrigger}
        >
          {isPending ? 'Running...' : 'Run Prospecting Agent'}
        </Button>
      </View>
      <LeadMetaSection lead={leadData} colors={colors} />
      <AgentActivitySection entityType="lead" entityId={leadData.id} testIDPrefix="sales-lead-detail" />
    </ScrollView>
  );
}

export default function SalesLeadDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useLead(id);
  const triggerProspecting = useTriggerProspectingAgent();
  const leadData = parseLeadPayload(data);

  const handleTriggerProspecting = async (leadId: string) => {
    const result = await triggerProspecting.mutateAsync({ leadId });
    router.push(wedgeHref(`/activity/${result.runId}`));
  };

  if (isLoading) return <LeadLoading colors={colors} />;
  if (error || !leadData) return <LeadError colors={colors} message={error?.message || 'Lead not found'} />;

  return (
    <>
      <Stack.Screen options={leadDetailHeaderOptions(colors)} />
      <LeadDetailContent
        leadData={leadData}
        colors={colors}
        isPending={triggerProspecting.isPending}
        onTrigger={() => {
          void handleTriggerProspecting(leadData.id);
        }}
      />
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  section: { padding: 16 },
  sectionTitle: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 16, borderRadius: 8 },
});
