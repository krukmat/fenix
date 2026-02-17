// Task 4.2 — FR-300: Register Screen

import React, { useState } from 'react';
import { StyleSheet } from 'react-native';
import { TextInput, Button, HelperText } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { isAxiosError } from 'axios';

import { authApi } from '../../src/services/api';
import { useAuthStore } from '../../src/stores/authStore';
import { AuthFormLayout } from '../../src/components/ui/AuthFormLayout';

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
  onGoToLogin: () => void;
}

function RegisterForm(props: RegisterFormProps) {
  return (
    <>
      <TextInput
        testID="name-input"
        label="Display Name"
        value={props.displayName}
        onChangeText={props.onDisplayNameChange}
        autoCapitalize="words"
        mode="outlined"
        style={styles.input}
      />

      <TextInput
        testID="email-input"
        label="Email"
        value={props.email}
        onChangeText={props.onEmailChange}
        keyboardType="email-address"
        autoCapitalize="none"
        autoComplete="email"
        mode="outlined"
        style={styles.input}
      />

      <TextInput
        testID="workspace-input"
        label="Workspace Name"
        value={props.workspaceName}
        onChangeText={props.onWorkspaceChange}
        autoCapitalize="words"
        mode="outlined"
        style={styles.input}
      />

      <TextInput
        testID="password-input"
        label="Password"
        value={props.password}
        onChangeText={props.onPasswordChange}
        secureTextEntry
        mode="outlined"
        style={styles.input}
      />

      {props.error ? (
        <HelperText type="error" visible>
          {props.error}
        </HelperText>
      ) : null}

      <Button
        testID="register-button"
        mode="contained"
        onPress={props.onSubmit}
        loading={props.loading}
        disabled={props.loading}
        style={styles.button}
      >
        Sign Up
      </Button>

      <Button mode="text" onPress={props.onGoToLogin} style={styles.linkButton}>
        Already have an account? Sign in
      </Button>
    </>
  );
}

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
    } catch (err) {
      console.error('Register error:', err);
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
    <AuthFormLayout title="Create Account" subtitle="Sign up for FenixCRM">
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
        onGoToLogin={() => router.push('/(auth)/login')}
      />
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
