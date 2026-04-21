// ui-redesign-command-center: switch to dark base theme
import { MD3DarkTheme } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import { brandColors } from './colors';

export const fenixTheme: MD3Theme = {
  ...MD3DarkTheme,
  colors: {
    ...MD3DarkTheme.colors,
    ...brandColors,
  },
};

export { brandColors };
