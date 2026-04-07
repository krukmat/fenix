// W2-T1 (mobile_wedge_harmonization_plan): Governance tab — quota states and usage summary
// Full implementation: W5-T3 (pending wave definition)
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

export default function GovernanceScreen() {
  return (
    <View style={styles.container} testID="governance-screen">
      <Text style={styles.title}>Governance</Text>
      <Text style={styles.subtitle}>Usage · Quotas · Policies</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  title: { fontSize: 24, fontWeight: '700', marginBottom: 8 },
  subtitle: { fontSize: 14, color: '#757575' },
});
