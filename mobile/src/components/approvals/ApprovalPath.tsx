import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { semanticColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

const APPROVAL_STEPS = ['pending', 'approved', 'rejected', 'expired', 'cancelled'] as const;

type ApprovalStep = typeof APPROVAL_STEPS[number];
type ApprovalPathStatus = ApprovalStep | 'denied';

const STEP_LABELS: Record<ApprovalStep, string> = {
  pending: 'Pending',
  approved: 'Approved',
  rejected: 'Rejected',
  expired: 'Expired',
  cancelled: 'Cancelled',
};

export const APPROVAL_STEP_COLORS: Record<ApprovalStep, string> = {
  pending: semanticColors.info,
  approved: semanticColors.success,
  rejected: semanticColors.warning,
  expired: semanticColors.confidenceLow,
  cancelled: semanticColors.confidenceLow,
};

export function normalizeApprovalStatus(status: ApprovalPathStatus): ApprovalStep {
  return status === 'denied' ? 'rejected' : status;
}

export interface ApprovalPathProps {
  status: ApprovalPathStatus;
  testIDPrefix?: string;
}

export function ApprovalPath({ status, testIDPrefix = 'approval-path' }: ApprovalPathProps) {
  const theme = useTheme();
  const activeStatus = normalizeApprovalStatus(status);

  return (
    <View style={styles.container} testID={testIDPrefix}>
      {APPROVAL_STEPS.map((step, index) => {
        const isActive = step === activeStatus;
        const stepColor = APPROVAL_STEP_COLORS[step];
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
                {STEP_LABELS[step]}
              </Text>
            </View>
            {index < APPROVAL_STEPS.length - 1 ? (
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
    width: 52,
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
