// ui-redesign-command-center: dark Command Center palette
export const brandColors = {
  primary:              '#3B82F6',
  onPrimary:            '#FFFFFF',
  primaryContainer:     '#1E3A5F',
  onPrimaryContainer:   '#93C5FD',
  secondary:            '#F59E0B',
  onSecondary:          '#0A0D12',
  secondaryContainer:   '#3D2C00',
  onSecondaryContainer: '#FDE68A',
  error:                '#EF4444',
  onError:              '#FFFFFF',
  errorContainer:       '#3B0F0F',
  onErrorContainer:     '#FCA5A5',
  background:           '#0A0D12',
  onBackground:         '#F0F4FF',
  surface:              '#111620',
  onSurface:            '#E2E8F0',
  surfaceVariant:       '#1A2030',
  onSurfaceVariant:     '#8899AA',
  outline:              '#2E3A50',
  outlineVariant:       '#1E2B3E',
  inverseSurface:       '#E2E8F0',
  inverseOnSurface:     '#0A0D12',
  inversePrimary:       '#3B82F6',
} as const;

export type BrandColors = typeof brandColors;

export const semanticColors = {
  success:              '#10B981',
  successContainer:     '#052E1C',
  onSuccessContainer:   '#6EE7B7',
  warning:              '#F59E0B',
  warningContainer:     '#3D2C00',
  onWarningContainer:   '#FDE68A',
  info:                 '#60A5FA',
  confidenceHigh:       '#10B981',
  confidenceMed:        '#F59E0B',
  confidenceLow:        '#6B7280',
} as const;
