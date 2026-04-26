// Task 4.5 — FR-230: Agent Run Detail Screen styles

import { StyleSheet } from 'react-native';
import { brandColors } from '../../theme/colors';
import { elevation, radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

const SPACE_BETWEEN = 'space-between' as const;

export const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  summaryCard: {
    margin: spacing.base,
    padding: spacing.base,
    borderRadius: radius.md,
    ...elevation.card,
  },
  summaryHeader: {
    flexDirection: 'row',
    justifyContent: SPACE_BETWEEN,
    alignItems: 'center',
    marginBottom: spacing.md,
  },
  agentName: {
    ...typography.headingMD,
  },
  statusBadge: {
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderRadius: radius.full,
  },
  statusBadgeText: {
    color: brandColors.onError,
    ...typography.labelMD,
  },
  summaryMetrics: {
    flexDirection: 'row',
    justifyContent: 'space-around',
  },
  summaryMetric: {
    fontSize: 12,
  },
  section: { padding: spacing.base },
  sectionTitle: {
    ...typography.eyebrow,
    marginBottom: spacing.md,
  },
  codeBlock: {
    padding: spacing.md,
    borderRadius: radius.xs,
  },
  evidenceCard: {
    padding: spacing.md,
    borderRadius: radius.md,
    marginBottom: spacing.sm,
  },
  evidenceHeader: {
    flexDirection: 'row',
    justifyContent: SPACE_BETWEEN,
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  scoreBadge: {
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
    borderRadius: radius.full,
  },
  scoreBadgeText: {
    color: brandColors.onError,
    fontSize: 10,
    fontWeight: '600',
  },
  reasoningStep: {
    padding: spacing.md,
    borderRadius: radius.sm,
    marginBottom: spacing.sm,
    borderLeftWidth: 3,
    borderLeftColor: brandColors.primary,
  },
  toolCallCard: {
    padding: spacing.md,
    borderRadius: radius.md,
    marginBottom: spacing.sm,
  },
  toolName: {
    fontSize: 14,
    fontWeight: '600',
  },
  outputBlock: {
    padding: spacing.base,
    borderRadius: radius.md,
  },
  auditEvent: {
    padding: spacing.md,
    borderRadius: radius.md,
    marginBottom: spacing.sm,
  },
  auditHeader: {
    flexDirection: 'row',
    justifyContent: SPACE_BETWEEN,
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  auditFooter: {
    flexDirection: 'row',
    justifyContent: SPACE_BETWEEN,
    alignItems: 'center',
  },
  outcomeBadge: {
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
    borderRadius: radius.full,
  },
  outcomeBadgeText: {
    color: brandColors.onError,
    fontSize: 10,
    fontWeight: '600',
  },
  backButton: {
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xl,
    borderRadius: radius.sm,
  },
  backButtonText: {
    color: brandColors.onError,
    fontSize: 14,
    fontWeight: '600',
  },
});
