import { describe, it, expect, beforeEach } from '@jest/globals';
import { useUIStore } from '../../src/stores/uiStore';

describe('uiStore', () => {
  beforeEach(() => {
    useUIStore.setState({
      themeMode: 'light',
      isDrawerOpen: false,
    });
  });

  it('should have initial state', () => {
    const state = useUIStore.getState();
    expect(state.themeMode).toBe('light');
    expect(state.isDrawerOpen).toBe(false);
  });

  it('setThemeMode should update theme mode', () => {
    useUIStore.getState().setThemeMode('dark');
    expect(useUIStore.getState().themeMode).toBe('dark');

    useUIStore.getState().setThemeMode('system');
    expect(useUIStore.getState().themeMode).toBe('system');
  });

  it('setDrawerOpen should update drawer state', () => {
    useUIStore.getState().setDrawerOpen(true);
    expect(useUIStore.getState().isDrawerOpen).toBe(true);

    useUIStore.getState().setDrawerOpen(false);
    expect(useUIStore.getState().isDrawerOpen).toBe(false);
  });

  it('toggleDrawer should invert drawer state', () => {
    expect(useUIStore.getState().isDrawerOpen).toBe(false);

    useUIStore.getState().toggleDrawer();
    expect(useUIStore.getState().isDrawerOpen).toBe(true);

    useUIStore.getState().toggleDrawer();
    expect(useUIStore.getState().isDrawerOpen).toBe(false);
  });
});
