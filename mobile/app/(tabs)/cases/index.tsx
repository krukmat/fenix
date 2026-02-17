// Task 4.2 — FR-300: Cases List Placeholder
import React from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useCases } from '../../../src/hooks/useCRM';

export default function CasesListScreen() {
  const theme = useTheme();
  const { data, isLoading, error } = useCases();
  if (isLoading) return <View style={[styles.centered, { backgroundColor: theme.colors.background }]}><ActivityIndicator size="large" /></View>;
  if (error) return <View style={[styles.centered, { backgroundColor: theme.colors.background }]}><Text style={{ color: theme.colors.error }}>Error loading cases</Text></View>;
  const cases = data?.data || [];
  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <FlatList data={cases} keyExtractor={(item) => item.id || Math.random().toString()} renderItem={({ item }) => (
        <View style={[styles.item, { backgroundColor: theme.colors.surface }]}>
          <Text style={{ color: theme.colors.onSurface }}>{item.title || 'Unnamed Case'}</Text>
          <Text style={{ color: theme.colors.onSurfaceVariant }}>{item.status || 'Open'}</Text>
        </View>
      )} ListEmptyComponent={<View style={styles.emptyContainer}><Text>No cases found</Text></View>} contentContainerStyle={styles.listContent} />
    </View>
  );
}
const styles = StyleSheet.create({
  container: { flex: 1 }, centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  listContent: { padding: 16 }, item: { padding: 16, marginBottom: 12, borderRadius: 8 },
  emptyContainer: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 40 },
});
