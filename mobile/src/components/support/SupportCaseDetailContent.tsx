import React, { useState } from 'react';
import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useRouter } from 'expo-router';
import { CopilotPanel } from '../copilot';
import { CRMDetailHeader, EntityTimeline, type TimelineEvent } from '../crm';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { ThemeColors } from '../../theme/types';
import type { SupportCaseDetailData } from './supportCaseDetail.types';
import { SupportCaseSummaryTab } from './SupportCaseDetailSections';
import { getPriorityColor, supportCaseDetailStyles } from './SupportCaseDetailMeta';

type SupportTab = 'summary' | 'timeline';

function getMetadata(c: SupportCaseDetailData) {
  return [
    { label: 'Priority', value: c.priority },
    { label: 'Owner', value: c.assignee || 'Unassigned' },
    { label: 'SLA Deadline', value: c.slaDeadline || 'Not set' },
  ];
}

function TabButton({
  label,
  active,
  onPress,
  testID,
  colors,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
  testID: string;
  colors: ThemeColors;
}) {
  return (
    <TouchableOpacity
      onPress={onPress}
      testID={testID}
      style={[
        styles.tabButton,
        {
          backgroundColor: active ? colors.primary : colors.surface,
          borderColor: active ? colors.primary : colors.outline,
        },
      ]}
    >
      <Text style={[styles.tabButtonText, { color: active ? brandColors.onError : colors.onSurface }]}>{label}</Text>
    </TouchableOpacity>
  );
}

function TimelineSection({
  colors,
  timelineEvents,
  timelineLoading,
}: {
  colors: ThemeColors;
  timelineEvents: TimelineEvent[];
  timelineLoading: boolean;
}) {
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Timeline</Text>
      <View style={[styles.card, { backgroundColor: colors.background }]} testID="support-case-timeline-panel">
        {timelineLoading ? (
          <Text style={{ color: colors.onSurfaceVariant }} testID="support-case-timeline-loading">
            Loading timeline...
          </Text>
        ) : (
          <EntityTimeline
            events={timelineEvents}
            testIDPrefix="support-case-timeline"
            emptyMessage="No case timeline available yet"
          />
        )}
      </View>
    </View>
  );
}

export function SupportCaseDetailContent({
  caseData,
  colors,
  router,
  signalSummary,
  triggerAgent,
  triggerKBIsPending,
  onTriggerKB,
  timelineEvents,
  timelineLoading,
}: {
  caseData: SupportCaseDetailData;
  colors: ThemeColors;
  router: ReturnType<typeof useRouter>;
  signalSummary: string | null;
  triggerAgent: { mutate: (context: { caseId: string; customerQuery: string }) => void; isPending: boolean };
  triggerKBIsPending: boolean;
  onTriggerKB: () => void;
  timelineEvents: TimelineEvent[];
  timelineLoading: boolean;
}) {
  const [activeTab, setActiveTab] = useState<SupportTab>('summary');

  return (
    <View testID="support-case-detail-screen" style={[styles.container, { backgroundColor: colors.background }]}>
      <ScrollView style={styles.container}>
        <View style={[styles.priorityBanner, { backgroundColor: getPriorityColor(caseData.priority) }]}>
          <Text style={styles.priorityText}>PRIORITY: {caseData.priority.toUpperCase()}</Text>
        </View>

        <CRMDetailHeader
          title={caseData.subject || 'No Subject'}
          subtitle={caseData.description}
          metadata={getMetadata(caseData)}
          testIDPrefix="support-case-detail"
        />

        <View style={styles.tabRow}>
          <TabButton
            label="Summary"
            active={activeTab === 'summary'}
            onPress={() => setActiveTab('summary')}
            testID="support-case-tab-summary"
            colors={colors}
          />
          <TabButton
            label="Timeline"
            active={activeTab === 'timeline'}
            onPress={() => setActiveTab('timeline')}
            testID="support-case-tab-timeline"
            colors={colors}
          />
        </View>

        {activeTab === 'summary' ? (
          <SupportCaseSummaryTab
            caseData={caseData}
            colors={colors}
            router={router}
            signalSummary={signalSummary}
            triggerAgent={triggerAgent}
            triggerKBIsPending={triggerKBIsPending}
            onTriggerKB={onTriggerKB}
          />
        ) : (
          <TimelineSection colors={colors} timelineEvents={timelineEvents} timelineLoading={timelineLoading} />
        )}

        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>Copilot</Text>
          <View style={styles.copilotContainer} testID="support-case-detail-copilot-panel">
            <CopilotPanel
              initialContext={{ entityType: 'case', entityId: caseData.id }}
              onSupportTrigger={(customerQuery) => triggerAgent.mutate({ caseId: caseData.id, customerQuery })}
              scrollEnabled={false}
            />
          </View>
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  ...supportCaseDetailStyles,
  container: { flex: 1 },
  priorityBanner: { padding: spacing.sm, alignItems: 'center' },
  priorityText: { color: brandColors.onError, fontWeight: '600', fontSize: 14 },
  tabRow: {
    flexDirection: 'row',
    gap: spacing.sm,
    paddingHorizontal: spacing.base,
    paddingBottom: spacing.sm,
  },
  tabButton: {
    flex: 1,
    borderRadius: radius.full,
    borderWidth: 1,
    paddingVertical: spacing.sm,
    alignItems: 'center',
  },
  tabButtonText: {
    ...typography.labelMD,
    fontWeight: '700',
  },
  copilotContainer: { height: 480, borderRadius: radius.md, overflow: 'hidden' },
});
