import React, { useState } from 'react';
import { View, Text, StyleSheet, ScrollView } from 'react-native';
import { useTheme, TextInput, Button } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { useTriggerInsightsAgent } from '../../../src/hooks/useWedge';
import { wedgeHref } from '../../../src/utils/navigation';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  return useTheme().colors as ThemeColors;
}

function serializeDateInput(value: string, boundary: 'start' | 'end'): string | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;

  if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
    const suffix = boundary === 'start' ? 'T00:00:00.000Z' : 'T23:59:59.000Z';
    return `${trimmed}${suffix}`;
  }

  const parsed = new Date(trimmed);
  if (Number.isNaN(parsed.getTime())) return undefined;
  return parsed.toISOString();
}

function insightsHeaderOptions(colors: ThemeColors) {
  return {
    title: 'Insights',
    headerShown: true,
    headerBackButtonDisplayMode: 'minimal' as const,
    headerShadowVisible: false,
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.primary,
    headerTitleStyle: { color: colors.onSurface, fontSize: 18, fontWeight: '700' as const },
  };
}

export default function InsightsScreen() {
  const colors = useColors();
  const router = useRouter();
  const triggerInsights = useTriggerInsightsAgent();
  const [query, setQuery] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');

  const trimmedQuery = query.trim();
  const isRunDisabled = trimmedQuery.length === 0 || triggerInsights.isPending;

  const handleRun = async () => {
    if (!trimmedQuery) return;

    const result = await triggerInsights.mutateAsync({
      query: trimmedQuery,
      date_from: serializeDateInput(dateFrom, 'start'),
      date_to: serializeDateInput(dateTo, 'end'),
    });
    router.push(wedgeHref(`/activity/${result.runId}`));
  };

  return (
    <>
      <Stack.Screen options={insightsHeaderOptions(colors)} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]} testID="insights-screen">
        <View style={styles.section}>
          <Text style={[styles.title, { color: colors.onSurface }]}>Ask the Insights Agent</Text>
          <Text style={[styles.subtitle, { color: colors.onSurfaceVariant }]}>
            Run an ad-hoc analytical query across the CRM and inspect the result in activity detail.
          </Text>
        </View>

        <View style={styles.section}>
          <TextInput
            testID="insights-query-input"
            mode="outlined"
            label="Query"
            placeholder="Show stalled deals from the last two weeks"
            value={query}
            onChangeText={setQuery}
            multiline
          />
        </View>

        <View style={styles.section}>
          <TextInput
            testID="insights-date-from"
            mode="outlined"
            label="Date From"
            placeholder="YYYY-MM-DD"
            value={dateFrom}
            onChangeText={setDateFrom}
          />
        </View>

        <View style={styles.section}>
          <TextInput
            testID="insights-date-to"
            mode="outlined"
            label="Date To"
            placeholder="YYYY-MM-DD"
            value={dateTo}
            onChangeText={setDateTo}
          />
        </View>

        <View style={styles.section}>
          <Button
            mode="contained"
            testID="insights-run-button"
            disabled={isRunDisabled}
            onPress={() => {
              void handleRun();
            }}
          >
            {triggerInsights.isPending ? 'Running...' : 'Run Insights'}
          </Button>
        </View>
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  section: { paddingHorizontal: 16, paddingTop: 16 },
  title: { fontSize: 22, fontWeight: '700', marginBottom: 8 },
  subtitle: { fontSize: 14, lineHeight: 20 },
});
