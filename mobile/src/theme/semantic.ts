// ui-redesign-command-center: shared color helpers
import { semanticColors } from './colors';

export function getAgentStatusColor(status: string): string {
  const map: Record<string, string> = {
    completed:               semanticColors.success,
    completed_with_warnings: semanticColors.warning,
    awaiting_approval:       '#3B82F6',
    handed_off:              '#A78BFA',
    denied_by_policy:        '#EF4444',
    abstained:               semanticColors.confidenceLow,
    failed:                  '#DC2626',
    won:                     semanticColors.success,
    lost:                    '#EF4444',
    open:                    '#3B82F6',
    high:                    '#EF4444',
    medium:                  semanticColors.warning,
    low:                     semanticColors.success,
    success:                 semanticColors.success,
    denied:                  '#EF4444',
    error:                   '#DC2626',
  };
  return map[status] ?? semanticColors.confidenceLow;
}

export function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.8) return semanticColors.confidenceHigh;
  if (confidence >= 0.5) return semanticColors.confidenceMed;
  return semanticColors.confidenceLow;
}

export function getConfidenceLabel(confidence: number): 'High' | 'Medium' | 'Low' {
  if (confidence >= 0.8) return 'High';
  if (confidence >= 0.5) return 'Medium';
  return 'Low';
}

export function confidenceGlowStyle(confidence: number): object {
  if (confidence >= 0.8) return {
    borderWidth: 1, borderColor: 'rgba(16,185,129,0.6)',
    shadowColor: '#10B981', shadowOpacity: 0.3, shadowRadius: 6, elevation: 4,
  };
  if (confidence >= 0.5) return {
    borderWidth: 1, borderColor: 'rgba(245,158,11,0.5)',
    shadowColor: '#F59E0B', shadowOpacity: 0.2, shadowRadius: 4, elevation: 3,
  };
  return { borderWidth: 1, borderColor: 'rgba(107,114,128,0.3)' };
}
