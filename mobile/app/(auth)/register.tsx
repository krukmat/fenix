// Task 4.2 — FR-300: Register Screen

import React, { useState } from 'react';
import { View, StyleSheet, KeyboardAvoidingView, Platform, ScrollView } from 'react-native';
import { TextInput, Button, Text, HelperText } from 'react-native-paper';
import { useRouter } from 'expo-router';

import { authApi } from '../../src/services/api';
import { useAuthStore } from '../../src/stores/authStore';

export default function RegisterScreen() {
  const router = useRouter();
  const login = useAuthStore((state) => state.login);
  
  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [workspaceName, setWorkspaceName] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleRegister = async () => {
    if (!displayName || !email || !workspaceName || !password) {
      setError('Please fill in all fields');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await authApi.register(displayName, email, password, workspaceName);
      
      // Response format: { token, userId, workspaceId }
      await login({
        token: response.token,
        userId: response.userId,
        workspaceId: response.workspaceId,
      });

      // Navigate to main app
      router.replace('/(tabs)/accounts');
    } catch (err: any) {
      console.error('Register error:', err);
      if (err.response?.status === 409) {
        setError('User already exists');
      } else {
        setError('Registration failed. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <KeyboardAvoidingView 
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView contentContainerStyle={styles.scrollContent}>
        <View style={styles.form}>
          <Text variant="headlineMedium" style={styles.title}>
            Create Account
          </Text>
          <Text variant="bodyLarge" style={styles.subtitle}>
            Sign up for FenixCRM
          </Text>

          <TextInput
            testID="name-input"
            label="Display Name"
            value={displayName}
            onChangeText={setDisplayName}
            autoCapitalize="words"
            mode="outlined"
            style={styles.input}
          />

          <TextInput
            testID="email-input"
            label="Email"
            value={email}
            onChangeText={setEmail}
            keyboardType="email-address"
            autoCapitalize="none"
            autoComplete="email"
            mode="outlined"
            style={styles.input}
          />

          <TextInput
            testID="workspace-input"
            label="Workspace Name"
            value={workspaceName}
            onChangeText={setWorkspaceName}
            autoCapitalize="words"
            mode="outlined"
            style={styles.input}
          />

          <TextInput
            testID="password-input"
            label="Password"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            mode="outlined"
            style={styles.input}
          />

          {error ? (
            <HelperText type="error" visible={true}>
              {error}
            </HelperText>
          ) : null}

          <Button
            testID="register-button"
            mode="contained"
            onPress={handleRegister}
            loading={loading}
            disabled={loading}
            style={styles.button}
          >
            Sign Up
          </Button>

          <Button
            mode="text"
            onPress={() => router.push('/(auth)/login')}
            style={styles.linkButton}
          >
            Already have an account? Sign in
          </Button>
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
  input: {
    marginBottom: 16,
  },
  button: {
    marginTop: 8,
    paddingVertical: 4,
  },
  linkButton: {
    marginTop: 16,
  },
});
