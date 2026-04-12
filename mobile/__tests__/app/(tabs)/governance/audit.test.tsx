import React from 'react';
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render, screen } from '@testing-library/react-native';

const mockUseAuditEvents = jest.fn();

jest.mock('../../../../src/hooks/useWedge', () => ({
  useAuditEvents: (...args: unknown[]) => mockUseAuditEvents(...args),
}));

jest.mock('react-native-paper', () => ({
  useTheme: () => ({
    colors: {
      primary: '#E53935', surface: '#fff', onSurface: '#000',
      onSurfaceVariant: '#666', background: '#fff', error: '#B00020',
    },
  }),
  Text: require('react-native').Text,
}));

jest.mock('../../../../src/components/governance/AuditEventCard', () => ({
  AuditEventCard: ({ testIDPrefix }: { testIDPrefix: string }) => {
    const React = require('react');
    const { View } = require('react-native');
    return React.createElement(View, { testID: `${testIDPrefix}-card` });
  },
}));

const eventPage = {
  data: [
    {
      id: 'audit-1',
      workspace_id: 'ws-1',
      actor_id: 'user-1',
      actor_type: 'user',
      action: 'case.updated',
      outcome: 'success',
      created_at: '2026-04-12T10:00:00Z',
    },
  ],
  meta: { total: 40, limit: 20, offset: 0 },
};

describe('Governance audit screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAuditEvents.mockImplementation((_filters = {}, page = 1) => ({
      data: { ...eventPage, meta: { ...eventPage.meta, offset: ((page as number) - 1) * 20 } },
      isLoading: false,
      isFetching: false,
      error: null,
    }));
  });

  it('shows loading state', () => {
    mockUseAuditEvents.mockReturnValue({ data: undefined, isLoading: true, isFetching: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/governance/audit');
    render(React.createElement(Screen));
    expect(screen.getByTestId('audit-loading')).toBeTruthy();
  });

  it('renders audit cards on data', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/audit');
    render(React.createElement(Screen));
    expect(screen.getByTestId('audit-screen')).toBeTruthy();
    expect(screen.getByTestId('audit-event-0-card')).toBeTruthy();
  });

  it('shows empty state', () => {
    mockUseAuditEvents.mockReturnValue({
      data: { data: [], meta: { total: 0, limit: 20, offset: 0 } },
      isLoading: false,
      isFetching: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/governance/audit');
    render(React.createElement(Screen));
    expect(screen.getByTestId('audit-empty')).toBeTruthy();
  });

  it('shows error state', () => {
    mockUseAuditEvents.mockReturnValue({ data: undefined, isLoading: false, isFetching: false, error: new Error('boom') });
    const { default: Screen } = require('../../../../app/(tabs)/governance/audit');
    render(React.createElement(Screen));
    expect(screen.getByTestId('audit-error')).toBeTruthy();
  });

  it('changes outcome filter and resets query to page 1', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/audit');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('audit-filter-outcome-denied'));
    expect(mockUseAuditEvents).toHaveBeenLastCalledWith({ outcome: 'denied' }, 1);
  });

  it('loads the next page when the list reaches the end', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/audit');
    render(React.createElement(Screen));
    fireEvent(screen.getByTestId('audit-list'), 'endReached');
    expect(mockUseAuditEvents).toHaveBeenLastCalledWith({}, 2);
  });
});
