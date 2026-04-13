import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'case-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseCase = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useCase: (...args: unknown[]) => mockUseCase(...args),
}));

const mockUseTriggerKBAgent = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerSupportAgent: () => ({ mutate: jest.fn(), isPending: false }),
  useTriggerKBAgent: () => mockUseTriggerKBAgent(),
  useAgentRuns: () => ({ data: null }),
}));

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CRMDetailHeader: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-header` }),
  };
});

jest.mock('../../../../src/components/agents/AgentActivitySection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    AgentActivitySection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('../../../../src/components/signals/EntitySignalsSection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    EntitySignalsSection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('../../../../src/components/signals/SignalCountBadge', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    SignalCountBadge: ({ testID }: { testID: string }) => React.createElement(View, { testID }),
  };
});

jest.mock('react-native-paper', () => {
  const React = require('react');
  const { TouchableOpacity, Text } = require('react-native');
  return {
    useTheme: () => ({
      colors: {
        primary: '#E53935',
        surface: '#f5f5f5',
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
  };
});

const makeCasePayload = (status: string) => ({
  case: {
    id: 'case-1',
    subject: 'Login broken',
    status,
    priority: 'high',
    description: 'Users cannot log in',
  },
  active_signal_count: 0,
});

describe('Support KB trigger', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseCase.mockReturnValue({ data: makeCasePayload('resolved'), isLoading: false, error: null });
    mockUseTriggerKBAgent.mockReturnValue({ mutate: jest.fn(), isPending: false });
  });

  it('does not render the KB trigger button when the case is not resolved', () => {
    mockUseCase.mockReturnValue({ data: makeCasePayload('open'), isLoading: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('kb-trigger-button')).toBeNull();
  });

  it('renders the KB trigger button when the case is resolved', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('kb-trigger-button')).toBeTruthy();
  });

  it('calls mutate with the case id when pressed', () => {
    const mockMutate = jest.fn();
    mockUseTriggerKBAgent.mockReturnValue({ mutate: mockMutate, isPending: false });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('kb-trigger-button'));
    expect(mockMutate).toHaveBeenCalledWith({ caseId: 'case-1' });
  });

  it('disables the button and shows running label while pending', () => {
    mockUseTriggerKBAgent.mockReturnValue({ mutate: jest.fn(), isPending: true });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    const button = screen.getByTestId('kb-trigger-button');
    expect(button.props.accessibilityState?.disabled).toBe(true);
    expect(screen.getByText('Running...')).toBeTruthy();
  });
});
