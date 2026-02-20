// Task 4.5 — FR-230: Agent Run Detail Screen

import React from 'react';
import { View, Text, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { useAgentRun } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';
import { renderContent } from './[id].helpers';
import { styles } from './[id].styles';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

export default function AgentRunDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useAgentRun(id);
  const run = data?.data;
  const title = run ? `Run: ${run.agent_name}` : 'Agent Run';

  return (
    <>
      <Stack.Screen options={{ title, headerBackTitle: 'Back' }} />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]}>
        {isLoading ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>
              Loading agent run...
            </Text>
          </View>
        ) : error || !run ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <Text style={{ color: colors.error, fontSize: 16 }}>
              {error?.message || 'Agent run not found'}
            </Text>
            <TouchableOpacity
              style={[styles.backButton, { marginTop: 16, backgroundColor: colors.primary }]}
              onPress={() => router.push('/agents')}
            >
              <Text style={styles.backButtonText}>Back to List</Text>
            </TouchableOpacity>
          </View>
        ) : (
          renderContent(run, colors)
        )}
      </ScrollView>
    </>
  );
}
