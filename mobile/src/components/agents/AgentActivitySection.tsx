import React, { useMemo } from 'react';
import { StyleSheet, TouchableOpacity, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useAgentRunsByEntity } from '../../hooks/useAgentSpec';
import { formatLatency, getStatusColor, getStatusLabel } from '../../screens/agents/agentDetail.helpers';

interface AgentActivitySectionProps {
  entityType: 'account' | 'deal' | 'case' | 'lead';
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
          onPress={() => router.push(`/agents/${run.id}`)}
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
  section: { padding: 16 },
  title: { fontSize: 18, fontWeight: '600', marginBottom: 12 },
  card: { padding: 12, borderRadius: 8, marginBottom: 10 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 },
  name: { fontSize: 14, fontWeight: '600', flex: 1, marginRight: 8 },
  badge: { paddingHorizontal: 8, paddingVertical: 4, borderRadius: 12 },
  badgeText: { color: '#FFF', fontSize: 11, fontWeight: '600' },
});
