import React from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from 'react-native';
import { Text } from 'react-native-paper';

interface AuthFormLayoutProps {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}

export function AuthFormLayout({ title, subtitle, children }: AuthFormLayoutProps) {
  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView contentContainerStyle={styles.scrollContent}>
        <View style={styles.form}>
          <Text variant="headlineMedium" style={styles.title}>
            {title}
          </Text>
          <Text variant="bodyLarge" style={styles.subtitle}>
            {subtitle}
          </Text>
          {children}
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
    justifyContent: 'center',
    padding: 20,
  },
  form: {
    width: '100%',
    maxWidth: 400,
    alignSelf: 'center',
  },
  title: {
    textAlign: 'center',
    marginBottom: 8,
    fontWeight: 'bold',
    color: '#1565C0',
  },
  subtitle: {
    textAlign: 'center',
    marginBottom: 32,
    color: '#666',
  },
});
