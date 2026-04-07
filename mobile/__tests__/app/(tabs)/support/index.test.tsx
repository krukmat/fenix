// Support case list screen tests — W3-T1
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';
import SupportScreen from '../../../../app/(tabs)/support/index';

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

jest.mock('../../../../src/hooks/useWedge', () => ({
  useInbox: () => ({ data: null, isLoading: false }),
}));

jest.mock('../../../../src/components/crm', () => {
  const React = require('react');
  const { FlatList, View, Text } = require('react-native');
  return {
    CRMListScreen: ({
      data,
      renderItem,
      testIDPrefix,
      emptyTitle,
      loading,
    }: {
      data: unknown[];
      renderItem: (info: { item: unknown; index: number }) => React.ReactNode;
      testIDPrefix: string;
      emptyTitle: string;
      loading: boolean;
    }) =>
      React.createElement(
        View,
        { testID: `${testIDPrefix}-list-container` },
        loading
          ? React.createElement(Text, { testID: 'loading-indicator' }, 'Loading...')
          : data.length === 0
          ? React.createElement(Text, { testID: 'empty-state' }, emptyTitle)
          : React.createElement(FlatList, {
              data,
              keyExtractor: (_: unknown, i: number) => String(i),
              renderItem,
            })
      ),
  };
});

jest.mock('react-native-paper', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    useTheme: () => ({ colors: { primary: '#E53935', surface: '#fff', onSurface: '#000', onSurfaceVariant: '#666', background: '#fff', error: '#B00020' } }),
    FAB: ({ testID, onPress }: { testID: string; onPress: () => void }) =>
      React.createElement(View, { testID, onTouchEnd: onPress }),
  };
});

jest.mock('../../../../src/components/signals/SignalCountBadge', () => {
  const React = require('react');
  const { View } = require('react-native');
  return { SignalCountBadge: ({ testID }: { testID: string }) => React.createElement(View, { testID }) };
});

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const makePage = (items: object[]) => ({ pages: [{ data: items }] });

const caseA = { id: 'case-1', subject: 'Login broken', status: 'open', priority: 'high' as const, accountName: 'Acme' };
const caseB = { id: 'case-2', subject: 'Slow checkout', status: 'pending', priority: 'medium' as const };

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Support case list screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseCases.mockReturnValue({
      data: makePage([caseA, caseB]),
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
      refetch: jest.fn(),
      isRefetching: false,
    });
  });

  it('renders the support cases list container', () => {
    render(React.createElement(SupportScreen));
    expect(screen.getByTestId('support-cases-list-container')).toBeTruthy();
  });

  it('renders case items from useCases', () => {
    render(React.createElement(SupportScreen));
    expect(screen.getByTestId('support-cases-list-item-0')).toBeTruthy();
    expect(screen.getByTestId('support-cases-list-item-1')).toBeTruthy();
  });

  it('navigates to /support/[id] when a case is pressed', () => {
    render(React.createElement(SupportScreen));
    fireEvent.press(screen.getByTestId('support-cases-list-item-0'));
    expect(mockPush).toHaveBeenCalledWith('/support/case-1');
  });

  it('shows empty state when no cases', () => {
    mockUseCases.mockReturnValue({
      data: makePage([]),
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
      refetch: jest.fn(),
      isRefetching: false,
    });
    render(React.createElement(SupportScreen));
    expect(screen.getByTestId('empty-state')).toBeTruthy();
  });

  it('shows loading indicator while fetching', () => {
    mockUseCases.mockReturnValue({
      data: undefined,
      isLoading: true,
      isFetchingNextPage: false,
      hasNextPage: false,
      fetchNextPage: jest.fn(),
      error: null,
      refetch: jest.fn(),
      isRefetching: false,
    });
    render(React.createElement(SupportScreen));
    expect(screen.getByTestId('loading-indicator')).toBeTruthy();
  });

  it('does NOT render a create FAB (creation removed from support wedge)', () => {
    render(React.createElement(SupportScreen));
    expect(screen.queryByTestId('create-case-fab')).toBeNull();
  });
});
