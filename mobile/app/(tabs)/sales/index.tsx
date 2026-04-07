// W2-T1 (mobile_wedge_harmonization_plan): Sales tab — accounts and deals entry point
// Full implementation: W4-T1
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

export default function SalesScreen() {
  return (
    <View style={styles.container} testID="sales-screen">
      <Text style={styles.title}>Sales</Text>
      <Text style={styles.subtitle}>Accounts · Deals · Brief</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  title: { fontSize: 24, fontWeight: '700', marginBottom: 8 },
  subtitle: { fontSize: 14, color: '#757575' },
});
