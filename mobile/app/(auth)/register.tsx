// Task 4.2 — FR-300: Register Screen

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

interface RegisterFormProps {
  displayName: string;
  email: string;
  workspaceName: string;
  password: string;
  error: string;
  loading: boolean;
  onDisplayNameChange: (value: string) => void;
  onEmailChange: (value: string) => void;
  onWorkspaceChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: () => void;
}

function RegisterForm(props: RegisterFormProps) {
  return (
    <>
      <AuthTextField
        testID="register-name-input"
        label="Display Name"
        value={props.displayName}
        onChangeText={props.onDisplayNameChange}
        autoCapitalize="words"
      />

      <AuthTextField
        testID="register-email-input"
        label="Email"
        value={props.email}
        onChangeText={props.onEmailChange}
        keyboardType="email-address"
        autoCapitalize="none"
        autoComplete="email"
      />

      <AuthTextField
        testID="register-workspace-input"
        label="Workspace Name"
        value={props.workspaceName}
        onChangeText={props.onWorkspaceChange}
        autoCapitalize="words"
      />

      <AuthTextField
        testID="register-password-input"
        label="Password"
        value={props.password}
        onChangeText={props.onPasswordChange}
        secureTextEntry
      />

      <AuthErrorMessage error={props.error} />

      <AuthSubmitButton
        testID="register-submit-button"
        label="Sign Up"
        loading={props.loading}
        onPress={props.onSubmit}
      />

      <AuthRouteLink
        testID="go-to-login-link"
        href="/login"
        label="Already have an account? Sign in"
      />
    </>
  );
}

export default function RegisterScreen() {
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
    } catch (err) {
      if (isAxiosError(err) && err.response?.status === 409) {
        setError('User already exists');
      } else {
        setError('Registration failed. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthFormLayout title="Create Account" subtitle="Sign up for FenixCRM" testID="register-screen">
      <RegisterForm
        displayName={displayName}
        email={email}
        workspaceName={workspaceName}
        password={password}
        error={error}
        loading={loading}
        onDisplayNameChange={setDisplayName}
        onEmailChange={setEmail}
        onWorkspaceChange={setWorkspaceName}
        onPasswordChange={setPassword}
        onSubmit={handleRegister}
      />
    </AuthFormLayout>
  );
}
