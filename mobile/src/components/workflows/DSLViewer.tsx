// Task Mobile P1.4 — T4: DSLViewer component
import React from 'react';
import { ScrollView, Text, StyleSheet, View } from 'react-native';
import { useTheme } from 'react-native-paper';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

interface DSLViewerProps {
  dsl: string;
  testIDPrefix?: string;
}

export function DSLViewer({ dsl, testIDPrefix = 'dsl-viewer' }: DSLViewerProps) {
  const theme = useTheme();

  return (
    <View
      style={[styles.container, { backgroundColor: theme.colors.surfaceVariant }]}
      testID={`${testIDPrefix}-container`}
    >
      <ScrollView horizontal showsHorizontalScrollIndicator>
        <ScrollView nestedScrollEnabled>
          <Text
            style={[styles.code, { color: theme.colors.onSurface }]}
            testID={`${testIDPrefix}-code`}
            selectable
          >
            {dsl || '# No DSL source'}
          </Text>
        </ScrollView>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { borderRadius: radius.sm, padding: spacing.md, maxHeight: 200 },
  code: { ...typography.mono, lineHeight: 18 },
});
