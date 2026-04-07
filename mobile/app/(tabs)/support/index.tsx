// W2-T1 (mobile_wedge_harmonization_plan): Support tab — case list entry point
// Full implementation: W3-T1
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

export default function SupportScreen() {
  return (
    <View style={styles.container} testID="support-screen">
      <Text style={styles.title}>Support</Text>
      <Text style={styles.subtitle}>Cases · Agent Runs · Copilot</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  title: { fontSize: 24, fontWeight: '700', marginBottom: 8 },
  subtitle: { fontSize: 14, color: '#757575' },
});
