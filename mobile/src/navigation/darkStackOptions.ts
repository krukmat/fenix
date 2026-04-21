// ui-redesign-command-center: shared dark header options for nested stacks
export const darkStackScreenOptions = {
  headerShown: false,
  animation: 'slide_from_right' as const,
  headerShadowVisible: false,
  headerStyle: { backgroundColor: '#111620' },
  headerTintColor: '#F0F4FF',
  headerTitleAlign: 'left' as const,
  headerTitleStyle: {
    color: '#F0F4FF',
    fontSize: 18,
    fontWeight: '700' as const,
    letterSpacing: -0.3,
  },
  contentStyle: { backgroundColor: '#0A0D12' },
} as const;
