// Task 4.5 — FR-230: Agent Run Detail Screen styles

import { StyleSheet } from 'react-native';

export const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  summaryCard: {
    margin: 16,
    padding: 16,
    borderRadius: 8,
    elevation: 2,
  },
  summaryHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  agentName: {
    fontSize: 20,
    fontWeight: '600',
  },
  statusBadge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 4,
  },
  statusBadgeText: {
    color: '#FFF',
    fontSize: 11,
    fontWeight: '600',
  },
  summaryMetrics: {
    flexDirection: 'row',
    justifyContent: 'space-around',
  },
  summaryMetric: {
    fontSize: 12,
  },
  section: { padding: 16 },
  sectionTitle: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 12,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  codeBlock: {
    padding: 12,
    borderRadius: 6,
    fontFamily: 'monospace',
  },
  evidenceCard: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
  },
  evidenceHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  scoreBadge: {
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  scoreBadgeText: {
    color: '#FFF',
    fontSize: 10,
    fontWeight: '600',
  },
  reasoningStep: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
    borderLeftWidth: 3,
    borderLeftColor: '#2196F3',
  },
  toolCallCard: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
  },
  toolName: {
    fontSize: 14,
    fontWeight: '600',
  },
  outputBlock: {
    padding: 16,
    borderRadius: 6,
  },
  auditEvent: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
  },
  auditHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  auditFooter: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  outcomeBadge: {
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  outcomeBadgeText: {
    color: '#FFF',
    fontSize: 10,
    fontWeight: '600',
  },
  backButton: {
    paddingVertical: 8,
    paddingHorizontal: 24,
    borderRadius: 8,
  },
  backButtonText: {
    color: '#FFF',
    fontSize: 14,
    fontWeight: '600',
  },
});
