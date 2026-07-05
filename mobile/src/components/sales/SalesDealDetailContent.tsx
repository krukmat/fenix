import React from 'react';
import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { Button } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { CRMDetailHeader } from '../crm';
import { AgentActivitySection } from '../agents/AgentActivitySection';
import { EntitySignalsSection } from '../signals/EntitySignalsSection';
import { CopilotPanel } from '../copilot';
import { DealStagePath } from './DealStagePath';
import { wedgeHref, wedgeHrefObject } from '../../utils/navigation';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { getAgentStatusColor } from '../../theme/semantic';
import { typography } from '../../theme/typography';
import type { ThemeColors } from '../../theme/types';

// Deal-stage FSM order for DealStagePath. An unrecognized deal.stage renders
// as a neutral "unknown" step rather than guessing a position (see invariants).
export const DEAL_STAGE_ORDER = ['prospecting', 'qualification', 'proposal', 'negotiation', 'closed_won'];

export interface SalesDealDetailData {
  id: string;
  title: string;
  status: string;
  amount?: number;
  stage?: string;
  closeDate?: string;
  accountId?: string;
  accountName?: string;
  activeSignalCount?: number;
}

function getMetadata(d: SalesDealDetailData) {
  return [
    { label: 'Status', value: d.status },
    { label: 'Stage', value: d.stage || 'N/A' },
    { label: 'Close Date', value: d.closeDate || 'Not set' },
  ];
}

function DealAmountSection({ amount, colors }: { amount?: number; colors: ThemeColors }) {
  if (amount === undefined) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Value</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="sales-deal-amount">
        <Text style={[typography.monoLG, { color: colors.onSurface, fontSize: 24 }]}>${amount.toLocaleString()}</Text>
      </View>
    </View>
  );
}

function DealStageSection({ stage, colors }: { stage?: string; colors: ThemeColors }) {
  if (!stage) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Stage</Text>
      <View style={[styles.card, { backgroundColor: colors.surface }]} testID="sales-deal-stage">
        <DealStagePath stage={stage} stages={DEAL_STAGE_ORDER} testIDPrefix="sales-deal-stage-path" />
      </View>
    </View>
  );
}

function DealAccountSection({
  accountId,
  accountName,
  router,
  colors,
}: {
  accountId?: string;
  accountName?: string;
  router: ReturnType<typeof useRouter>;
  colors: ThemeColors;
}) {
  if (!accountId) return null;
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Account</Text>
      <TouchableOpacity
        style={[styles.card, { backgroundColor: colors.surface }]}
        onPress={() => router.push(wedgeHref(`/sales/${accountId}`))}
      >
        <Text style={{ color: colors.onSurface, fontWeight: '500' }}>{accountName || 'View Account'}</Text>
      </TouchableOpacity>
    </View>
  );
}

function DealPrimaryActions({
  dealRouteId,
  dealId,
  router,
}: {
  dealRouteId: string;
  dealId: string;
  router: ReturnType<typeof useRouter>;
}) {
  return (
    <View style={styles.section}>
      <Button
        mode="contained"
        testID="sales-deal-brief-button"
        style={styles.actionButton}
        onPress={() =>
          router.push(wedgeHrefObject(`/sales/${dealRouteId}/brief`, { entity_type: 'deal', entity_id: dealId }))
        }
      >
        Sales Brief
      </Button>
      <Button
        mode="outlined"
        testID="sales-deal-copilot-button"
        onPress={() =>
          router.push(wedgeHrefObject(`/sales/${dealRouteId}/copilot`, { entity_type: 'deal', entity_id: dealId }))
        }
      >
        Open Copilot
      </Button>
    </View>
  );
}

function DealRiskActionSection({
  dealId,
  router,
  isPending,
  onTrigger,
}: {
  dealId: string;
  router: ReturnType<typeof useRouter>;
  isPending: boolean;
  onTrigger: (dealId: string, onSuccess: (runId: string) => void) => void;
}) {
  const label = isPending ? 'Running...' : 'Analyze Deal Risk';
  return (
    <View style={styles.section}>
      <Button
        mode="outlined"
        testID="deal-risk-trigger-button"
        disabled={isPending}
        onPress={() => onTrigger(dealId, (runId) => router.push(wedgeHref(`/activity/${runId}`)))}
      >
        {label}
      </Button>
    </View>
  );
}

export function SalesDealDetailContent({
  dealData,
  dealRouteId,
  colors,
  router,
  riskPending,
  onTriggerRisk,
}: {
  dealData: SalesDealDetailData;
  dealRouteId: string;
  colors: ThemeColors;
  router: ReturnType<typeof useRouter>;
  riskPending: boolean;
  onTriggerRisk: (dealId: string, onSuccess: (runId: string) => void) => void;
}) {
  return (
    <ScrollView testID="sales-deal-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={[styles.statusBanner, { backgroundColor: getAgentStatusColor(dealData.status) }]}>
        <Text style={styles.statusText}>STATUS: {dealData.status.toUpperCase()}</Text>
      </View>
      <CRMDetailHeader
        title={dealData.title}
        subtitle={dealData.accountName}
        metadata={getMetadata(dealData)}
        testIDPrefix="sales-deal-detail"
      />
      <DealAmountSection amount={dealData.amount} colors={colors} />
      <DealStageSection stage={dealData.stage} colors={colors} />
      <DealAccountSection
        accountId={dealData.accountId}
        accountName={dealData.accountName}
        router={router}
        colors={colors}
      />
      <DealPrimaryActions dealRouteId={dealRouteId} dealId={dealData.id} router={router} />
      <DealRiskActionSection
        dealId={dealData.id}
        router={router}
        isPending={riskPending}
        onTrigger={onTriggerRisk}
      />
      <AgentActivitySection entityType="deal" entityId={dealData.id} testIDPrefix="sales-deal-detail" />
      <EntitySignalsSection entityType="deal" entityId={dealData.id} testIDPrefix="sales-deal-detail" />
      <View style={styles.section}>
        <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Copilot</Text>
        <View style={styles.copilotContainer} testID="sales-deal-detail-copilot-panel">
          <CopilotPanel initialContext={{ entityType: 'deal', entityId: dealData.id }} />
        </View>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  statusBanner: { padding: spacing.sm, alignItems: 'center' },
  statusText: { color: brandColors.onError, fontWeight: '600', fontSize: 14 },
  section: { padding: spacing.base },
  sectionTitle: { ...typography.headingMD, marginBottom: spacing.md },
  card: { padding: spacing.base, borderRadius: radius.md },
  actionButton: { marginBottom: spacing.md },
  copilotContainer: { height: 480, borderRadius: radius.md, overflow: 'hidden' },
});
