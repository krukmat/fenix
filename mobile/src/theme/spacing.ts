// ui-redesign-command-center: spacing, radius, elevation tokens
export const spacing = { xs: 4, sm: 8, md: 12, base: 16, lg: 20, xl: 24, xxl: 32 } as const;

export const radius = { xs: 4, sm: 6, md: 10, lg: 14, full: 999 } as const;

export const elevation = {
  card:   { borderWidth: 1, borderColor: '#1E2B3E', elevation: 0 },
  raised: { shadowColor: '#3B82F6', shadowOpacity: 0.08, shadowOffset: { width: 0, height: 2 }, shadowRadius: 8, elevation: 3 },
  tabBar: { shadowColor: '#000000', shadowOpacity: 0.4, shadowOffset: { width: 0, height: -2 }, shadowRadius: 12, elevation: 12 },
} as const;
