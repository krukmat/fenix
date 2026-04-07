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

    expect(visibleNames).toContain('inbox/index');
    expect(visibleNames).toContain('support/index');
    expect(visibleNames).toContain('sales/index');
    expect(visibleNames).toContain('activity/index');
    expect(visibleNames).toContain('governance/index');
  });

  it('keeps legacy redirect shims registered but hidden (href: null)', () => {
    render(React.createElement(TabsLayout));

    type ScreenCall = [{ name: string; options?: { href?: null | string } }];
    const tabsScreen = getTabsScreenMock();
    const hiddenNames = (tabsScreen.mock.calls as ScreenCall[])
      .map(([p]) => p)
      .filter((p) => p.options?.href === null)
      .map((p) => p.name);

    expect(hiddenNames).toContain('home/index');
    expect(hiddenNames).toContain('accounts/index');
    expect(hiddenNames).toContain('cases/index');
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
