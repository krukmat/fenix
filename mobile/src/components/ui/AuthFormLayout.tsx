import React from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from 'react-native';
import { Text } from 'react-native-paper';

interface AuthFormLayoutProps {
  title: string;
  subtitle: string;
  children: React.ReactNode;
  testID?: string;
}

export function AuthFormLayout({ title, subtitle, children, testID }: AuthFormLayoutProps) {
  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      testID={testID}
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
    backgroundColor: '#0A0D12',
  },
  scrollContent: {
    flexGrow: 1,
    justifyContent: 'center',
    paddingHorizontal: 24,
    paddingVertical: 32,
  },
  form: {
    width: '100%',
    maxWidth: 360,
    alignSelf: 'center',
    backgroundColor: '#111620',
    paddingHorizontal: 20,
    paddingVertical: 22,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: '#1E2B3E',
    borderLeftWidth: 3,
    borderLeftColor: '#3B82F6',
    shadowColor: '#3B82F6',
    shadowOpacity: 0.12,
    shadowOffset: { width: 0, height: 4 },
    shadowRadius: 16,
    elevation: 6,
  },
  title: {
    textAlign: 'left',
    marginBottom: 4,
    fontWeight: '700',
    color: '#F0F4FF',
    fontSize: 22,
    letterSpacing: -0.2,
  },
  subtitle: {
    textAlign: 'left',
    marginBottom: 20,
    color: '#8899AA',
    fontSize: 12,
  },
});
