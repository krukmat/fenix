// ui-redesign-command-center: shared dark header options for nested stacks
import { brandColors } from '../theme/colors';
import { typography } from '../theme/typography';

export const darkStackScreenOptions = {
  headerShown: false,
  animation: 'slide_from_right' as const,
  headerShadowVisible: false,
  headerStyle: { backgroundColor: brandColors.surface },
  headerTintColor: brandColors.onBackground,
  headerTitleAlign: 'left' as const,
  headerTitleStyle: {
    color: brandColors.onBackground,
    ...typography.headingLG,
    fontSize: typography.headingMD.fontSize,
  },
  contentStyle: { backgroundColor: brandColors.background },
} as const;
