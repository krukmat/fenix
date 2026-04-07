// Governance screen tests — W5-T3
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseGovernanceSummary = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useGovernanceSummary: () => mockUseGovernanceSummary(),
}));

jest.mock('react-native-paper', () => ({
  useTheme: () => ({
    colors: {
      primary: '#E53935', surface: '#fff', onSurface: '#000',
      onSurfaceVariant: '#666', background: '#fff', error: '#B00020',
    },
  }),
}));

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const summaryFull = {
  recentUsage: [
    { id: 'u-1', metric_name: 'tokens', value: 1500, recorded_at: '2026-04-07T10:00:00Z', run_id: 'run-1' },
    { id: 'u-2', metric_name: 'cost_euros', value: 0.05, recorded_at: '2026-04-07T10:00:00Z', run_id: 'run-1' },
  ],
  quotaStates: [
    {
      policyId: 'pol-1',
      policyType: 'token_budget',
      metricName: 'tokens',
      limitValue: 100000,
      resetPeriod: 'daily',
      enforcementMode: 'soft',
      currentValue: 1500,
      periodStart: '2026-04-07T00:00:00Z',
      periodEnd: '2026-04-07T23:59:59Z',
      lastEventAt: '2026-04-07T10:00:00Z',
      statePresent: true,
    },
  ],
};

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Governance screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseGovernanceSummary.mockReturnValue({ data: summaryFull, isLoading: false, error: null });
  });

  it('renders the governance screen', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-screen')).toBeTruthy();
  });

  it('renders recent usage section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-recent-usage')).toBeTruthy();
  });

  it('renders quota states section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-quota-states')).toBeTruthy();
  });

  it('renders usage event items', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-usage-item-0')).toBeTruthy();
    expect(screen.getByTestId('governance-usage-item-1')).toBeTruthy();
  });

  it('renders quota state items', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-quota-item-0')).toBeTruthy();
  });

  it('shows empty quota message when no active quota policies', () => {
    mockUseGovernanceSummary.mockReturnValue({
      data: { recentUsage: [], quotaStates: [] },
      isLoading: false, error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-no-quota')).toBeTruthy();
  });

  it('remains functional when only recentUsage is available', () => {
    mockUseGovernanceSummary.mockReturnValue({
      data: { recentUsage: summaryFull.recentUsage, quotaStates: [] },
      isLoading: false, error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-recent-usage')).toBeTruthy();
    expect(screen.getByTestId('governance-no-quota')).toBeTruthy();
  });

  it('remains functional when only quotaStates is available', () => {
    mockUseGovernanceSummary.mockReturnValue({
      data: { recentUsage: [], quotaStates: summaryFull.quotaStates },
      isLoading: false, error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-quota-item-0')).toBeTruthy();
  });

  it('shows loading state', () => {
    mockUseGovernanceSummary.mockReturnValue({ data: undefined, isLoading: true, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-loading')).toBeTruthy();
  });

  it('shows error state', () => {
    mockUseGovernanceSummary.mockReturnValue({ data: null, isLoading: false, error: new Error('Failed') });
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-error')).toBeTruthy();
  });
});
