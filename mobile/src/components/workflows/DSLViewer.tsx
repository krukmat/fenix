// Task Mobile P1.3 — Read-only monospace DSL viewer with horizontal scroll

import React from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { Text, Surface, useTheme } from 'react-native-paper';

interface DSLViewerProps {
  dsl: string;
  testIDPrefix?: string;
}

export function DSLViewer({ dsl, testIDPrefix = 'dsl-viewer' }: DSLViewerProps) {
  const theme = useTheme();

  return (
    <Surface
      style={[styles.surface, { backgroundColor: theme.colors.surfaceVariant }]}
      elevation={0}
      testID={testIDPrefix}
    >
      <ScrollView horizontal showsHorizontalScrollIndicator testID={`${testIDPrefix}-hscroll`}>
        <ScrollView showsVerticalScrollIndicator testID={`${testIDPrefix}-vscroll`}>
          <Text
            style={[styles.code, { color: theme.colors.onSurfaceVariant }]}
            testID={`${testIDPrefix}-code`}
          >
            {dsl}
          </Text>
        </ScrollView>
      </ScrollView>
    </Surface>
  );
}

const styles = StyleSheet.create({
  surface: { borderRadius: 8, overflow: 'hidden' },
  code: {
    fontFamily: 'monospace',
    fontSize: 13,
    lineHeight: 20,
    padding: 16,
  },
});
