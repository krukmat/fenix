import React from 'react';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import type { ThemeColors } from '../../theme/types';

type ListItem = { id: string };

type ListHeaderProps = {
  colors: ThemeColors;
  testIDPrefix: string;
  hasVisibleData: boolean;
  searchValue: string;
  handleSearchChange: (value: string) => void;
  statusFilter?: string;
  onStatusFilterChange?: (status: string) => void;
  availableStatuses?: string[];
  primaryActionLabel?: string;
  onPrimaryAction?: () => void;
  selectionEnabled: boolean;
  selectedCount: number;
  selectionDisabled?: boolean;
  onSelectAllVisible?: () => void;
  onClearSelection?: () => void;
  onBulkDelete?: () => void;
  bulkDeletePending?: boolean;
};

function PrimaryAction({
  colors,
  testIDPrefix,
  label,
  disabled,
  onPress,
}: {
  colors: ThemeColors;
  testIDPrefix: string;
  label?: string;
  disabled?: boolean;
  onPress?: () => void;
}) {
  if (!label || !onPress) return null;
  return (
    <TouchableOpacity
      testID={`${testIDPrefix}-primary-action`}
      style={[styles.primaryAction, { backgroundColor: colors.primary }]}
      onPress={onPress}
      disabled={disabled}
    >
      <Text style={[styles.primaryActionText, { color: colors.onPrimary }]}>{label}</Text>
    </TouchableOpacity>
  );
}

function SelectionActions({
  colors,
  testIDPrefix,
  visible,
  selectedCount,
  disabled,
  onSelectAllVisible,
  onClearSelection,
  onBulkDelete,
  bulkDeletePending,
}: {
  colors: ThemeColors;
  testIDPrefix: string;
  visible: boolean;
  selectedCount: number;
  disabled?: boolean;
  onSelectAllVisible?: () => void;
  onClearSelection?: () => void;
  onBulkDelete?: () => void;
  bulkDeletePending?: boolean;
}) {
  if (!visible) return null;
  const isDisabled = disabled || bulkDeletePending;
  return (
    <View style={styles.selectionActions}>
      <TouchableOpacity
        testID={`${testIDPrefix}-select-all`}
        style={[styles.selectionAction, { borderColor: colors.outline }]}
        onPress={onSelectAllVisible}
        disabled={isDisabled}
      >
        <Text style={[styles.selectionActionText, { color: colors.onSurface }]}>Select all</Text>
      </TouchableOpacity>
      <TouchableOpacity
        testID={`${testIDPrefix}-clear-selection`}
        style={[styles.selectionAction, { borderColor: colors.outline }]}
        onPress={onClearSelection}
        disabled={isDisabled || selectedCount === 0}
      >
        <Text style={[styles.selectionActionText, { color: colors.onSurface }]}>Clear</Text>
      </TouchableOpacity>
      <Text testID={`${testIDPrefix}-selection-count`} style={[styles.selectionCount, { color: colors.onSurfaceVariant }]}>
        {selectedCount} selected
      </Text>
      {onBulkDelete && selectedCount > 0 && (
        <TouchableOpacity
          testID={`${testIDPrefix}-delete-selected`}
          style={[styles.selectionAction, { borderColor: colors.error, backgroundColor: colors.error }]}
          onPress={onBulkDelete}
          disabled={isDisabled}
        >
          <Text style={[styles.selectionActionText, { color: colors.onPrimary }]}>Delete selected</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

function SearchBox({
  colors,
  testIDPrefix,
  value,
  onChange,
}: {
  colors: ThemeColors;
  testIDPrefix: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <View style={[styles.searchContainer, { backgroundColor: colors.surfaceVariant, borderColor: colors.outline }]}>
      <TextInput
        style={[styles.searchInput, { color: colors.onSurface }]}
        placeholder="Search..."
        placeholderTextColor={colors.onSurfaceVariant}
        value={value}
        onChangeText={onChange}
        testID={`${testIDPrefix}-search`}
      />
    </View>
  );
}

function StatusFilters({
  colors,
  testIDPrefix,
  statusFilter,
  onStatusFilterChange,
  availableStatuses,
}: {
  colors: ThemeColors;
  testIDPrefix: string;
  statusFilter?: string;
  onStatusFilterChange?: (status: string) => void;
  availableStatuses?: string[];
}) {
  if (!availableStatuses || availableStatuses.length === 0) return null;
  return (
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
  );
}

export function ListHeader({
  colors,
  testIDPrefix,
  hasVisibleData,
  searchValue,
  handleSearchChange,
  statusFilter,
  onStatusFilterChange,
  availableStatuses,
  primaryActionLabel,
  onPrimaryAction,
  selectionEnabled,
  selectedCount,
  selectionDisabled,
  onSelectAllVisible,
  onClearSelection,
  onBulkDelete,
  bulkDeletePending,
}: ListHeaderProps) {
  return (
    <View style={styles.header}>
      <PrimaryAction
        colors={colors}
        testIDPrefix={testIDPrefix}
        label={primaryActionLabel}
        onPress={onPrimaryAction}
        disabled={selectionDisabled || bulkDeletePending}
      />
      <SelectionActions
        colors={colors}
        testIDPrefix={testIDPrefix}
        visible={selectionEnabled && hasVisibleData}
        selectedCount={selectedCount}
        disabled={selectionDisabled}
        onSelectAllVisible={onSelectAllVisible}
        onClearSelection={onClearSelection}
        onBulkDelete={onBulkDelete}
        bulkDeletePending={bulkDeletePending}
      />
      <SearchBox colors={colors} testIDPrefix={testIDPrefix} value={searchValue} onChange={handleSearchChange} />
      <StatusFilters
        colors={colors}
        testIDPrefix={testIDPrefix}
        statusFilter={statusFilter}
        onStatusFilterChange={onStatusFilterChange}
        availableStatuses={availableStatuses}
      />
    </View>
  );
}

export function SelectableItem<T extends ListItem>({
  item,
  index,
  testIDPrefix,
  colors,
  selected,
  disabled,
  onToggleSelect,
  onEdit,
  renderItem,
}: {
  item: T;
  index: number;
  testIDPrefix: string;
  colors: ThemeColors;
  selected: boolean;
  disabled?: boolean;
  onToggleSelect: (id: string) => void;
  onEdit?: (id: string) => void;
  renderItem: ({ item, index }: { item: T; index: number }) => React.ReactElement;
}) {
  return (
    <View style={styles.selectableRow}>
      <TouchableOpacity
        testID={`${testIDPrefix}-item-${index}-select`}
        accessibilityRole="checkbox"
        accessibilityState={{ checked: selected, disabled: !!disabled }}
        style={[
          styles.checkbox,
          { borderColor: selected ? colors.primary : colors.outline, backgroundColor: selected ? colors.primary : colors.surface },
        ]}
        onPress={() => onToggleSelect(item.id)}
        disabled={disabled}
      >
        <Text style={[styles.checkboxText, { color: selected ? colors.onPrimary : colors.onSurfaceVariant }]}>
          {selected ? '✓' : ''}
        </Text>
      </TouchableOpacity>
      <View style={styles.selectableContent}>{renderItem({ item, index })}</View>
      {onEdit && (
        <TouchableOpacity
          testID={`${testIDPrefix}-item-${index}-edit`}
          accessibilityLabel="Edit"
          style={styles.editButton}
          onPress={() => onEdit(item.id)}
          disabled={disabled}
        >
          <MaterialCommunityIcons name="pencil-outline" color={colors.primary} size={26} />
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  header: { padding: 16 },
  primaryAction: { minHeight: 44, borderRadius: 8, alignItems: 'center', justifyContent: 'center', marginBottom: 12 },
  primaryActionText: { fontSize: 15, fontWeight: '700' },
  selectionActions: { flexDirection: 'row', alignItems: 'center', flexWrap: 'wrap', gap: 8, marginBottom: 12 },
  selectionAction: { minHeight: 36, borderRadius: 8, borderWidth: 1, paddingHorizontal: 12, alignItems: 'center', justifyContent: 'center' },
  selectionActionText: { fontSize: 14, fontWeight: '600' },
  selectionCount: { fontSize: 13, fontWeight: '500' },
  searchContainer: { borderRadius: 8, borderWidth: 1, paddingHorizontal: 12, paddingVertical: 4 },
  searchInput: { fontSize: 16, paddingVertical: 8 },
  statusFilterContainer: { flexDirection: 'row', flexWrap: 'wrap', marginTop: 12, gap: 8 },
  statusChip: { paddingHorizontal: 12, paddingVertical: 6, borderRadius: 16 },
  statusChipText: { fontSize: 14, fontWeight: '500' },
  selectableRow: { flexDirection: 'row', alignItems: 'stretch', paddingLeft: 16 },
  checkbox: { width: 32, height: 32, borderRadius: 6, borderWidth: 1.5, alignItems: 'center', justifyContent: 'center', marginTop: 16, marginRight: 10 },
  checkboxText: { fontSize: 18, fontWeight: '800', lineHeight: 20 },
  selectableContent: { flex: 1 },
  editButton: { width: 40, height: 40, alignItems: 'center', justifyContent: 'center', marginTop: 12, marginLeft: 4, marginRight: 8 },
});
