// Support inbox connections tests — W3-T5
// Verifies that the support index shows inbox badge count and
// that the inbox tab navigates to /support when a handoff is tapped
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';

// ─── Mocks ────────────────────────────────────────────────────────────────────

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseCases = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useCases: () => mockUseCases(),
}));

const mockUseInbox = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useInbox: () => mockUseInbox(),
  useApproveApproval: () => ({ mutate: jest.fn(), isPending: false }),
  useRejectApproval: () => ({ mutate: jest.fn(), isPending: false }),
}));

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CRMListScreen: ({ testIDPrefix, data }: { testIDPrefix: string; data: unknown[] }) =>
      React.createElement(View, { testID: `${testIDPrefix}-list-container`, accessibilityLabel: String(data.length) }),
  };
});

jest.mock('react-native-paper', () => ({
  useTheme: () => ({ colors: { primary: '#E53935', surface: '#f5f5f5', onSurface: '#000', onSurfaceVariant: '#666', background: '#fff', error: '#B00020' } }),
  FAB: jest.fn(() => null),
}));

jest.mock('../../../../src/components/signals/SignalCountBadge', () => ({
  SignalCountBadge: jest.fn(() => null),
}));

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Support inbox connections', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseCases.mockReturnValue({
      data: { pages: [{ data: [] }] },
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
      refetch: jest.fn(),
      isRefetching: false,
    });
    mockUseInbox.mockReturnValue({ data: null, isLoading: false });
  });

  it('shows pending inbox badge when inbox has approvals or handoffs', () => {
    mockUseInbox.mockReturnValue({
      data: { approvals: [{ id: 'a1' }], handoffs: [{ run_id: 'r1' }], signals: [] },
      isLoading: false,
    });
    const { default: Screen } = require('../../../../app/(tabs)/support/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-inbox-badge')).toBeTruthy();
  });

  it('does not show inbox badge when inbox is empty', () => {
    mockUseInbox.mockReturnValue({
      data: { approvals: [], handoffs: [], signals: [] },
      isLoading: false,
    });
    const { default: Screen } = require('../../../../app/(tabs)/support/index');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('support-inbox-badge')).toBeNull();
  });

  it('navigates to /inbox when badge is pressed', () => {
    mockUseInbox.mockReturnValue({
      data: { approvals: [{ id: 'a1' }], handoffs: [], signals: [] },
      isLoading: false,
    });
    const { default: Screen } = require('../../../../app/(tabs)/support/index');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('support-inbox-badge'));
    expect(mockPush).toHaveBeenCalledWith('/inbox');
  });
});
