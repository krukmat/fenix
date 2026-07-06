import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react-native';
import * as mockHookMocks from './testSupportDetailHookMocks';
import * as mockComponentMocks from './testSupportDetailComponentMocks';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'case-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

jest.mock('../../../../src/hooks/useCRM', () => mockHookMocks.mockSupportDetailUseCRMModule());

const mockUseTriggerKBAgent = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerSupportAgent: () => ({ mutate: jest.fn(), isPending: false }),
  useTriggerKBAgent: () => mockUseTriggerKBAgent(),
  useAgentRuns: () => ({ data: null }),
}));

jest.mock('../../../../src/components/crm', () => mockComponentMocks.mockSupportDetailCRMModule());

jest.mock('../../../../src/components/copilot', () => mockComponentMocks.mockInlineSupportCopilotModule());

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

jest.mock('react-native-paper', () => mockComponentMocks.mockSupportDetailPaperModule());

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
    mockHookMocks.mockUseCase.mockReturnValue({ data: makeCasePayload('resolved'), isLoading: false, error: null });
    mockHookMocks.seedEmptySupportDetailQueries();
    mockUseTriggerKBAgent.mockReturnValue({ mutate: jest.fn(), isPending: false });
  });

  it('does not render the KB trigger button when the case is not resolved', () => {
    mockHookMocks.mockUseCase.mockReturnValue({ data: makeCasePayload('open'), isLoading: false, error: null });
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
    expect(mockMutate).toHaveBeenCalledWith({ caseId: 'case-1' }, expect.objectContaining({ onSuccess: expect.any(Function) }));
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
