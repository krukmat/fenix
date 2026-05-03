// Governance screen tests — W5-T3
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

const mockRouterPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockRouterPush, replace: jest.fn() }),
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

// Wave 1 — mock UsageDetailCard so the screen test doesn't depend on Card.Content internals
jest.mock('../../../../src/components/governance/UsageDetailCard', () => ({
  UsageDetailCard: ({ testIDPrefix }: { testIDPrefix: string }) => {
    const React = require('react');
    const { View } = require('react-native');
    return React.createElement(View, { testID: `${testIDPrefix}-card` });
  },
}));

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const summaryFull = {
  recentUsage: [
    {
      id: 'u-1', workspaceId: 'ws-1', actorType: 'agent',
      toolName: 'send_email', modelName: 'claude-sonnet-4-6',
      estimatedCost: 0.00123, latencyMs: 842, createdAt: '2026-04-07T10:00:00Z', runId: 'run-1',
      inputUnits: 120, outputUnits: 48,
    },
    {
      id: 'u-2', workspaceId: 'ws-1', actorType: 'user',
      toolName: 'create_task', modelName: undefined,
      estimatedCost: 0.0005, latencyMs: 210, createdAt: '2026-04-07T10:05:00Z',
      inputUnits: 33, outputUnits: 7,
    },
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
    // Wave 1: testID now includes -card suffix from UsageDetailCard (testIDPrefix-card)
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-usage-item-0-card')).toBeTruthy();
    expect(screen.getByTestId('governance-usage-item-1-card')).toBeTruthy();
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

  // Wave 1 — governance_mobile_enhancement_plan: new assertions

  it('renders rich UsageDetailCard for each usage event', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    // Each UsageDetailCard has testIDPrefix governance-usage-item-{i}, so card testID is governance-usage-item-{i}-card
    expect(screen.getByTestId('governance-usage-item-0-card')).toBeTruthy();
    expect(screen.getByTestId('governance-usage-item-1-card')).toBeTruthy();
  });

  it('renders "View All" touch target for usage section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-view-all-usage')).toBeTruthy();
  });

  it('navigates to usage screen when "View All" is pressed', () => {
    const { fireEvent } = require('@testing-library/react-native');
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('governance-view-all-usage'));
    expect(mockRouterPush).toHaveBeenCalledWith('/governance/usage');
  });

  it('renders "Audit Trail" navigation link', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-audit-trail-link')).toBeTruthy();
  });

  it('navigates to audit screen when audit trail link is pressed', () => {
    const { fireEvent } = require('@testing-library/react-native');
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('governance-audit-trail-link'));
    expect(mockRouterPush).toHaveBeenCalledWith('/governance/audit');
  });

  it('renders "Workflows" navigation link', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('governance-workflows-link')).toBeTruthy();
  });

  it('navigates to workflows screen when workflows link is pressed', () => {
    const { fireEvent } = require('@testing-library/react-native');
    const { default: Screen } = require('../../../../app/(tabs)/governance/index');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('governance-workflows-link'));
    expect(mockRouterPush).toHaveBeenCalledWith('/(tabs)/workflows');
  });
});
