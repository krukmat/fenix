// W2-T2 (mobile_wedge_harmonization_plan): extracted from deals/index — renderDealItem moved
// here so it can be unit-tested independently of the screen route.
import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useRouter } from 'expo-router';
import { SignalCountBadge } from '../signals/SignalCountBadge';
import type { ThemeColors } from '../../theme/types';

export interface DealData {
  id: string;
  title?: string;
  name?: string;
  amount?: number;
  value?: number;
  status: 'open' | 'won' | 'lost';
  stage?: string;
  accountName?: string;
  closeDate?: string;
  active_signal_count?: number;
}

function getStatusColor(status: string, colors: ThemeColors): string {
  if (status === 'won') return '#10B981';
  if (status === 'lost') return '#EF4444';
  return colors.primary;
}

export function renderDealItem(
  { item }: { item: DealData },
  colors: ThemeColors,
  router: ReturnType<typeof useRouter>
) {
  return (
    <TouchableOpacity
      style={[styles.dealItem, { backgroundColor: colors.surface }]}
      onPress={() => router.push(`/deals/${item.id}`)}
      testID={`deal-item-${item.id}`}
    >
      <View style={styles.dealHeader}>
        <Text style={[styles.dealName, { color: colors.onSurface }]}>
          {item.title || item.name || 'Unnamed Deal'}
        </Text>
        <View
          style={[styles.statusChip, { backgroundColor: getStatusColor(item.status, colors) }]}
          testID={`deal-status-${item.status}`}
        >
          <Text style={[styles.statusChipText, { color: '#FFFFFF' }]}>{item.status}</Text>
        </View>
      </View>
      {item.accountName && (
        <Text style={[styles.dealAccount, { color: colors.onSurfaceVariant }]}>
          {item.accountName}
        </Text>
      )}
      {(item.amount ?? item.value) !== undefined && (
        <Text style={[styles.dealValue, { color: colors.onSurfaceVariant }]}>
          ${((item.amount ?? item.value) as number).toLocaleString()}
        </Text>
      )}
      <View style={styles.badgeRow}>
        <SignalCountBadge count={item.active_signal_count} testID={`deal-signals-badge-${item.id}`} />
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  dealItem: {
    padding: 16,
    marginBottom: 12,
    borderRadius: 8,
    elevation: 2,
  },
  dealHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 4,
  },
  dealName: {
    fontSize: 16,
    fontWeight: '600',
    flex: 1,
  },
  statusChip: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 12,
  },
  statusChipText: {
    fontSize: 12,
    fontWeight: '500',
  },
  dealAccount: {
    fontSize: 14,
  },
  dealValue: {
    fontSize: 14,
    fontWeight: '500',
    marginTop: 4,
  },
  badgeRow: {
    alignItems: 'flex-start',
    marginTop: 8,
  },
});
