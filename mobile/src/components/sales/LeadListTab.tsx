import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, ActivityIndicator } from 'react-native';
import { useRouter } from 'expo-router';
import { useLeads } from '../../hooks/useCRM';
import { brandColors } from '../../theme/colors';
import { elevation, radius, spacing } from '../../theme/spacing';
import { getAgentStatusColor } from '../../theme/semantic';
import { typography } from '../../theme/typography';
import { wedgeHref } from '../../utils/navigation';
import type { ThemeColors } from '../../theme/types';

interface LeadItem {
  id: string;
  source?: string;
  status: string;
  score?: number;
  metadata?: string;
}

function getLeadStatusColor(status: string): string {
  return status === 'qualified' ? getAgentStatusColor('success') : getAgentStatusColor('open');
}

function parseLeadHeadline(item: LeadItem): { title: string; subtitle?: string } {
  if (!item.metadata) return { title: `Lead ${item.id}`, subtitle: item.source };
  try {
    const parsed = JSON.parse(item.metadata) as Record<string, unknown>;
    return {
      title: typeof parsed.name === 'string' ? parsed.name : `Lead ${item.id}`,
      subtitle: typeof parsed.email === 'string' ? parsed.email : item.source,
    };
  } catch {
    return { title: `Lead ${item.id}`, subtitle: item.source };
  }
}

function LeadRow({
  item,
  index,
  colors,
  onPress,
}: {
  item: LeadItem;
  index: number;
  colors: ThemeColors;
  onPress: () => void;
}) {
  const headline = parseLeadHeadline(item);

  return (
    <TouchableOpacity
      testID={`sales-lead-item-${index}`}
      style={[styles.row, { backgroundColor: colors.surface }]}
      onPress={onPress}
    >
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.onSurface }]}>{headline.title}</Text>
        <View style={[styles.statusChip, { backgroundColor: getLeadStatusColor(item.status) }]}>
          <Text style={styles.statusText}>{item.status}</Text>
        </View>
      </View>
      {headline.subtitle ? <Text style={[styles.sub, { color: colors.onSurfaceVariant }]}>{headline.subtitle}</Text> : null}
      {item.score !== undefined ? <Text style={[styles.sub, { color: colors.onSurfaceVariant }]}>Score: {item.score}</Text> : null}
    </TouchableOpacity>
  );
}

export function LeadListTab({ colors, router }: { colors: ThemeColors; router: ReturnType<typeof useRouter> }) {
  const { data, isLoading, fetchNextPage, hasNextPage } = useLeads();

  if (isLoading) {
    return (
      <View style={styles.center} testID="sales-leads-loading">
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  const items: LeadItem[] = (data?.pages ?? []).flatMap(
    (page: { data?: unknown[] }) => (page.data ?? []) as LeadItem[],
  );

  if (items.length === 0) {
    return (
      <View style={styles.center} testID="sales-leads-empty">
        <Text style={{ color: colors.onSurfaceVariant }}>No leads yet</Text>
      </View>
    );
  }

  return (
    <FlatList
      data={items}
      keyExtractor={(item) => item.id}
      renderItem={({ item, index }) => (
        <LeadRow
          item={item}
          index={index}
          colors={colors}
          onPress={() => router.push(wedgeHref(`/sales/leads/${item.id}`))}
        />
      )}
      contentContainerStyle={styles.listContent}
      onEndReached={() => {
        if (hasNextPage) fetchNextPage();
      }}
      onEndReachedThreshold={0.3}
    />
  );
}

const styles = StyleSheet.create({
  listContent: { padding: spacing.base },
  row: { padding: spacing.base, borderRadius: radius.md, marginBottom: spacing.md, ...elevation.card },
  header: { flexDirection: 'row', alignItems: 'center', marginBottom: 2 },
  title: { fontSize: 16, fontWeight: '600', marginBottom: 2, flex: 1 },
  sub: { fontSize: 14, marginTop: 2 },
  statusChip: { paddingHorizontal: spacing.sm, paddingVertical: spacing.xs, borderRadius: radius.full },
  statusText: { color: brandColors.onError, ...typography.labelMD },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: spacing.xl },
});
