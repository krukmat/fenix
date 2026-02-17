// Task 4.2 — FR-300: Contacts List Placeholder
import React from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useContacts } from '../../../src/hooks/useCRM';

export default function ContactsListScreen() {
  const theme = useTheme();
  const { data, isLoading, error } = useContacts();
  if (isLoading) return <View style={[styles.centered, { backgroundColor: theme.colors.background }]}><ActivityIndicator size="large" /></View>;
  if (error) return <View style={[styles.centered, { backgroundColor: theme.colors.background }]}><Text style={{ color: theme.colors.error }}>Error loading contacts</Text></View>;
  const contacts = data?.data || [];
  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <FlatList data={contacts} keyExtractor={(item) => item.id || Math.random().toString()} renderItem={({ item }) => (
        <View style={[styles.item, { backgroundColor: theme.colors.surface }]}>
          <Text style={{ color: theme.colors.onSurface }}>{item.firstName} {item.lastName}</Text>
        </View>
      )} ListEmptyComponent={<View style={styles.emptyContainer}><Text>No contacts found</Text></View>} contentContainerStyle={styles.listContent} />
    </View>
  );
}
const styles = StyleSheet.create({
  container: { flex: 1 }, centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  listContent: { padding: 16 }, item: { padding: 16, marginBottom: 12, borderRadius: 8 },
  emptyContainer: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 40 },
});
