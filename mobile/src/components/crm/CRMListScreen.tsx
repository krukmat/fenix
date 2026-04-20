// Task 4.3 — Reusable CRM List Screen Component

import React, { useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  RefreshControl,
  StyleSheet,
  ActivityIndicator,
  TouchableOpacity,
} from 'react-native';
import { useTheme } from 'react-native-paper';
import type { ThemeColors } from '../../theme/types';
import { ListHeader, SelectableItem } from './CRMListSelection';

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
  renderItem: ({ item, index }: { item: T; index: number }) => React.ReactElement;
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
  primaryActionLabel?: string;
  onPrimaryAction?: () => void;
  selectedIds?: ReadonlySet<string>;
  onToggleSelect?: (id: string) => void;
  onSelectAllVisible?: () => void;
  onClearSelection?: () => void;
  selectionDisabled?: boolean;
  onRowEdit?: (id: string) => void;
  onBulkDelete?: () => void;
  bulkDeletePending?: boolean;
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

function renderListItem<T extends CRMListItem>({
  item,
  index,
  testIDPrefix,
  colors,
  selectedIds,
  selectionEnabled,
  selectionDisabled,
  onToggleSelect,
  onRowEdit,
  renderItem,
}: {
  item: T;
  index: number;
  testIDPrefix: string;
  colors: ThemeColors;
  selectedIds?: ReadonlySet<string>;
  selectionEnabled: boolean;
  selectionDisabled?: boolean;
  onToggleSelect?: (id: string) => void;
  onRowEdit?: (id: string) => void;
  renderItem: ({ item, index }: { item: T; index: number }) => React.ReactElement;
}) {
  if (!selectionEnabled || !onToggleSelect) return renderItem({ item, index });
  return (
    <SelectableItem
      item={item}
      index={index}
      testIDPrefix={testIDPrefix}
      colors={colors}
      selected={selectedIds?.has(item.id) ?? false}
      disabled={selectionDisabled}
      onToggleSelect={onToggleSelect}
      onEdit={onRowEdit}
      renderItem={renderItem}
    />
  );
}

type ListContentProps<T extends CRMListItem> = CRMListScreenProps<T> & {
  colors: ThemeColors;
  selectionEnabled: boolean;
  selectedCount: number;
  handleSearchChange: (value: string) => void;
};

function ListHeaderComponent<T extends CRMListItem>(props: ListContentProps<T>) {
  const { colors, testIDPrefix, data, searchValue, handleSearchChange, statusFilter, onStatusFilterChange,
    availableStatuses, primaryActionLabel, onPrimaryAction, selectionEnabled, selectedCount,
    selectionDisabled, onSelectAllVisible, onClearSelection, onBulkDelete, bulkDeletePending } = props;
  return (
    <ListHeader
      colors={colors} testIDPrefix={testIDPrefix} hasVisibleData={data.length > 0}
      searchValue={searchValue} handleSearchChange={handleSearchChange}
      statusFilter={statusFilter} onStatusFilterChange={onStatusFilterChange}
      availableStatuses={availableStatuses} primaryActionLabel={primaryActionLabel}
      onPrimaryAction={onPrimaryAction} selectionEnabled={selectionEnabled}
      selectedCount={selectedCount} selectionDisabled={selectionDisabled}
      onSelectAllVisible={onSelectAllVisible} onClearSelection={onClearSelection}
      onBulkDelete={onBulkDelete} bulkDeletePending={bulkDeletePending}
    />
  );
}

function CRMListContent<T extends CRMListItem>(props: ListContentProps<T>) {
  const { data, onRefresh, onEndReached, renderItem, testIDPrefix,
    isRefreshing = false, loadingMore = false, hasMore = false,
    selectedIds, onToggleSelect, selectionDisabled, onRowEdit,
    colors, selectionEnabled } = props;
  const HeaderComponent = () => <ListHeaderComponent {...props} />;
  return (
    <View style={[styles.container, { backgroundColor: colors.background }]} testID={`${testIDPrefix}-list`}>
      <FlatList
        data={data}
        keyExtractor={(item) => item.id}
        renderItem={({ item, index }) => renderListItem({
          item, index, testIDPrefix, colors, selectedIds,
          selectionEnabled, selectionDisabled, onToggleSelect, onRowEdit, renderItem,
        })}
        ListHeaderComponent={HeaderComponent}
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

export function CRMListScreen<T extends CRMListItem>(props: CRMListScreenProps<T>) {
  const {
    data,
    loading,
    error,
    emptyTitle,
    emptySubtitle,
    testIDPrefix,
    onRetry,
    hasData,
    selectedIds,
    onToggleSelect,
    onSearchChange,
  } = props;
  const colors = useThemeColors();
  const selectionEnabled = !!selectedIds && !!onToggleSelect;
  const selectedCount = selectedIds?.size ?? 0;

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

  return (
    <CRMListContent
      {...props}
      colors={colors}
      selectionEnabled={selectionEnabled}
      selectedCount={selectedCount}
      handleSearchChange={handleSearchChange}
    />
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center' },
  listContent: { paddingBottom: 16 },
  header: { padding: 16 },
  loadingText: { marginTop: 12, fontSize: 16 },
  errorText: { fontSize: 16, textAlign: 'center', marginBottom: 16 },
  retryButton: { paddingHorizontal: 24, paddingVertical: 12, borderRadius: 8 },
  retryButtonText: { fontSize: 16, fontWeight: '500' },
  emptyTitle: { fontSize: 18, fontWeight: '500', textAlign: 'center', marginBottom: 8 },
  emptySubtitle: { fontSize: 14, textAlign: 'center' },
  footerLoading: { padding: 16, alignItems: 'center' },
});

export default CRMListScreen;
