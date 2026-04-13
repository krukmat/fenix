// Support case detail screen tests — W3-T2
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

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

jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerSupportAgent: () => ({ mutate: jest.fn(), isPending: false }),
  useTriggerKBAgent: () => ({ mutate: jest.fn(), isPending: false }),
  useAgentRuns: () => ({ data: null }),
}));

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  return {
    CRMDetailHeader: ({ title, testIDPrefix }: { title: string; testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-header` },
        React.createElement(Text, null, title)
      ),
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
    useTheme: () => ({ colors: { primary: '#E53935', surface: '#f5f5f5', onSurface: '#000', onSurfaceVariant: '#666', background: '#fff', error: '#B00020' } }),
    Button: ({ testID, onPress, children }: { testID: string; onPress: () => void; children: React.ReactNode }) =>
      React.createElement(TouchableOpacity, { testID, onPress },
        React.createElement(Text, null, children)
      ),
  };
});

// ─── Fixture ──────────────────────────────────────────────────────────────────

const casePayload = {
  case: {
    id: 'case-1',
    subject: 'Login broken',
    status: 'open',
    priority: 'high',
    description: 'Users cannot log in',
    accountId: 'acc-1',
    sla_deadline: '2026-04-10T00:00:00Z',
  },
  account: { name: 'Acme Corp' },
  handoff: { status: 'escalated' },
  active_signal_count: 2,
};

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Support case detail screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseCase.mockReturnValue({ data: casePayload, isLoading: false, error: null });
  });

  it('renders the detail screen container', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-screen')).toBeTruthy();
  });

  it('renders the case header with subject', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-header')).toBeTruthy();
  });

  it('renders SLA deadline when present', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-sla-deadline')).toBeTruthy();
  });

  it('renders handoff status when present', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-handoff-status')).toBeTruthy();
  });

  it('renders agent activity section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-agent-activity-section')).toBeTruthy();
  });

  it('navigates to support copilot route when copilot button pressed', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    const { fireEvent } = require('@testing-library/react-native');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('support-copilot-button'));
    expect(mockPush).toHaveBeenCalledWith(
      expect.objectContaining({ pathname: '/support/case-1/copilot' })
    );
  });

  it('does NOT show edit case button (edit removed from wedge)', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('case-edit-button')).toBeNull();
  });

  it('shows loading state', () => {
    mockUseCase.mockReturnValue({ data: undefined, isLoading: true, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-loading')).toBeTruthy();
  });

  it('shows error state', () => {
    mockUseCase.mockReturnValue({ data: undefined, isLoading: false, error: new Error('Not found') });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-error')).toBeTruthy();
  });
});
