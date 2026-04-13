import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseTriggerInsightsAgent = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerInsightsAgent: () => mockUseTriggerInsightsAgent(),
}));

jest.mock('react-native-paper', () => {
  const React = require('react');
  const { View, Text, TextInput: RNTextInput, TouchableOpacity } = require('react-native');
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
    TextInput: ({
      testID,
      value,
      onChangeText,
      placeholder,
    }: {
      testID: string;
      value?: string;
      onChangeText: (value: string) => void;
      placeholder?: string;
    }) => React.createElement(RNTextInput, { testID, value, onChangeText, placeholder }),
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
    ActivityIndicator: ({ testID }: { testID?: string }) => React.createElement(View, { testID }),
  };
});

describe('Insights screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseTriggerInsightsAgent.mockReturnValue({
      mutateAsync: jest.fn().mockResolvedValue({ runId: 'run-ins-1', status: 'queued', agent: 'insights' }),
      isPending: false,
    });
  });

  it('renders query input, date inputs and run button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/insights');
    render(React.createElement(Screen));

    expect(screen.getByTestId('insights-query-input')).toBeTruthy();
    expect(screen.getByTestId('insights-date-from')).toBeTruthy();
    expect(screen.getByTestId('insights-date-to')).toBeTruthy();
    expect(screen.getByTestId('insights-run-button')).toBeTruthy();
  });

  it('keeps the run button disabled when query is empty', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/insights');
    render(React.createElement(Screen));
    expect(screen.getByTestId('insights-run-button').props.accessibilityState?.disabled).toBe(true);
  });

  it('enables the run button when query has content', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/insights');
    render(React.createElement(Screen));
    fireEvent.changeText(screen.getByTestId('insights-query-input'), 'show stalled deals');
    expect(screen.getByTestId('insights-run-button').props.accessibilityState?.disabled).toBe(false);
  });

  it('serializes dates to RFC3339 and navigates to activity detail on success', async () => {
    const mutateAsync = jest.fn().mockResolvedValue({ runId: 'run-ins-1', status: 'queued', agent: 'insights' });
    mockUseTriggerInsightsAgent.mockReturnValue({ mutateAsync, isPending: false });
    const { default: Screen } = require('../../../../app/(tabs)/activity/insights');
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
      expect(mockPush).toHaveBeenCalledWith('/activity/run-ins-1');
    });
  });

  it('shows pending state while running', () => {
    mockUseTriggerInsightsAgent.mockReturnValue({ mutateAsync: jest.fn(), isPending: true });
    const { default: Screen } = require('../../../../app/(tabs)/activity/insights');
    render(React.createElement(Screen));
    fireEvent.changeText(screen.getByTestId('insights-query-input'), 'show stalled deals');

    expect(screen.getByTestId('insights-run-button').props.accessibilityState?.disabled).toBe(true);
    expect(screen.getByText('Running...')).toBeTruthy();
  });
});
