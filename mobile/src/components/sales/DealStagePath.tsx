import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

export interface DealStagePathProps {
  stage: string;
  stages: string[];
  testIDPrefix?: string;
}

function toLabel(stage: string): string {
  return stage
    .split(/[_\s]+/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

export function DealStagePath({ stage, stages, testIDPrefix = 'deal-stage-path' }: DealStagePathProps) {
  const theme = useTheme();
  const activeIndex = stages.indexOf(stage);

  if (activeIndex === -1) {
    return (
      <View style={styles.container} testID={testIDPrefix}>
        <View style={styles.stepBlock}>
          <View
            testID={`${testIDPrefix}-unknown-dot`}
            style={[
              styles.dot,
              { backgroundColor: theme.colors.surfaceVariant, borderColor: theme.colors.outline },
            ]}
          />
          <Text
            variant="labelSmall"
            testID={`${testIDPrefix}-unknown-label`}
            style={[styles.label, { color: theme.colors.onSurfaceVariant, opacity: 0.7 }]}
          >
            {stage ? toLabel(stage) : 'Unknown'}
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container} testID={testIDPrefix}>
      {stages.map((step, index) => {
        const isActive = index === activeIndex;
        const stepColor = theme.colors.primary;
        return (
          <React.Fragment key={step}>
            <View style={styles.stepBlock}>
              <View
                testID={`${testIDPrefix}-${step}-dot`}
                style={[
                  styles.dot,
                  {
                    backgroundColor: isActive ? stepColor : theme.colors.surfaceVariant,
                    borderColor: isActive ? stepColor : theme.colors.outline,
                  },
                ]}
              />
              <Text
                variant="labelSmall"
                testID={`${testIDPrefix}-${step}-label`}
                style={[
                  styles.label,
                  {
                    color: isActive ? theme.colors.onSurface : theme.colors.onSurfaceVariant,
                    opacity: isActive ? 1 : 0.7,
                  },
                ]}
              >
                {toLabel(step)}
              </Text>
            </View>
            {index < stages.length - 1 ? (
              <View
                testID={`${testIDPrefix}-${step}-connector`}
                style={[styles.connector, { backgroundColor: theme.colors.outlineVariant }]}
              />
            ) : null}
          </React.Fragment>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginTop: spacing.md,
    marginBottom: spacing.sm,
  },
  stepBlock: {
    alignItems: 'center',
    width: 68,
  },
  dot: {
    width: 12,
    height: 12,
    borderRadius: radius.full,
    borderWidth: 1,
    marginBottom: spacing.xs,
  },
  label: {
    ...typography.labelMD,
    textAlign: 'center',
    lineHeight: 13,
  },
  connector: {
    flex: 1,
    height: 2,
    marginTop: 5,
    marginHorizontal: spacing.xs,
  },
});
