// Sales index screen tests — W4-T1
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';
import SalesScreen from '../../../../app/(tabs)/sales/index';

// ─── Mocks ────────────────────────────────────────────────────────────────────

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseAccounts = jest.fn();
const mockUseDeals = jest.fn();
const mockUseLeads = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useAccounts: () => mockUseAccounts(),
  useDeals: () => mockUseDeals(),
  useLeads: () => mockUseLeads(),
}));

jest.mock('react-native-paper', () => ({
  useTheme: () => ({
    colors: {
      primary: '#E53935',
      surface: '#fff',
      onSurface: '#000',
      onSurfaceVariant: '#666',
      background: '#fff',
      error: '#B00020',
    },
  }),
}));

jest.mock('../../../../src/components/signals/SignalCountBadge', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    SignalCountBadge: ({ testID }: { testID: string }) =>
      React.createElement(View, { testID }),
  };
});

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const makePage = (items: object[]) => ({ pages: [{ data: items }] });

const accountA = { id: 'acc-1', name: 'Acme Corp', industry: 'Tech', active_signal_count: 2 };
const accountB = { id: 'acc-2', name: 'Globex', industry: 'Finance', active_signal_count: 0 };
const dealA = { id: 'deal-1', title: 'Big Deal', status: 'open' as const, amount: 50000, accountName: 'Acme Corp' };
const dealB = { id: 'deal-2', title: 'Small Deal', status: 'won' as const, amount: 10000 };
const leadA = { id: 'lead-1', source: 'website', status: 'new', score: 88, metadata: JSON.stringify({ name: 'Jane Roe', email: 'jane@example.com' }) };
const leadB = { id: 'lead-2', source: 'event', status: 'qualified', score: 70 };

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Sales index screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAccounts.mockReturnValue({
      data: makePage([accountA, accountB]),
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
    });
    mockUseDeals.mockReturnValue({
      data: makePage([dealA, dealB]),
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
    });
    mockUseLeads.mockReturnValue({
      data: makePage([leadA, leadB]),
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
    });
  });

  it('renders the sales screen container', () => {
    render(React.createElement(SalesScreen));
    expect(screen.getByTestId('sales-screen')).toBeTruthy();
  });

  it('renders Accounts, Deals and Leads tabs', () => {
    render(React.createElement(SalesScreen));
    expect(screen.getByTestId('sales-tab-accounts')).toBeTruthy();
    expect(screen.getByTestId('sales-tab-deals')).toBeTruthy();
    expect(screen.getByTestId('sales-tab-leads')).toBeTruthy();
  });

  it('shows accounts list by default', () => {
    render(React.createElement(SalesScreen));
    expect(screen.getByTestId('sales-account-item-0')).toBeTruthy();
    expect(screen.getByTestId('sales-account-item-1')).toBeTruthy();
  });

  it('navigates to /sales/[id] when an account is pressed', () => {
    render(React.createElement(SalesScreen));
    fireEvent.press(screen.getByTestId('sales-account-item-0'));
    expect(mockPush).toHaveBeenCalledWith('/sales/acc-1');
  });

  it('switches to deals list when Deals tab is pressed', () => {
    render(React.createElement(SalesScreen));
    fireEvent.press(screen.getByTestId('sales-tab-deals'));
    expect(screen.getByTestId('sales-deal-item-0')).toBeTruthy();
    expect(screen.getByTestId('sales-deal-item-1')).toBeTruthy();
  });

  it('navigates to /sales/deals/[id] when a deal is pressed', () => {
    render(React.createElement(SalesScreen));
    fireEvent.press(screen.getByTestId('sales-tab-deals'));
    fireEvent.press(screen.getByTestId('sales-deal-item-0'));
    expect(mockPush).toHaveBeenCalledWith('/sales/deals/deal-1');
  });

  it('switches to leads list when Leads tab is pressed', () => {
    render(React.createElement(SalesScreen));
    fireEvent.press(screen.getByTestId('sales-tab-leads'));
    expect(screen.getByTestId('sales-lead-item-0')).toBeTruthy();
    expect(screen.getByTestId('sales-lead-item-1')).toBeTruthy();
  });

  it('navigates to /sales/leads/[id] when a lead is pressed', () => {
    render(React.createElement(SalesScreen));
    fireEvent.press(screen.getByTestId('sales-tab-leads'));
    fireEvent.press(screen.getByTestId('sales-lead-item-0'));
    expect(mockPush).toHaveBeenCalledWith('/sales/leads/lead-1');
  });

  it('shows loading indicator for accounts while fetching', () => {
    mockUseAccounts.mockReturnValue({
      data: undefined,
      isLoading: true,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
    });
    render(React.createElement(SalesScreen));
    expect(screen.getByTestId('sales-accounts-loading')).toBeTruthy();
  });

  it('shows empty state when no accounts', () => {
    mockUseAccounts.mockReturnValue({
      data: makePage([]),
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
    });
    render(React.createElement(SalesScreen));
    expect(screen.getByTestId('sales-accounts-empty')).toBeTruthy();
  });

  it('does NOT render a create FAB', () => {
    render(React.createElement(SalesScreen));
    expect(screen.queryByTestId('create-account-fab')).toBeNull();
    expect(screen.queryByTestId('create-deal-fab')).toBeNull();
  });
});
