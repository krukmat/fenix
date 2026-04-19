// Task Mobile P1.4 — T1: Contacts list screen
import React from 'react';
import { View, StyleSheet } from 'react-native';
import { ContactsListContent } from '../../../src/components/contacts/ContactsListContent';

export default function ContactsScreen() {
  return (
    <View style={styles.container} testID="contacts-screen">
      <ContactsListContent />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
});
