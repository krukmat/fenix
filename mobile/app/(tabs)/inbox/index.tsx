// W2-T1 (mobile_wedge_harmonization_plan): Inbox tab — unified approvals, handoffs, signals
// Full implementation: W3-T5
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

export default function InboxScreen() {
  return (
    <View style={styles.container} testID="inbox-screen">
      <Text style={styles.title}>Inbox</Text>
      <Text style={styles.subtitle}>Approvals · Handoffs · Signals</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  title: { fontSize: 24, fontWeight: '700', marginBottom: 8 },
  subtitle: { fontSize: 14, color: '#757575' },
});
