// Sales deal detail screen tests — W4-T2
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'deal-deal-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseDeal = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useDeal: () => mockUseDeal(),
}));

const mockMutate = jest.fn();
const mockUseTriggerDealRiskAgent = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerDealRiskAgent: () => mockUseTriggerDealRiskAgent(),
}));

jest.mock('react-native-paper', () => {
  const mockReact = require('react');
  const { TouchableOpacity, Text } = require('react-native');
  return {
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
    Button: ({
      children,
      onPress,
      testID,
      disabled,
    }: {
      children: unknown;
      onPress: () => void;
      testID: string;
      disabled?: boolean;
    }) =>
      mockReact.createElement(
        TouchableOpacity,
        { testID, onPress, accessibilityState: { disabled: !!disabled } },
        mockReact.createElement(Text, null, children),
      ),
  };
});

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  return {
    CRMDetailHeader: ({ title, testIDPrefix }: { title: string; testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-header` },
        React.createElement(Text, null, title)),
  };
});

jest.mock('../../../../src/components/agents/AgentActivitySection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    AgentActivitySection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-agent-activity` }),
  };
});

jest.mock('../../../../src/components/signals/EntitySignalsSection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    EntitySignalsSection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-signals` }),
  };
});

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const dealPayload = {
  deal: {
    id: 'deal-1',
    title: 'Big Deal',
    status: 'open',
    amount: 50000,
    stage: 'Proposal',
    closeDate: '2026-06-30',
    accountId: 'acc-1',
  },
  account: { id: 'acc-1', name: 'Acme Corp' },
  active_signal_count: 1,
};

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Sales deal detail screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseTriggerDealRiskAgent.mockReturnValue({
      mutate: mockMutate,
      isPending: false,
    });
    mockUseDeal.mockReturnValue({
      data: dealPayload,
      isLoading: false,
      error: null,
    });
  });

  it('renders the deal detail screen', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-detail-screen')).toBeTruthy();
  });

  it('renders deal header', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-detail-header')).toBeTruthy();
  });

  it('shows deal amount', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-amount')).toBeTruthy();
  });

  it('shows deal stage', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-stage')).toBeTruthy();
  });

  it('renders agent activity section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-detail-agent-activity')).toBeTruthy();
  });

  it('renders Sales Brief button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-brief-button')).toBeTruthy();
  });

  it('renders Deal Risk trigger button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    const button = screen.getByTestId('deal-risk-trigger-button');
    expect(button).toBeTruthy();
    expect(button.props.accessibilityState?.disabled).toBe(false);
  });

  it('navigates to deal brief when button pressed', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('sales-deal-brief-button'));
    expect(mockPush).toHaveBeenCalledWith(expect.objectContaining({ pathname: '/sales/deal-deal-1/brief' }));
  });

  it('does NOT render an edit button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('sales-deal-edit-button')).toBeNull();
  });

  it('triggers deal risk agent on press', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));

    fireEvent.press(screen.getByTestId('deal-risk-trigger-button'));

    expect(mockMutate).toHaveBeenCalledWith(
      { dealId: 'deal-1', language: 'es' },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it('disables Deal Risk button while pending', () => {
    mockUseTriggerDealRiskAgent.mockReturnValue({
      mutate: mockMutate,
      isPending: true,
    });
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));

    const button = screen.getByTestId('deal-risk-trigger-button');
    expect(button.props.accessibilityState?.disabled).toBe(true);
    expect(screen.getByText('Running...')).toBeTruthy();
  });

  it('navigates to run detail on Deal Risk success', () => {
    mockMutate.mockImplementation((_input, options) => {
      options?.onSuccess?.({ runId: 'run-123', status: 'queued', agent: 'deal-risk' });
    });
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));

    fireEvent.press(screen.getByTestId('deal-risk-trigger-button'));

    expect(mockPush).toHaveBeenCalledWith('/activity/run-123');
  });

  it('shows loading state', () => {
    mockUseDeal.mockReturnValue({ data: undefined, isLoading: true, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-detail-loading')).toBeTruthy();
  });

  it('shows error state', () => {
    mockUseDeal.mockReturnValue({ data: null, isLoading: false, error: new Error('Not found') });
    const { default: Screen } = require('../../../../app/(tabs)/sales/deal-[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-deal-detail-error')).toBeTruthy();
  });
});
