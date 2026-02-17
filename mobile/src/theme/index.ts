// Task 4.2 — FR-300: Tema MD3 con colores FenixCRM

import { MD3LightTheme } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import { brandColors } from './colors';

export const fenixTheme: MD3Theme = {
  ...MD3LightTheme,
  colors: {
    ...MD3LightTheme.colors,
    ...brandColors,
  },
};

export { brandColors };
