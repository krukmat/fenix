import React from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { spacing } from '../../theme/spacing';

interface CenteredStateProps {
  testID: string;
  backgroundColor: string;
  children: React.ReactNode;
}

function CenteredState({ testID, backgroundColor, children }: CenteredStateProps) {
  return (
    <View style={[styles.centered, { backgroundColor }]} testID={testID}>
      {children}
    </View>
  );
}

export function CenteredLoadingState({
  testID,
  backgroundColor,
  indicatorColor,
  message,
  messageColor,
}: {
  testID: string;
  backgroundColor: string;
  indicatorColor: string;
  message: string;
  messageColor: string;
}) {
  return (
    <CenteredState testID={testID} backgroundColor={backgroundColor}>
      <ActivityIndicator size="large" color={indicatorColor} />
      <Text style={[styles.message, { color: messageColor }]}>{message}</Text>
    </CenteredState>
  );
}

export function CenteredMessageState({
  testID,
  backgroundColor,
  message,
  messageColor,
}: {
  testID: string;
  backgroundColor: string;
  message: string;
  messageColor: string;
}) {
  return (
    <CenteredState testID={testID} backgroundColor={backgroundColor}>
      <Text style={[styles.message, { color: messageColor }]}>{message}</Text>
    </CenteredState>
  );
}

const styles = StyleSheet.create({
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
  },
  message: {
    marginTop: spacing.md,
    textAlign: 'center',
  },
});
