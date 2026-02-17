// Task 4.2 — FR-300: Agents Placeholder (Task 4.5)
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from 'react-native-paper';

export default function AgentsScreen() {
  const theme = useTheme();
  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <Text style={{ color: theme.colors.onBackground, fontSize: 18 }}>
        Agent Runs
      </Text>
      <Text style={{ color: theme.colors.onSurfaceVariant, marginTop: 16 }}>
        Coming in Task 4.5
      </Text>
    </View>
  );
}
const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 },
});
