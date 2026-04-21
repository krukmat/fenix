// src/theme/types.ts
// Task 4.3.td — shared theme type, replaces per-file duplicates

export interface ThemeColors {
  background: string;
  surface: string;
  surfaceVariant: string;
  primary: string;
  onPrimary: string;
  onSurface: string;
  onSurfaceVariant: string;
  error: string;
  outline: string;
  success?: string;
  warning?: string;
  info?: string;
  surfaceContainerHigh?: string;
}
