// Task 4.2 — FR-300: Paleta de marca FenixCRM (MD3 color scheme)

export const brandColors = {
  primary: '#1565C0',
  onPrimary: '#FFFFFF',
  primaryContainer: '#D6E4FF',
  onPrimaryContainer: '#001A41',
  secondary: '#0288D1',
  onSecondary: '#FFFFFF',
  secondaryContainer: '#CDE5FF',
  onSecondaryContainer: '#001D31',
  error: '#B3261E',
  errorContainer: '#F9DEDC',
  onError: '#FFFFFF',
  onErrorContainer: '#410E0B',
  background: '#FEFBFF',
  onBackground: '#1C1B1F',
  surface: '#FEFBFF',
  onSurface: '#1C1B1F',
  surfaceVariant: '#E7E0EC',
  onSurfaceVariant: '#49454F',
  outline: '#79747E',
  outlineVariant: '#CAC4D0',
  inverseSurface: '#313033',
  inverseOnSurface: '#F4EFF4',
  inversePrimary: '#A8C7FF',
} as const;

export type BrandColors = typeof brandColors;
