// Task 4.8 — GAP 2: Account create form screen

import React, { useState } from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { TextInput, Button, useTheme } from 'react-native-paper';
import { useRouter, Stack } from 'expo-router';
import { crmApi } from '../../../src/services/api';
import type { ThemeColors } from '../../../src/theme/types';

export default function NewAccountScreen() {
  const theme = useTheme();
  const colors = theme.colors as ThemeColors;
  const router = useRouter();
  const [name, setName] = useState('');
  const [industry, setIndustry] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSubmitting(true);
    try {
      await crmApi.createAccount({ name, industry });
      router.back();
    } catch (e) {
      console.error('create account failed:', e);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Stack.Screen options={{ title: 'New Account' }} />
      <ScrollView
        testID="account-form-screen"
        style={[styles.container, { backgroundColor: colors.background }]}
        contentContainerStyle={styles.content}
      >
        <TextInput
          testID="account-name-input"
          label="Account Name"
          value={name}
          onChangeText={setName}
          mode="outlined"
          style={styles.input}
        />
        <TextInput
          testID="account-industry-input"
          label="Industry"
          value={industry}
          onChangeText={setIndustry}
          mode="outlined"
          style={styles.input}
        />
        <Button
          testID="account-form-submit"
          mode="contained"
          onPress={handleSubmit}
          loading={submitting}
          disabled={submitting || !name.trim()}
          style={styles.button}
        >
          Create Account
        </Button>
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  content: { padding: 16 },
  input: { marginBottom: 16 },
  button: { marginTop: 8 },
});