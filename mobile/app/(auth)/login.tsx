// Task 4.2 — FR-300: Login Screen

import React, { useState } from 'react';
import { isAxiosError } from 'axios';

import { authApi } from '../../src/services/api';
import { useAuthStore } from '../../src/stores/authStore';
import { AuthFormLayout } from '../../src/components/ui/AuthFormLayout';
import {
  AuthErrorMessage,
  AuthRouteLink,
  AuthSubmitButton,
  AuthTextField,
} from '../../src/components/ui/AuthControls';

export default function LoginScreen() {
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

      await login({
        token: response.token,
        userId: response.userId,
        workspaceId: response.workspaceId,
      });
    } catch (err) {
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
    <AuthFormLayout title="FenixCRM" subtitle="Sign in to your account" testID="login-screen">
      <AuthTextField
        testID="login-email-input"
        label="Email"
        value={email}
        onChangeText={setEmail}
        keyboardType="email-address"
        autoCapitalize="none"
        autoComplete="email"
      />

      <AuthTextField
        testID="login-password-input"
        label="Password"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
      />

      <AuthErrorMessage error={error} />

      <AuthSubmitButton testID="login-submit-button" label="Sign In" loading={loading} onPress={handleLogin} />

      <AuthRouteLink
        testID="go-to-register-link"
        href="/register"
        label="Do not have an account? Sign up"
      />
    </AuthFormLayout>
  );
}
