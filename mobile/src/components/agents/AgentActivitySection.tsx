import React, { useMemo } from 'react';
import { StyleSheet, TouchableOpacity, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useAgentRunsByEntity } from '../../hooks/useAgentSpec';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import { formatLatency, getStatusColor, getStatusLabel } from '../../screens/agents/agentDetail.helpers';
import { wedgeHref } from '../../utils/navigation';

interface AgentActivitySectionProps {
  entityType: 'account' | 'deal' | 'case' | 'lead' | 'contact';
  entityId: string;
  testIDPrefix?: string;
}

interface AgentRunSummary {
  id: string;
  agent_name?: string;
  status: string;
  started_at?: string;
  latency_ms?: number;
}

export function AgentActivitySection({ entityType, entityId, testIDPrefix = 'agent-activity' }: AgentActivitySectionProps) {
  const theme = useTheme();
  const router = useRouter();
  const { data, isLoading } = useAgentRunsByEntity(entityType, entityId);

  const runs = useMemo(() => {
    if (!data?.pages) return [];
    return data.pages.flatMap((page) => (page.data as AgentRunSummary[] | undefined) ?? []).slice(0, 3);
  }, [data]);

  if (isLoading || runs.length === 0) {
    return null;
  }

  return (
    <View style={styles.section} testID={`${testIDPrefix}-section`}>
      <Text style={[styles.title, { color: theme.colors.onSurface }]}>Agent Activity</Text>
      {runs.map((run) => (
        <TouchableOpacity
          key={run.id}
          style={[styles.card, { backgroundColor: theme.colors.surface }]}
          onPress={() => router.push(wedgeHref(`/activity/${run.id}`))}
          testID={`${testIDPrefix}-item-${run.id}`}
        >
          <View style={styles.header}>
            <Text style={[styles.name, { color: theme.colors.onSurface }]}>{run.agent_name || 'Agent Run'}</Text>
            <View style={[styles.badge, { backgroundColor: getStatusColor(run.status) }]}>
              <Text style={styles.badgeText}>{getStatusLabel(run.status)}</Text>
            </View>
          </View>
          <Text style={{ color: theme.colors.onSurfaceVariant, fontSize: 12 }}>
            {run.started_at ? new Date(run.started_at).toLocaleString() : 'Unknown start'}
          </Text>
          {typeof run.latency_ms === 'number' ? (
            <Text style={{ color: theme.colors.onSurfaceVariant, fontSize: 12, marginTop: 4 }}>
              {formatLatency(run.latency_ms)}
            </Text>
          ) : null}
        </TouchableOpacity>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { padding: spacing.base },
  title: { ...typography.headingMD, marginBottom: spacing.md },
  card: { padding: spacing.md, borderRadius: radius.md, marginBottom: radius.md },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: spacing.sm },
  name: { fontSize: 14, fontWeight: '600', flex: 1, marginRight: spacing.sm },
  badge: { paddingHorizontal: spacing.sm, paddingVertical: spacing.xs, borderRadius: radius.full },
  badgeText: { color: brandColors.onError, ...typography.labelMD },
});
