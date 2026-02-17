// Task 4.2 — FR-300: Accounts List Placeholder

import React from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useAccounts } from '../../../src/hooks/useCRM';

export default function AccountsListScreen() {
  const theme = useTheme();
  const { data, isLoading, error } = useAccounts();

  if (isLoading) {
    return (
      <View style={[styles.container, styles.centered, { backgroundColor: theme.colors.background }]}>
        <ActivityIndicator size="large" color={theme.colors.primary} />
      </View>
    );
  }

  if (error) {
    return (
      <View style={[styles.container, styles.centered, { backgroundColor: theme.colors.background }]}>
        <Text style={[styles.errorText, { color: theme.colors.error }]}>
          Error loading accounts
        </Text>
      </View>
    );
  }

  const accounts = data?.data || [];

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <FlatList
        data={accounts}
        keyExtractor={(item) => item.id || Math.random().toString()}
        renderItem={({ item }) => (
          <View style={[styles.item, { backgroundColor: theme.colors.surface }]}>
            <Text style={[styles.itemTitle, { color: theme.colors.onSurface }]}>
              {item.name || 'Unnamed Account'}
            </Text>
            <Text style={{ color: theme.colors.onSurfaceVariant }}>
              {item.industry || 'No industry'}
            </Text>
          </View>
        )}
        ListEmptyComponent={
          <View style={styles.emptyContainer}>
            <Text style={{ color: theme.colors.onSurfaceVariant }}>
              No accounts found
            </Text>
          </View>
        }
        contentContainerStyle={styles.listContent}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  centered: {
    justifyContent: 'center',
    alignItems: 'center',
  },
  listContent: {
    padding: 16,
  },
  item: {
    padding: 16,
    marginBottom: 12,
    borderRadius: 8,
    elevation: 2,
  },
  itemTitle: {
    fontSize: 16,
    fontWeight: '500',
    marginBottom: 4,
  },
  errorText: {
    fontSize: 16,
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 40,
  },
});
