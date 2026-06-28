import React from 'react';
import { StyleSheet } from 'react-native';
import { Button, HelperText, TextInput } from 'react-native-paper';
import { useRouter } from 'expo-router';

interface AuthTextFieldProps {
  testID: string;
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  keyboardType?: 'default' | 'email-address';
  autoCapitalize?: 'none' | 'words' | 'sentences';
  autoComplete?: 'email';
  secureTextEntry?: boolean;
}

export function AuthTextField(props: AuthTextFieldProps) {
  return (
    <TextInput
      testID={props.testID}
      label={props.label}
      value={props.value}
      onChangeText={props.onChangeText}
      keyboardType={props.keyboardType}
      autoCapitalize={props.autoCapitalize}
      autoComplete={props.autoComplete}
      secureTextEntry={props.secureTextEntry}
      mode="outlined"
      dense
      style={styles.input}
      contentStyle={styles.inputContent}
    />
  );
}

export function AuthErrorMessage({ error }: { error: string }) {
  if (!error) {
    return null;
  }

  return (
    <HelperText type="error" visible>
      {error}
    </HelperText>
  );
}

export function AuthSubmitButton({
  testID,
  label,
  loading,
  onPress,
}: {
  testID: string;
  label: string;
  loading: boolean;
  onPress: () => void;
}) {
  return (
    <Button
      testID={testID}
      mode="contained"
      onPress={onPress}
      loading={loading}
      disabled={loading}
      style={styles.button}
      labelStyle={styles.buttonLabel}
    >
      {label}
    </Button>
  );
}

export function AuthRouteLink({
  testID,
  href,
  label,
}: {
  testID: string;
  href: '/login' | '/register';
  label: string;
}) {
  const router = useRouter();

  return (
    <Button
      testID={testID}
      mode="text"
      onPress={() => router.push(href)}
      style={styles.linkButton}
      labelStyle={styles.linkLabel}
    >
      {label}
    </Button>
  );
}

const styles = StyleSheet.create({
  input: {
    marginBottom: 10,
    height: 48,
    fontSize: 14,
  },
  inputContent: {
    fontSize: 14,
  },
  button: {
    marginTop: 6,
    borderRadius: 8,
  },
  buttonLabel: {
    fontSize: 14,
    fontWeight: '700',
  },
  linkButton: {
    marginTop: 10,
  },
  linkLabel: {
    fontSize: 13,
  },
});
