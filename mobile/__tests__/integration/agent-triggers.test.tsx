import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';

const mockPush = jest.fn();
const mockUseLocalSearchParams = jest.fn();

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => mockUseLocalSearchParams(),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseCase = jest.fn();
const mockUseLead = jest.fn();
jest.mock('../../src/hooks/useCRM', () => ({
  useCase: () => mockUseCase(),
  useLead: () => mockUseLead(),
}));

const mockUseTriggerKBAgent = jest.fn();
const mockUseTriggerProspectingAgent = jest.fn();
const mockUseTriggerInsightsAgent = jest.fn();
jest.mock('../../src/hooks/useWedge', () => ({
  useTriggerSupportAgent: () => ({ mutate: jest.fn(), isPending: false }),
  useTriggerKBAgent: () => mockUseTriggerKBAgent(),
  useTriggerProspectingAgent: () => mockUseTriggerProspectingAgent(),
  useTriggerInsightsAgent: () => mockUseTriggerInsightsAgent(),
  useAgentRuns: () => ({ data: null }),
}));

jest.mock('../../src/components/crm', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CRMDetailHeader: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-header` }),
  };
});

jest.mock('../../src/components/agents/AgentActivitySection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    AgentActivitySection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('../../src/components/signals/EntitySignalsSection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    EntitySignalsSection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('../../src/components/signals/SignalCountBadge', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    SignalCountBadge: ({ testID }: { testID: string }) => React.createElement(View, { testID }),
  };
});

jest.mock('react-native-paper', () => {
  const React = require('react');
  const { TouchableOpacity, Text, TextInput: RNTextInput, View } = require('react-native');
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
      testID,
      onPress,
      children,
      disabled,
    }: {
      testID: string;
      onPress: () => void;
      children: React.ReactNode;
      disabled?: boolean;
    }) =>
      React.createElement(
        TouchableOpacity,
        { testID, onPress, accessibilityState: { disabled: !!disabled } },
        React.createElement(Text, null, children),
      ),
    TextInput: ({
      testID,
      value,
      onChangeText,
    }: {
      testID: string;
      value?: string;
      onChangeText: (value: string) => void;
    }) => React.createElement(RNTextInput, { testID, value, onChangeText }),
    ActivityIndicator: ({ testID }: { testID?: string }) => React.createElement(View, { testID }),
  };
});

describe('Agent trigger integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseLocalSearchParams.mockReturnValue({ id: 'case-1' });
    mockUseCase.mockReturnValue({
      data: { case: { id: 'case-1', subject: 'Resolved Case', status: 'resolved', priority: 'medium' }, active_signal_count: 0 },
      isLoading: false,
      error: null,
    });
    mockUseLead.mockReturnValue({
      data: { id: 'lead-1', source: 'website', status: 'qualified', metadata: '{"name":"Lead One"}' },
      isLoading: false,
      error: null,
    });
    mockUseTriggerKBAgent.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseTriggerProspectingAgent.mockReturnValue({
      mutateAsync: jest.fn().mockResolvedValue({ runId: 'run-pro-1', status: 'queued', agent: 'prospecting' }),
      isPending: false,
    });
    mockUseTriggerInsightsAgent.mockReturnValue({
      mutateAsync: jest.fn().mockResolvedValue({ runId: 'run-ins-1', status: 'queued', agent: 'insights' }),
      isPending: false,
    });
  });

  it('shows KB trigger only for resolved cases', () => {
    const { default: Screen } = require('../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('kb-trigger-button')).toBeTruthy();

    mockUseCase.mockReturnValue({
      data: { case: { id: 'case-1', subject: 'Open Case', status: 'open', priority: 'medium' }, active_signal_count: 0 },
      isLoading: false,
      error: null,
    });
    render(React.createElement(Screen));
    expect(screen.queryByTestId('kb-trigger-button')).toBeNull();
  });

  it('navigates to activity detail after a prospecting trigger succeeds', async () => {
    mockUseLocalSearchParams.mockReturnValue({ id: 'lead-1' });
    const { default: Screen } = require('../../app/(tabs)/sales/leads/[id]');
    render(React.createElement(Screen));

    fireEvent.press(screen.getByTestId('prospecting-trigger-button'));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/activity/run-pro-1');
    });
  });

  it('serializes insights dates to RFC3339 before the trigger', async () => {
    mockUseLocalSearchParams.mockReturnValue({});
    const mutateAsync = jest.fn().mockResolvedValue({ runId: 'run-ins-1', status: 'queued', agent: 'insights' });
    mockUseTriggerInsightsAgent.mockReturnValue({ mutateAsync, isPending: false });
    const { default: Screen } = require('../../app/(tabs)/activity/insights');
    render(React.createElement(Screen));

    fireEvent.changeText(screen.getByTestId('insights-query-input'), 'show stalled deals');
    fireEvent.changeText(screen.getByTestId('insights-date-from'), '2026-04-01');
    fireEvent.changeText(screen.getByTestId('insights-date-to'), '2026-04-13');
    fireEvent.press(screen.getByTestId('insights-run-button'));

    await waitFor(() => {
      expect(mutateAsync).toHaveBeenCalledWith({
        query: 'show stalled deals',
        date_from: '2026-04-01T00:00:00.000Z',
        date_to: '2026-04-13T23:59:59.000Z',
      });
    });
  });

  it('keeps trigger buttons disabled while pending across surfaces', () => {
    mockUseTriggerKBAgent.mockReturnValue({ mutate: jest.fn(), isPending: true });
    const { default: SupportScreen } = require('../../app/(tabs)/support/[id]');
    render(React.createElement(SupportScreen));
    expect(screen.getByTestId('kb-trigger-button').props.accessibilityState?.disabled).toBe(true);

    mockUseLocalSearchParams.mockReturnValue({ id: 'lead-1' });
    mockUseTriggerProspectingAgent.mockReturnValue({ mutateAsync: jest.fn(), isPending: true });
    const { default: LeadScreen } = require('../../app/(tabs)/sales/leads/[id]');
    render(React.createElement(LeadScreen));
    expect(screen.getByTestId('prospecting-trigger-button').props.accessibilityState?.disabled).toBe(true);

    mockUseLocalSearchParams.mockReturnValue({});
    mockUseTriggerInsightsAgent.mockReturnValue({ mutateAsync: jest.fn(), isPending: true });
    const { default: InsightsScreen } = require('../../app/(tabs)/activity/insights');
    render(React.createElement(InsightsScreen));
    fireEvent.changeText(screen.getByTestId('insights-query-input'), 'query');
    expect(screen.getByTestId('insights-run-button').props.accessibilityState?.disabled).toBe(true);
  });
});
