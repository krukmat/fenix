// Task 4.2 — FR-300: Error Boundary Component

import React, { Component, ReactNode } from 'react';
import { View, Text, StyleSheet, Button } from 'react-native';
import { useTheme } from 'react-native-paper';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

class ErrorBoundaryClass extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: undefined });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <View style={styles.container}>
          <Text style={styles.title}>Something went wrong</Text>
          <Text style={styles.message}>
            {this.state.error?.message || 'An unexpected error occurred'}
          </Text>
          <Button title="Try Again" onPress={this.handleReset} />
        </View>
      );
    }

    return this.props.children;
  }
}

// Wrapper with theme access
export function ErrorBoundary({ children, fallback }: Props) {
  const theme = useTheme();
  
  const fallbackWithTheme = fallback || (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <Text style={[styles.title, { color: theme.colors.error }]}>Something went wrong</Text>
      <Text style={[styles.message, { color: theme.colors.onSurface }]}>
        An unexpected error occurred
      </Text>
    </View>
  );

  return (
    <ErrorBoundaryClass fallback={fallbackWithTheme}>
      {children}
    </ErrorBoundaryClass>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
  },
  title: {
    fontSize: 20,
    fontWeight: 'bold',
    marginBottom: 10,
  },
  message: {
    fontSize: 16,
    textAlign: 'center',
    marginBottom: 20,
  },
});

export { ErrorBoundaryClass };
