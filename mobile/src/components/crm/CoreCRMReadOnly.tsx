import React from 'react';
import { ActivityIndicator, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useTheme } from 'react-native-paper';
import type { ThemeColors } from '../../theme/types';
import { CRMDetailHeader } from './CRMDetailHeader';

export type DetailMeta = { label: string; value: string };

export function useCRMColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

export function asText(value: unknown, fallback = ''): string {
  return typeof value === 'string' && value.trim() !== '' ? value : fallback;
}

export function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

export function unwrapDataArray<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[];
  const record = value as { data?: unknown[] } | null | undefined;
  return Array.isArray(record?.data) ? (record.data as T[]) : [];
}

export function CRMReadOnlyRow({
  title,
  subtitle,
  meta,
  testID,
  onPress,
}: {
  title: string;
  subtitle?: string;
  meta?: string;
  testID: string;
  onPress: () => void;
}) {
  const colors = useCRMColors();
  return (
    <TouchableOpacity
      testID={testID}
      style={[styles.row, { backgroundColor: colors.surface }]}
      onPress={onPress}
    >
      <Text style={[styles.rowTitle, { color: colors.onSurface }]}>{title}</Text>
      {subtitle ? <Text style={[styles.rowSub, { color: colors.onSurfaceVariant }]}>{subtitle}</Text> : null}
      {meta ? <Text style={[styles.rowMeta, { color: colors.onSurfaceVariant }]}>{meta}</Text> : null}
    </TouchableOpacity>
  );
}

export function CRMDetailShell({
  title,
  subtitle,
  metadata,
  loading,
  error,
  testIDPrefix,
  children,
  primaryActionLabel,
  onPrimaryAction,
}: {
  title: string;
  subtitle?: string;
  metadata: DetailMeta[];
  loading: boolean;
  error?: string | null;
  testIDPrefix: string;
  children?: React.ReactNode;
  primaryActionLabel?: string;
  onPrimaryAction?: () => void;
}) {
  const colors = useCRMColors();
  if (loading) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID={`${testIDPrefix}-loading`}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }
  if (error) {
    return (
      <View style={[styles.centered, { backgroundColor: colors.background }]} testID={`${testIDPrefix}-error`}>
        <Text style={[styles.errorText, { color: colors.error }]}>{error}</Text>
      </View>
    );
  }
  return (
    <ScrollView testID={`${testIDPrefix}-screen`} style={[styles.container, { backgroundColor: colors.background }]}>
      <CRMDetailHeader title={title} subtitle={subtitle} metadata={metadata} testIDPrefix={testIDPrefix} />
      {primaryActionLabel && onPrimaryAction ? (
        <TouchableOpacity
          testID={`${testIDPrefix}-primary-action`}
          style={[styles.primaryAction, { backgroundColor: colors.primary }]}
          onPress={onPrimaryAction}
        >
          <Text style={[styles.primaryActionText, { color: colors.onPrimary }]}>{primaryActionLabel}</Text>
        </TouchableOpacity>
      ) : null}
      {children}
    </ScrollView>
  );
}

export function CRMDetailSection({
  title,
  empty,
  children,
}: {
  title: string;
  empty?: string;
  children?: React.ReactNode;
}) {
  const colors = useCRMColors();
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>{title}</Text>
      {children ?? (
        <View style={[styles.emptyCard, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurfaceVariant }}>{empty ?? 'No records'}</Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  errorText: { fontSize: 16, textAlign: 'center' },
  row: { padding: 16, marginHorizontal: 16, marginBottom: 12, borderRadius: 8, elevation: 1 },
  rowTitle: { fontSize: 16, fontWeight: '700', marginBottom: 3 },
  rowSub: { fontSize: 14 },
  rowMeta: { fontSize: 12, marginTop: 4 },
  section: { paddingHorizontal: 16, paddingBottom: 16 },
  sectionTitle: { fontSize: 17, fontWeight: '700', marginBottom: 10 },
  emptyCard: { padding: 14, borderRadius: 8 },
  primaryAction: { minHeight: 44, borderRadius: 8, alignItems: 'center', justifyContent: 'center', marginHorizontal: 16, marginBottom: 16 },
  primaryActionText: { fontSize: 15, fontWeight: '700' },
});
