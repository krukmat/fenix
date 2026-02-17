// Task 4.2 — FR-300: UI Store (theme mode, drawer state)

import { create } from 'zustand';

type ThemeMode = 'light' | 'dark' | 'system';

interface UIState {
  themeMode: ThemeMode;
  isDrawerOpen: boolean;
  setThemeMode: (mode: ThemeMode) => void;
  setDrawerOpen: (open: boolean) => void;
  toggleDrawer: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  themeMode: 'light',
  isDrawerOpen: false,

  setThemeMode: (mode: ThemeMode) => {
    set({ themeMode: mode });
  },

  setDrawerOpen: (open: boolean) => {
    set({ isDrawerOpen: open });
  },

  toggleDrawer: () => {
    set((state) => ({ isDrawerOpen: !state.isDrawerOpen }));
  },
}));
