import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'lead-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseLead = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useLead: () => mockUseLead(),
}));

const mockUseTriggerProspectingAgent = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerProspectingAgent: () => mockUseTriggerProspectingAgent(),
}));

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  return {
    CRMDetailHeader: ({ title, testIDPrefix }: { title: string; testIDPrefix: string }) =>
      React.createElement(
        View,
        { testID: `${testIDPrefix}-header` },
        React.createElement(Text, null, title),
      ),
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

jest.mock('react-native-paper', () => {
  const React = require('react');
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
      React.createElement(
        TouchableOpacity,
        { testID, onPress, accessibilityState: { disabled: !!disabled } },
        React.createElement(Text, null, children),
      ),
  };
});

const leadPayload = {
  id: 'lead-1',
  source: 'website',
  status: 'new',
  ownerId: 'owner-1',
  metadata: JSON.stringify({ name: 'Jane Roe', email: 'jane@example.com', company: 'Acme Corp' }),
};

describe('Sales prospecting trigger', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseLead.mockReturnValue({
      data: leadPayload,
      isLoading: false,
      error: null,
    });
    mockUseTriggerProspectingAgent.mockReturnValue({
      mutateAsync: jest.fn().mockResolvedValue({ runId: 'run-1', status: 'queued', agent: 'prospecting' }),
      isPending: false,
    });
  });

  it('renders the prospecting trigger button on lead detail', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/leads/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('prospecting-trigger-button')).toBeTruthy();
  });

  it('calls mutateAsync with leadId and navigates to the run detail on success', async () => {
    const mutateAsync = jest.fn().mockResolvedValue({ runId: 'run-1', status: 'queued', agent: 'prospecting' });
    mockUseTriggerProspectingAgent.mockReturnValue({ mutateAsync, isPending: false });
    const { default: Screen } = require('../../../../app/(tabs)/sales/leads/[id]');
    render(React.createElement(Screen));

    fireEvent.press(screen.getByTestId('prospecting-trigger-button'));

    await waitFor(() => {
      expect(mutateAsync).toHaveBeenCalledWith({ leadId: 'lead-1' });
      expect(mockPush).toHaveBeenCalledWith('/activity/run-1');
    });
  });

  it('disables the prospecting trigger while pending', () => {
    mockUseTriggerProspectingAgent.mockReturnValue({ mutateAsync: jest.fn(), isPending: true });
    const { default: Screen } = require('../../../../app/(tabs)/sales/leads/[id]');
    render(React.createElement(Screen));
    const button = screen.getByTestId('prospecting-trigger-button');
    expect(button.props.accessibilityState?.disabled).toBe(true);
    expect(screen.getByText('Running...')).toBeTruthy();
  });
});
