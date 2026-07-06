import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

export interface CaseStatusPathProps {
  status: string;
  knownStatuses: string[];
  testIDPrefix?: string;
}

function toLabel(status: string): string {
  return status
    .split(/[_\s]+/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

export function CaseStatusPath({
  status,
  knownStatuses,
  testIDPrefix = 'case-status-path',
}: CaseStatusPathProps) {
  const { colors } = useTheme();
  const activeIndex = knownStatuses.indexOf(status);

  if (activeIndex === -1) {
    return (
      <View style={styles.container} testID={testIDPrefix}>
        <View style={styles.stepBlock}>
          <View
            testID={`${testIDPrefix}-unknown-dot`}
            style={[
              styles.dot,
              { backgroundColor: colors.surfaceVariant, borderColor: colors.outline },
            ]}
          />
          <Text
            variant="labelSmall"
            testID={`${testIDPrefix}-unknown-label`}
            style={[styles.label, { color: colors.onSurfaceVariant, opacity: 0.7 }]}
          >
            {status ? toLabel(status) : 'Unknown'}
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container} testID={testIDPrefix}>
      {knownStatuses.map((step, index) => {
        const isActive = index === activeIndex;
        return (
          <React.Fragment key={step}>
            <View style={styles.stepBlock}>
              <View
                testID={`${testIDPrefix}-${step}-dot`}
                style={[
                  styles.dot,
                  {
                    backgroundColor: isActive ? colors.primary : colors.surfaceVariant,
                    borderColor: isActive ? colors.primary : colors.outline,
                  },
                ]}
              />
              <Text
                variant="labelSmall"
                testID={`${testIDPrefix}-${step}-label`}
                style={[
                  styles.label,
                  {
                    color: isActive ? colors.onSurface : colors.onSurfaceVariant,
                    opacity: isActive ? 1 : 0.7,
                  },
                ]}
              >
                {toLabel(step)}
              </Text>
            </View>
            {index < knownStatuses.length - 1 ? (
              <View
                testID={`${testIDPrefix}-${step}-connector`}
                style={[styles.connector, { backgroundColor: colors.outlineVariant }]}
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
