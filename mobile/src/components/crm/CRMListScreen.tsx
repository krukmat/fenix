// Task 4.3 — Reusable CRM List Screen Component

import React, { useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  RefreshControl,
  StyleSheet,
  ActivityIndicator,
  TextInput,
  TouchableOpacity,
} from 'react-native';
import { useTheme } from 'react-native-paper';
import type { ThemeColors } from '../../theme/types';

export interface CRMListItem {
  id: string;
}

export interface CRMListScreenProps<T extends CRMListItem> {
  data: T[];
  loading: boolean;
  error?: string | null;
  onRefresh: () => void;
  onEndReached?: () => void;
  searchValue: string;
  onSearchChange: (value: string) => void;
  renderItem: ({ item }: { item: T }) => React.ReactElement;
  emptyTitle: string;
  emptySubtitle?: string;
  testIDPrefix: string;
  onRetry?: () => void;
  isRefreshing?: boolean;
  loadingMore?: boolean;
  hasMore?: boolean;
  statusFilter?: string;
  onStatusFilterChange?: (status: string) => void;
  availableStatuses?: string[];
  hasData?: boolean;
}

// Helper to get theme colors
function useThemeColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

// Loading state component
function LoadingState({ testIDPrefix, colors }: { testIDPrefix: string; colors: ThemeColors }) {
  return (
    <View style={[styles.container, styles.centered]} testID={`${testIDPrefix}-loading`}>
      <ActivityIndicator size="large" color={colors.primary} />
      <Text style={[styles.loadingText, { color: colors.onSurfaceVariant }]}>Loading...</Text>
    </View>
  );
}

// Error state component
function ErrorState({
  testIDPrefix,
  colors,
  error,
  onRetry,
}: {
  testIDPrefix: string;
  colors: ThemeColors;
  error?: string | null;
  onRetry?: () => void;
}) {
  return (
    <View style={[styles.container, styles.centered]} testID={`${testIDPrefix}-error`}>
      <Text style={[styles.errorText, { color: colors.error }]}>{error}</Text>
      {onRetry && (
        <TouchableOpacity
          style={[styles.retryButton, { backgroundColor: colors.primary }]}
          onPress={onRetry}
          testID={`${testIDPrefix}-retry`}
        >
          <Text style={[styles.retryButtonText, { color: colors.onPrimary }]}>Retry</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

// Empty state component
function EmptyState({
  testIDPrefix,
  colors,
  emptyTitle,
  emptySubtitle,
}: {
  testIDPrefix: string;
  colors: ThemeColors;
  emptyTitle: string;
  emptySubtitle?: string;
}) {
  return (
    <View style={[styles.container, styles.centered]} testID={`${testIDPrefix}-empty`}>
      <Text style={[styles.emptyTitle, { color: colors.onSurface }]}>{emptyTitle}</Text>
      {emptySubtitle && <Text style={[styles.emptySubtitle, { color: colors.onSurfaceVariant }]}>{emptySubtitle}</Text>}
    </View>
  );
}

// Search and filter header component
function ListHeader({
  colors,
  testIDPrefix,
  searchValue,
  handleSearchChange,
  statusFilter,
  onStatusFilterChange,
  availableStatuses,
}: {
  colors: ThemeColors;
  testIDPrefix: string;
  searchValue: string;
  handleSearchChange: (value: string) => void;
  statusFilter?: string;
  onStatusFilterChange?: (status: string) => void;
  availableStatuses?: string[];
}) {
  return (
    <View style={styles.header}>
      <View style={[styles.searchContainer, { backgroundColor: colors.surfaceVariant, borderColor: colors.outline }]}>
        <TextInput
          style={[styles.searchInput, { color: colors.onSurface }]}
          placeholder="Search..."
          placeholderTextColor={colors.onSurfaceVariant}
          value={searchValue}
          onChangeText={handleSearchChange}
          testID={`${testIDPrefix}-search`}
        />
      </View>
      {availableStatuses && availableStatuses.length > 0 && (
        <View style={styles.statusFilterContainer}>
          <TouchableOpacity
            style={[styles.statusChip, { backgroundColor: !statusFilter ? colors.primary : colors.surfaceVariant }]}
            onPress={() => onStatusFilterChange?.('')}
            testID={`${testIDPrefix}-status-all`}
          >
            <Text style={[styles.statusChipText, { color: !statusFilter ? colors.onPrimary : colors.onSurfaceVariant }]}>All</Text>
          </TouchableOpacity>
          {availableStatuses.map((status) => (
            <TouchableOpacity
              key={status}
              style={[styles.statusChip, { backgroundColor: statusFilter === status ? colors.primary : colors.surfaceVariant }]}
              onPress={() => onStatusFilterChange?.(status)}
              testID={`${testIDPrefix}-status-${status}`}
            >
              <Text style={[styles.statusChipText, { color: statusFilter === status ? colors.onPrimary : colors.onSurfaceVariant }]}>{status}</Text>
            </TouchableOpacity>
          ))}
        </View>
      )}
    </View>
  );
}

// Footer loading component
function FooterLoading({ colors, loadingMore }: { colors: ThemeColors; loadingMore: boolean }) {
  if (!loadingMore) return null;
  return (
    <View style={styles.footerLoading}>
      <ActivityIndicator size="small" color={colors.primary} />
    </View>
  );
}

// Check initial states
function checkInitialState<T extends CRMListItem>(
  loading: boolean,
  error: string | null | undefined,
  data: T[]
): 'loading' | 'error' | 'empty' | 'list' {
  if (loading && data.length === 0) return 'loading';
  if (error && data.length === 0) return 'error';
  if (!loading && !error && data.length === 0) return 'empty';
  return 'list';
}

export function CRMListScreen<T extends CRMListItem>({
  data,
  loading,
  error,
  onRefresh,
  onEndReached,
  searchValue,
  onSearchChange,
  renderItem,
  emptyTitle,
  emptySubtitle,
  testIDPrefix,
  onRetry,
  isRefreshing = false,
  loadingMore = false,
  hasMore = false,
  statusFilter,
  onStatusFilterChange,
  availableStatuses,
  hasData,
}: CRMListScreenProps<T>) {
  const colors = useThemeColors();

  const handleSearchChange = useCallback(
    (value: string) => {
      onSearchChange(value);
    },
    [onSearchChange]
  );

  const stateType = checkInitialState(loading, error, data);

  // Render based on state
  if (stateType === 'loading') {
    return <LoadingState testIDPrefix={testIDPrefix} colors={colors} />;
  }

  if (stateType === 'error') {
    return <ErrorState testIDPrefix={testIDPrefix} colors={colors} error={error} onRetry={onRetry} />;
  }

  // FIX-3: Only show full empty state if hasData is not true (meaning no data at all)
  if (stateType === 'empty' && (hasData === undefined || !hasData)) {
    return <EmptyState testIDPrefix={testIDPrefix} colors={colors} emptyTitle={emptyTitle} emptySubtitle={emptySubtitle} />;
  }

  // List state or filtered-to-empty state (hasData is true)
  return (
    <View style={[styles.container, { backgroundColor: colors.background }]} testID={`${testIDPrefix}-list`}>
      <FlatList
        data={data}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        ListHeaderComponent={() => (
          <ListHeader
            colors={colors}
            testIDPrefix={testIDPrefix}
            searchValue={searchValue}
            handleSearchChange={handleSearchChange}
            statusFilter={statusFilter}
            onStatusFilterChange={onStatusFilterChange}
            availableStatuses={availableStatuses}
          />
        )}
        ListFooterComponent={() => <FooterLoading colors={colors} loadingMore={loadingMore} />}
        refreshControl={<RefreshControl refreshing={isRefreshing} onRefresh={onRefresh} tintColor={colors.primary} />}
        onEndReached={hasMore ? onEndReached : undefined}
        onEndReachedThreshold={0.5}
        contentContainerStyle={styles.listContent}
        showsVerticalScrollIndicator={false}
        testID={`${testIDPrefix}-flatlist`}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center' },
  listContent: { paddingBottom: 16 },
  header: { padding: 16 },
  searchContainer: { borderRadius: 8, borderWidth: 1, paddingHorizontal: 12, paddingVertical: 4 },
  searchInput: { fontSize: 16, paddingVertical: 8 },
  statusFilterContainer: { flexDirection: 'row', flexWrap: 'wrap', marginTop: 12, gap: 8 },
  statusChip: { paddingHorizontal: 12, paddingVertical: 6, borderRadius: 16 },
  statusChipText: { fontSize: 14, fontWeight: '500' },
  loadingText: { marginTop: 12, fontSize: 16 },
  errorText: { fontSize: 16, textAlign: 'center', marginBottom: 16 },
  retryButton: { paddingHorizontal: 24, paddingVertical: 12, borderRadius: 8 },
  retryButtonText: { fontSize: 16, fontWeight: '500' },
  emptyTitle: { fontSize: 18, fontWeight: '500', textAlign: 'center', marginBottom: 8 },
  emptySubtitle: { fontSize: 14, textAlign: 'center' },
  footerLoading: { padding: 16, alignItems: 'center' },
});

export default CRMListScreen;
