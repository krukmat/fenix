// Drawer navigation — 5-item layout, approvals badge, CRM submenu, auth guard
// FR-300 (navigation), FR-071 (approvals badge), UC-A5/A7 (Home as entry point)


import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import Layout from '../../app/(tabs)/_layout';

// ─── Mocks ───────────────────────────────────────────────────────────────────

const mockUsePendingApprovals = jest.fn();
const mockUseAuthStore = jest.fn();
const mockNavigate = jest.fn();
const mockReplace = jest.fn();

jest.mock('../../src/hooks/useAgentSpec', () => ({
  usePendingApprovals: () => mockUsePendingApprovals(),
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: (...args: unknown[]) => mockUseAuthStore(...args),
}));

jest.mock('expo-router', () => ({
  Redirect: () => null,
  useRouter: () => ({ replace: mockReplace }),
  Drawer: {
    Screen: () => null,
  },
}));

jest.mock('expo-router/drawer', () => {
  function DrawerMock({ drawerContent, children }: { drawerContent: (p: unknown) => unknown; children?: unknown }) {
    const { View } = require('react-native');
    const React = require('react');
    const fakeProps = { navigation: { navigate: mockNavigate, openDrawer: jest.fn() } };
    return React.createElement(View, null, drawerContent(fakeProps), children);
  }
  DrawerMock.Screen = () => null;
  return { Drawer: DrawerMock };
});

jest.mock('@react-navigation/drawer', () => ({
  DrawerContentScrollView: ({ children, testID }: { children: unknown; testID?: string }) => {
    const { View } = require('react-native');
    const React = require('react');
    return React.createElement(View, { testID }, children);
  },
}));

// ─── Helpers ─────────────────────────────────────────────────────────────────

function setupAuth(isAuthenticated = true) {
  mockUseAuthStore.mockImplementation((sel: unknown) => {
    if (typeof sel === 'function') {
      return (sel as (s: { isAuthenticated: boolean; userId: string; logout: () => Promise<void> }) => unknown)({
        isAuthenticated,
        userId: 'user-1',
        logout: jest.fn().mockResolvedValue(undefined),
      });
    }
    return { isAuthenticated, userId: 'user-1', logout: jest.fn() };
  });
}

function renderLayout() {
  return render(
    <PaperProvider>
      <Layout />
    </PaperProvider>
  );
}

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Drawer layout — 5-item structure', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setupAuth(true);
    mockUsePendingApprovals.mockReturnValue({ data: [] });
  });

  it('renders exactly the 5 top-level drawer items', () => {
    const { getByTestId } = renderLayout();
    expect(getByTestId('drawer-home-tab')).toBeTruthy();
    expect(getByTestId('drawer-crm-tab')).toBeTruthy();
    expect(getByTestId('drawer-copilot-tab')).toBeTruthy();
    expect(getByTestId('drawer-workflows-tab')).toBeTruthy();
    expect(getByTestId('drawer-activity-tab')).toBeTruthy();
  });

  it('does NOT render old individual entity tabs (Accounts, Contacts, Deals, Cases)', () => {
    const { queryByTestId } = renderLayout();
    expect(queryByTestId('drawer-accounts-tab')).toBeNull();
    expect(queryByTestId('drawer-contacts-tab')).toBeNull();
    expect(queryByTestId('drawer-deals-tab')).toBeNull();
    expect(queryByTestId('drawer-cases-tab')).toBeNull();
  });

  it('does NOT render old Agent Runs tab', () => {
    const { queryByTestId } = renderLayout();
    expect(queryByTestId('drawer-agents-tab')).toBeNull();
  });

  it('renders Activity Log tab (not Agent Runs)', () => {
    const { getByTestId } = renderLayout();
    expect(getByTestId('drawer-activity-tab').props.children).not.toBeNull();
  });
});

describe('Drawer layout — Home badge', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setupAuth(true);
  });

  it('shows badge on Home tab when there are pending approvals', () => {
    mockUsePendingApprovals.mockReturnValue({
      data: [{ id: 'a1' }, { id: 'a2' }, { id: 'a3' }],
    });
    const { getByTestId } = renderLayout();
    expect(getByTestId('drawer-home-tab-badge')).toBeTruthy();
    expect(within(getByTestId('drawer-home-tab-badge')).getByText('3')).toBeTruthy();
  });

  it('does NOT show badge when pending approvals count is 0', () => {
    mockUsePendingApprovals.mockReturnValue({ data: [] });
    const { queryByTestId } = renderLayout();
    expect(queryByTestId('drawer-home-tab-badge')).toBeNull();
  });

  it('caps badge display at 99+ when count exceeds 99', () => {
    mockUsePendingApprovals.mockReturnValue({
      data: Array.from({ length: 150 }, (_, i) => ({ id: `a${i}` })),
    });
    const { getByTestId } = renderLayout();
    expect(within(getByTestId('drawer-home-tab-badge')).getByText('99+')).toBeTruthy();
  });

  it('does NOT show badge when usePendingApprovals returns undefined data', () => {
    mockUsePendingApprovals.mockReturnValue({ data: undefined });
    const { queryByTestId } = renderLayout();
    expect(queryByTestId('drawer-home-tab-badge')).toBeNull();
  });
});

describe('Drawer layout — CRM submenu', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setupAuth(true);
    mockUsePendingApprovals.mockReturnValue({ data: [] });
  });

  it('CRM submenu is hidden before tapping CRM tab', () => {
    const { queryByTestId } = renderLayout();
    expect(queryByTestId('drawer-crm-submenu')).toBeNull();
  });

  it('CRM submenu expands after tapping CRM tab', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-crm-tab'));
    expect(getByTestId('drawer-crm-submenu')).toBeTruthy();
  });

  it('CRM submenu shows Accounts, Contacts, Deals, Cases sub-items', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-crm-tab'));
    expect(getByTestId('drawer-crm-accounts')).toBeTruthy();
    expect(getByTestId('drawer-crm-contacts')).toBeTruthy();
    expect(getByTestId('drawer-crm-deals')).toBeTruthy();
    expect(getByTestId('drawer-crm-cases')).toBeTruthy();
  });

  it('CRM submenu collapses again on second tap', () => {
    const { getByTestId, queryByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-crm-tab'));
    expect(getByTestId('drawer-crm-submenu')).toBeTruthy();
    fireEvent.press(getByTestId('drawer-crm-tab'));
    expect(queryByTestId('drawer-crm-submenu')).toBeNull();
  });

  it('navigates to crm/accounts when Accounts sub-item is pressed', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-crm-tab'));
    fireEvent.press(getByTestId('drawer-crm-accounts'));
    expect(mockNavigate).toHaveBeenCalledWith('crm/accounts/index');
  });

  it('navigates to crm/deals when Deals sub-item is pressed', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-crm-tab'));
    fireEvent.press(getByTestId('drawer-crm-deals'));
    expect(mockNavigate).toHaveBeenCalledWith('crm/deals/index');
  });
});

describe('Drawer layout — navigation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setupAuth(true);
    mockUsePendingApprovals.mockReturnValue({ data: [] });
  });

  it('navigates to home/index when Home is pressed', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-home-tab'));
    expect(mockNavigate).toHaveBeenCalledWith('home/index');
  });

  it('navigates to workflows/index when Workflows is pressed', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-workflows-tab'));
    expect(mockNavigate).toHaveBeenCalledWith('workflows/index');
  });

  it('navigates to activity when Activity Log is pressed', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-activity-tab'));
    expect(mockNavigate).toHaveBeenCalledWith('activity');
  });

  it('navigates to copilot/index when Copilot is pressed', () => {
    const { getByTestId } = renderLayout();
    fireEvent.press(getByTestId('drawer-copilot-tab'));
    expect(mockNavigate).toHaveBeenCalledWith('copilot/index');
  });
});

describe('Drawer layout — auth guard', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders Redirect when not authenticated', () => {
    setupAuth(false);
    mockUsePendingApprovals.mockReturnValue({ data: [] });
      // When unauthenticated, Layout returns <Redirect> which renders null in our mock
    const { queryByTestId } = render(
      <PaperProvider>
        <Layout />
      </PaperProvider>
    );
    expect(queryByTestId('drawer-content')).toBeNull();
  });
});