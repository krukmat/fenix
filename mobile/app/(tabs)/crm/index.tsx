// Task Mobile P1.4 — CRM hub: grid of 4 entity cards

import React from 'react';
import { View, StyleSheet, ScrollView } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';

interface EntityCard {
  label: string;
  icon: string;
  route: string;
  testID: string;
}

const ENTITIES: EntityCard[] = [
  { label: 'Accounts', icon: '🏢', route: '/crm/accounts', testID: 'crm-hub-accounts' },
  { label: 'Contacts', icon: '👤', route: '/crm/contacts', testID: 'crm-hub-contacts' },
  { label: 'Leads', icon: '🎯', route: '/crm/leads', testID: 'crm-hub-leads' },
  { label: 'Deals', icon: '💼', route: '/crm/deals', testID: 'crm-hub-deals' },
  { label: 'Cases', icon: '🎫', route: '/crm/cases', testID: 'crm-hub-cases' },
];

export default function CRMHubScreen() {
  const theme = useTheme();
  const router = useRouter();

  return (
    <>
      <Stack.Screen options={{ title: 'CRM', headerShown: true }} />
      <ScrollView
        style={[styles.container, { backgroundColor: theme.colors.background }]}
        contentContainerStyle={styles.content}
        testID="crm-hub"
      >
        <Text
          variant="titleMedium"
          style={[styles.heading, { color: theme.colors.onSurfaceVariant }]}
        >
          Select entity
        </Text>
        <View style={styles.grid}>
          {ENTITIES.map((entity) => (
            <Card
              key={entity.label}
              testID={entity.testID}
              style={styles.card}
              onPress={() => router.push(entity.route as never)}
            >
              <Card.Content style={styles.cardContent}>
                <Text style={styles.icon}>{entity.icon}</Text>
                <Text variant="titleMedium">{entity.label}</Text>
              </Card.Content>
            </Card>
          ))}
        </View>
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: 16 },
  heading: { marginBottom: 16 },
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: 12 },
  card: { width: '47%' },
  cardContent: { alignItems: 'center', paddingVertical: 24, gap: 8 },
  icon: { fontSize: 32 },
});
