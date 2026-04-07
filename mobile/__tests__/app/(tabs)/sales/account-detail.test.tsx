// Sales account detail screen tests — W4-T2
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'acc-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseAccount = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useAccount: () => mockUseAccount(),
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
    Button: ({ children, onPress, testID }: { children: unknown; onPress: () => void; testID: string }) =>
      mockReact.createElement(TouchableOpacity, { testID, onPress }, mockReact.createElement(Text, null, children)),
  };
});

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  return {
    CRMDetailHeader: ({ title, testIDPrefix }: { title: string; testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-header` },
        React.createElement(Text, null, title)),
    EntityTimeline: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-timeline` }),
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

jest.mock('../../../../src/components/signals/SignalCountBadge', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    SignalCountBadge: ({ testID }: { testID: string }) =>
      React.createElement(View, { testID }),
  };
});

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const accountPayload = {
  account: {
    id: 'acc-1',
    name: 'Acme Corp',
    industry: 'Technology',
    website: 'https://acme.com',
    phone: '+1-555-0100',
    owner: 'Alice',
  },
  active_signal_count: 3,
};

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Sales account detail screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAccount.mockReturnValue({
      data: accountPayload,
      isLoading: false,
      error: null,
    });
  });

  it('renders the detail screen', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-account-detail-screen')).toBeTruthy();
  });

  it('renders account header', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-account-detail-header')).toBeTruthy();
  });

  it('renders agent activity section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-account-detail-agent-activity')).toBeTruthy();
  });

  it('renders signals section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-account-detail-signals')).toBeTruthy();
  });

  it('renders Sales Brief button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-button')).toBeTruthy();
  });

  it('navigates to /sales/[id]/brief when Sales Brief button is pressed', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('sales-brief-button'));
    expect(mockPush).toHaveBeenCalledWith(expect.objectContaining({ pathname: '/sales/acc-1/brief' }));
  });

  it('renders Copilot button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-copilot-button')).toBeTruthy();
  });

  it('does NOT render an edit button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('sales-account-edit-button')).toBeNull();
  });

  it('shows loading state', () => {
    mockUseAccount.mockReturnValue({ data: undefined, isLoading: true, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-account-detail-loading')).toBeTruthy();
  });

  it('shows error state when account not found', () => {
    mockUseAccount.mockReturnValue({ data: null, isLoading: false, error: new Error('Not found') });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-account-detail-error')).toBeTruthy();
  });
});
