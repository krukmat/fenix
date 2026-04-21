// Task Mobile P1.4 — CRM hub: grid of 4 entity cards

import React from 'react';
import { View, StyleSheet, ScrollView } from 'react-native';
import { Card, Text, useTheme } from 'react-native-paper';
import { Stack, useRouter } from 'expo-router';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import type { ComponentProps } from 'react';

type CRMIconName = ComponentProps<typeof MaterialCommunityIcons>['name'];

interface EntityCard {
  label: string;
  iconName: CRMIconName;
  route: string;
  testID: string;
}

const ENTITIES: EntityCard[] = [
  { label: 'Accounts', iconName: 'domain', route: '/crm/accounts', testID: 'crm-hub-accounts' },
  { label: 'Contacts', iconName: 'account-group', route: '/crm/contacts', testID: 'crm-hub-contacts' },
  { label: 'Leads', iconName: 'target', route: '/crm/leads', testID: 'crm-hub-leads' },
  { label: 'Deals', iconName: 'handshake', route: '/crm/deals', testID: 'crm-hub-deals' },
  { label: 'Cases', iconName: 'ticket-confirmation', route: '/crm/cases', testID: 'crm-hub-cases' },
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
          style={styles.heading}
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
                <MaterialCommunityIcons name={entity.iconName} size={28} color="#3B82F6" />
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
  heading: { fontSize: 11, fontWeight: '700', letterSpacing: 1.2, textTransform: 'uppercase', color: '#8899AA', marginBottom: 16 },
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: 12 },
  card: { width: '47%', backgroundColor: '#111620', borderWidth: 1, borderColor: '#1E2B3E', borderRadius: 12 },
  cardContent: { alignItems: 'center', paddingVertical: 24, gap: 8 },
});
