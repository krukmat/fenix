// Support agent trigger flow tests — W3-T3
// Verifies that SupportCaseDetailScreen triggers support agent and reflects run status
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';

// ─── Mocks ────────────────────────────────────────────────────────────────────

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

const mockUseTriggerSupportAgent = jest.fn();
const mockUseAgentRuns = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerSupportAgent: () => mockUseTriggerSupportAgent(),
  useAgentRuns: (...args: unknown[]) => mockUseAgentRuns(...args),
}));

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
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
  return { SignalCountBadge: ({ testID }: { testID: string }) => React.createElement(View, { testID }) };
});

jest.mock('react-native-paper', () => {
  const React = require('react');
  const { TouchableOpacity, Text, View } = require('react-native');
  return {
    useTheme: () => ({ colors: { primary: '#E53935', surface: '#f5f5f5', onSurface: '#000', onSurfaceVariant: '#666', background: '#fff', error: '#B00020' } }),
    Button: ({ testID, onPress, children, disabled }: { testID: string; onPress: () => void; children: React.ReactNode; disabled?: boolean }) =>
      React.createElement(TouchableOpacity, { testID, onPress, accessibilityState: { disabled: !!disabled } },
        React.createElement(Text, null, children)
      ),
    ActivityIndicator: ({ testID }: { testID?: string }) => React.createElement(View, { testID }),
  };
});

// ─── Fixture ──────────────────────────────────────────────────────────────────

const casePayload = {
  case: { id: 'case-1', subject: 'Login broken', status: 'open', priority: 'high' },
  active_signal_count: 0,
};

const makeRun = (status: string) => ({
  id: 'run-1',
  agent_name: 'Support Agent',
  status,
  runtime_status: 'running',
  entity_type: 'case',
  entity_id: 'case-1',
  started_at: '2026-04-07T10:00:00Z',
  latency_ms: 0,
  cost_euros: 0,
});

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Support agent trigger flow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseCase.mockReturnValue({ data: casePayload, isLoading: false, error: null });
    mockUseTriggerSupportAgent.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseAgentRuns.mockReturnValue({ data: null });
  });

  it('renders the trigger agent button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-trigger-agent-button')).toBeTruthy();
  });

  it('calls triggerSupportAgent mutate with entity context on press', () => {
    const mockMutate = jest.fn();
    mockUseTriggerSupportAgent.mockReturnValue({ mutate: mockMutate, isPending: false });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('support-trigger-agent-button'));
    expect(mockMutate).toHaveBeenCalledWith({ entityType: 'case', entityId: 'case-1' });
  });

  it('disables trigger button while agent run is pending', () => {
    mockUseTriggerSupportAgent.mockReturnValue({ mutate: jest.fn(), isPending: true });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    const btn = screen.getByTestId('support-trigger-agent-button');
    expect(btn.props.accessibilityState?.disabled).toBe(true);
  });

  it('shows active run status badge when a run is in progress', () => {
    mockUseAgentRuns.mockReturnValue({ data: { data: [makeRun('awaiting_approval')] } });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-active-run-status')).toBeTruthy();
  });

  it('does not show run status badge when no active run', () => {
    mockUseAgentRuns.mockReturnValue({ data: null });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('support-active-run-status')).toBeNull();
  });
});
