import React from 'react';
import {
  KeyboardTypeOptions,
  StyleProp,
  StyleSheet,
  Text,
  TextInput,
  TextStyle,
  TouchableOpacity,
  View,
  ViewStyle,
} from 'react-native';
import { useTheme } from 'react-native-paper';

import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { ThemeColors } from '../../theme/types';

export type FieldProps = {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  testID: string;
  multiline?: boolean;
  keyboardType?: KeyboardTypeOptions;
  error?: string | null;
};

export type SubmitButtonProps = {
  testID: string;
  onPress: () => void;
  disabled: boolean;
  label: string;
  colors: ThemeColors;
};

export function useCRMColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

export function record(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

export function unwrapDataArray<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[];
  const payload = record(value);
  return Array.isArray(payload?.data) ? (payload.data as T[]) : [];
}

export function listItems<T>(data: { pages?: unknown[] } | undefined, normalize: (raw: unknown) => T): T[] {
  return (data?.pages ?? []).flatMap((page) => unwrapDataArray<unknown>(page).map(normalize));
}

export function Field({ label, value, onChangeText, testID, multiline, keyboardType, error }: FieldProps) {
  const colors = useCRMColors();
  return (
    <View style={baseFormStyles.field}>
      <Text style={[baseFormStyles.label, { color: colors.onSurfaceVariant }]}>{label}</Text>
      <TextInput
        testID={testID}
        value={value}
        onChangeText={onChangeText}
        multiline={multiline}
        keyboardType={keyboardType}
        style={[
          baseFormStyles.input,
          multiline ? baseFormStyles.multiline : null,
          { borderColor: colors.outline, color: colors.onSurface, backgroundColor: colors.surface },
        ]}
      />
      <FormErrorText error={error ?? null} style={[baseFormStyles.error, { color: colors.error }]} />
    </View>
  );
}

export function SubmitButton({ testID, onPress, disabled, label, colors }: SubmitButtonProps) {
  return (
    <TouchableOpacity
      testID={testID}
      style={[baseFormStyles.submit, { backgroundColor: colors.primary }, disabled ? baseFormStyles.disabled : null]}
      onPress={onPress}
      disabled={disabled}
      accessibilityState={{ disabled }}
    >
      <Text style={[baseFormStyles.submitText, { color: colors.onPrimary }]}>{label}</Text>
    </TouchableOpacity>
  );
}

export function FormErrorText({ error, style }: { error: string | null; style: StyleProp<TextStyle> }) {
  if (!error) return null;
  return <Text style={style}>{error}</Text>;
}

export function LoadingView({ testID, colors }: { testID: string; colors: ThemeColors }) {
  return (
    <View style={[baseFormStyles.centered, { backgroundColor: colors.background }]} testID={testID}>
      <Text style={{ color: colors.onSurfaceVariant }}>Loading...</Text>
    </View>
  );
}

export const baseFormStyles = StyleSheet.create({
  container: { flex: 1 } satisfies ViewStyle,
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: spacing.xl } satisfies ViewStyle,
  card: { margin: spacing.base, padding: spacing.base, borderRadius: radius.md } satisfies ViewStyle,
  field: { marginBottom: spacing.base } satisfies ViewStyle,
  label: { ...typography.labelMD, marginBottom: spacing.sm } satisfies TextStyle,
  input: { borderWidth: 1, borderRadius: radius.sm, minHeight: 44, paddingHorizontal: spacing.md, fontSize: 16 } satisfies TextStyle,
  multiline: { minHeight: 96, paddingTop: spacing.md, textAlignVertical: 'top' } satisfies TextStyle,
  error: { fontSize: 14, marginBottom: spacing.md } satisfies TextStyle,
  submit: { minHeight: 48, borderRadius: radius.sm, alignItems: 'center', justifyContent: 'center' } satisfies ViewStyle,
  disabled: { opacity: 0.7 } satisfies ViewStyle,
  submitText: { fontSize: 16, fontWeight: '700' } satisfies TextStyle,
});
