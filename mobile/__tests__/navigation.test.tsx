/**
 * Task 4.2 — FR-300: Tests del auth guard de navegación
 * W2-T1 (mobile_wedge_harmonization_plan): wedge bottom-tab structure tests
 *
 * Tests:
 * 1. redirects to login when unauthenticated
 * 2. renders slot when authenticated
 * 3. 5 wedge tabs are registered as visible screens
 * 4. legacy screens remain registered but hidden from tab bar
 *
 * Nota: el auth guard real vive en app/(tabs)/_layout.tsx (Redirect cuando !isAuthenticated)
 * y en app/_layout.tsx (Redirect antes de providers). Aquí testeamos las condiciones
 * de estado del store que disparan esas redirecciones.
 */

import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { useAuthStore } from '../src/stores/authStore';

// ─── W2-T1: Wedge tab layout structure ───────────────────────────────────────

// mockTabsScreen is populated via the factory — accessed through the module mock registry.
jest.mock('expo-router', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  const screenFn = jest.fn(() => null);

  const Tabs = ({ children }: { children: React.ReactNode }) =>
    React.createElement(View, { testID: 'tabs-root' }, children);
  Tabs.Screen = screenFn;

  return {
    __esModule: true,
    Tabs,
    Redirect: ({ href }: { href: string }) =>
      React.createElement(View, { testID: 'redirect' },
        React.createElement(Text, null, String(href))
      ),
    useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
    useLocalSearchParams: () => ({}),
    Stack: { Screen: jest.fn() },
    _tabsScreenMock: screenFn,
  };
});

jest.mock('../src/hooks/useWedge', () => ({
  useInbox: () => ({ data: null, isLoading: false }),
}));

import TabsLayout from '../app/(tabs)/_layout';
import * as ExpoRouterMock from 'expo-router';

// Retrieve the captured Tabs.Screen mock from the factory
const getTabsScreenMock = () =>
  (ExpoRouterMock as unknown as { _tabsScreenMock: jest.Mock })._tabsScreenMock;

describe('W2-T1: Wedge bottom-tab layout', () => {
  beforeEach(() => {
    getTabsScreenMock().mockClear();
    useAuthStore.setState({
      token: 'jwt-test',
      userId: 'user-1',
      workspaceId: 'ws-1',
      isAuthenticated: true,
      isLoading: false,
    });
  });

  it('renders the tabs root container when authenticated', () => {
    render(React.createElement(TabsLayout));
    expect(screen.getByTestId('tabs-root')).toBeTruthy();
  });

  it('registers the 5 visible wedge tabs', () => {
    render(React.createElement(TabsLayout));

    type ScreenCall = [{ name: string; options?: { href?: null | string } }];
    const tabsScreen = getTabsScreenMock();
    const visibleNames = (tabsScreen.mock.calls as ScreenCall[])
      .map(([p]) => p)
      .filter((p) => p.options?.href !== null)
      .map((p) => p.name);

    expect(visibleNames).toHaveLength(5);
    expect(visibleNames).toContain('inbox/index');
    expect(visibleNames).toContain('support');
    expect(visibleNames).toContain('sales');
    expect(visibleNames).toContain('activity');
    expect(visibleNames).toContain('governance');
  });

  it('assigns a real tab icon and stable testID to each visible wedge tab', () => {
    render(React.createElement(TabsLayout));

    type ScreenCall = [{ name: string; options?: { href?: null | string; tabBarButtonTestID?: string; tabBarIcon?: unknown } }];
    const tabsScreen = getTabsScreenMock();
    const visibleTabs = (tabsScreen.mock.calls as ScreenCall[])
      .map(([p]) => p)
      .filter((p) => p.options?.href !== null);

    expect(visibleTabs).toHaveLength(5);
    expect(visibleTabs).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: 'inbox/index', options: expect.objectContaining({ tabBarButtonTestID: 'tab-inbox', tabBarIcon: expect.any(Function) }) }),
        expect.objectContaining({ name: 'support', options: expect.objectContaining({ tabBarButtonTestID: 'tab-support', tabBarIcon: expect.any(Function) }) }),
        expect.objectContaining({ name: 'sales', options: expect.objectContaining({ tabBarButtonTestID: 'tab-sales', tabBarIcon: expect.any(Function) }) }),
        expect.objectContaining({ name: 'activity', options: expect.objectContaining({ tabBarButtonTestID: 'tab-activity', tabBarIcon: expect.any(Function) }) }),
        expect.objectContaining({ name: 'governance', options: expect.objectContaining({ tabBarButtonTestID: 'tab-governance', tabBarIcon: expect.any(Function) }) }),
      ])
    );
  });

  it('keeps legacy redirect shims registered but hidden (href: null)', () => {
    render(React.createElement(TabsLayout));

    type ScreenCall = [{ name: string; options?: { href?: null | string } }];
    const tabsScreen = getTabsScreenMock();
    const hiddenNames = (tabsScreen.mock.calls as ScreenCall[])
      .map(([p]) => p)
      .filter((p) => p.options?.href === null)
      .map((p) => p.name);

    expect(hiddenNames).toContain('home');
    expect(hiddenNames).toContain('accounts');
    expect(hiddenNames).toContain('deals');
    expect(hiddenNames).toContain('cases');
    expect(hiddenNames).toContain('contacts');
    expect(hiddenNames).toContain('workflows');
    expect(hiddenNames).toContain('crm');
    expect(hiddenNames).not.toContain('agents/index');
    expect(hiddenNames).not.toContain('crm/index');
  });

  it('redirects to login when unauthenticated', () => {
    useAuthStore.setState({
      token: null,
      userId: null,
      workspaceId: null,
      isAuthenticated: false,
      isLoading: false,
    });
    render(React.createElement(TabsLayout));
    expect(screen.getByTestId('redirect')).toBeTruthy();
  });
});

// ─── Tab bar overflow guard (mobile_tab_bar_overflow_fix_plan) ───────────────
// Asserts that every feature folder containing dynamic routes has its own
// _layout.tsx (Stack). Without it, expo-router leaks nested routes into the
// parent Tabs navigator as ghost tabs, breaking the footer.

describe('Tab bar overflow guard — feature folder layouts', () => {
  const fs = require('fs');
  const path = require('path');
  const tabsBase = path.join(__dirname, '..', 'app', '(tabs)');

  const foldersRequiringLayout = ['sales', 'support', 'accounts', 'cases', 'deals', 'contacts', 'workflows', 'crm'];

  it.each(foldersRequiringLayout)(
    '%s/ has a _layout.tsx to contain its dynamic routes',
    (folder) => {
      const layoutPath = path.join(tabsBase, folder, '_layout.tsx');
      expect(fs.existsSync(layoutPath)).toBe(true);
    },
  );
});

// ─── Auth guard — condiciones de navegación ───────────────────────────────────

describe('Auth guard — condiciones de navegación', () => {
  beforeEach(() => {
    useAuthStore.setState({
      token: null,
      userId: null,
      workspaceId: null,
      isAuthenticated: false,
      isLoading: false,
    });
  });

  it('isAuthenticated es false cuando no hay token almacenado', () => {
    const { isAuthenticated, token } = useAuthStore.getState();
    expect(isAuthenticated).toBe(false);
    expect(token).toBeNull();
    // El auth guard en (tabs)/_layout.tsx dispara <Redirect href="/(auth)/login" />
  });

  it('isAuthenticated es true después de login exitoso', async () => {
    await useAuthStore.getState().login({
      token: 'jwt-abc',
      userId: 'user-1',
      workspaceId: 'ws-1',
    });

    const { isAuthenticated, token, workspaceId } = useAuthStore.getState();
    expect(isAuthenticated).toBe(true);
    expect(token).toBe('jwt-abc');
    expect(workspaceId).toBe('ws-1');
    // El auth guard en (tabs)/_layout.tsx NO dispara Redirect → renderiza los Tabs
  });

  it('isLoading es true durante loadStoredToken y false al terminar', async () => {
    const loadPromise = useAuthStore.getState().loadStoredToken();
    expect(useAuthStore.getState().isLoading).toBe(true);
    await loadPromise;
    expect(useAuthStore.getState().isLoading).toBe(false);
  });

  it('isAuthenticated vuelve a false después de logout', async () => {
    await useAuthStore.getState().login({
      token: 'jwt-xyz',
      userId: 'user-2',
      workspaceId: 'ws-2',
    });
    expect(useAuthStore.getState().isAuthenticated).toBe(true);

    await useAuthStore.getState().logout();

    const { isAuthenticated, token } = useAuthStore.getState();
    expect(isAuthenticated).toBe(false);
    expect(token).toBeNull();
  });
});
