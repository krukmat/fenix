// ui-redesign-command-center: type scale tokens
import { Platform } from 'react-native';

const monoFont = Platform.OS === 'android' ? 'monospace' : 'Courier New';

export const typography = {
  headingLG: { fontFamily: 'Roboto', fontSize: 22, fontWeight: '700' as const, letterSpacing: -0.3 },
  headingMD: { fontFamily: 'Roboto', fontSize: 18, fontWeight: '600' as const },
  eyebrow:   { fontFamily: 'Roboto', fontSize: 11, fontWeight: '700' as const, letterSpacing: 1.2, textTransform: 'uppercase' as const },
  labelMD:   { fontFamily: 'Roboto', fontSize: 11, fontWeight: '600' as const, letterSpacing: 0.3 },
  mono:      { fontFamily: monoFont, fontSize: 12, fontWeight: '400' as const },
  monoLG:    { fontFamily: monoFont, fontSize: 14, fontWeight: '700' as const },
  monoSM:    { fontFamily: monoFont, fontSize: 11, fontWeight: '400' as const },
} as const;
