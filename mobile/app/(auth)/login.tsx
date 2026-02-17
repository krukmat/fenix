// Task 4.2 — FR-300: Login Screen

import React, { useState } from 'react';
import { StyleSheet } from 'react-native';
import { TextInput, Button, HelperText } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { isAxiosError } from 'axios';

import { authApi } from '../../src/services/api';
import { useAuthStore } from '../../src/stores/authStore';
import { AuthFormLayout } from '../../src/components/ui/AuthFormLayout';

export default function LoginScreen() {
  const router = useRouter();
  const login = useAuthStore((state) => state.login);
  
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleLogin = async () => {
    if (!email || !password) {
      setError('Please enter email and password');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await authApi.login(email, password);
      
      // Response format: { token, userId, workspaceId }
      await login({
        token: response.token,
        userId: response.userId,
        workspaceId: response.workspaceId,
      });

      // Navigate to main app
      router.replace('/(tabs)/accounts');
    } catch (err) {
      console.error('Login error:', err);
      if (isAxiosError(err) && err.response?.status === 401) {
        setError('Invalid credentials');
      } else {
        setError('Login failed. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthFormLayout title="FenixCRM" subtitle="Sign in to your account">
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
        testID="password-input"
        label="Password"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        mode="outlined"
        style={styles.input}
      />

      {error ? (
        <HelperText type="error" visible>
          {error}
        </HelperText>
      ) : null}

      <Button
        testID="login-button"
        mode="contained"
        onPress={handleLogin}
        loading={loading}
        disabled={loading}
        style={styles.button}
      >
        Sign In
      </Button>

      <Button mode="text" onPress={() => router.push('/(auth)/register')} style={styles.linkButton}>
        Do not have an account? Sign up
      </Button>
    </AuthFormLayout>
  );
}

const styles = StyleSheet.create({
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
