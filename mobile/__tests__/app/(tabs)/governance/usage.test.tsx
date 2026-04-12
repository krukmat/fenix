import React from 'react';
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render, screen } from '@testing-library/react-native';

const mockUseUsageEvents = jest.fn();
let mockSearchParams: Record<string, string> = {};

jest.mock('expo-router', () => ({
  __esModule: true,
  useLocalSearchParams: () => mockSearchParams,
}));

jest.mock('../../../../src/hooks/useWedge', () => ({
  useUsageEvents: (...args: unknown[]) => mockUseUsageEvents(...args),
}));

jest.mock('react-native-paper', () => ({
  useTheme: () => ({
    colors: {
      primary: '#E53935', surface: '#fff', onSurface: '#000',
      onSurfaceVariant: '#666', background: '#fff', error: '#B00020',
    },
  }),
}));

jest.mock('../../../../src/components/governance/UsageDetailCard', () => ({
  UsageDetailCard: ({ testIDPrefix }: { testIDPrefix: string }) => {
    const React = require('react');
    const { View } = require('react-native');
    return React.createElement(View, { testID: `${testIDPrefix}-card` });
  },
}));

jest.mock('../../../../src/components/governance/UsageCostSummaryCard', () => ({
  UsageCostSummaryCard: ({ testIDPrefix }: { testIDPrefix: string }) => {
    const React = require('react');
    const { View } = require('react-native');
    return React.createElement(View, { testID: `${testIDPrefix}-card` });
  },
}));

const usagePayload = {
  data: [
    {
      id: 'u-1',
      workspaceId: 'ws-1',
      actorType: 'agent',
      toolName: 'create_task',
      modelName: 'gpt-5.4',
      inputUnits: 100,
      outputUnits: 40,
      estimatedCost: 0.01,
      createdAt: '2026-04-12T10:00:00Z',
    },
  ],
  meta: { total: 1, limit: 20, offset: 0 },
};

function buildUsageEvents(count: number) {
  return Array.from({ length: count }, (_, index) => ({
    id: `u-${index + 1}`,
    workspaceId: 'ws-1',
    actorType: 'agent' as const,
    toolName: 'create_task',
    modelName: 'gpt-5.4',
    inputUnits: 100,
    outputUnits: 40,
    estimatedCost: 0.01,
    createdAt: '2026-04-12T10:00:00Z',
  }));
}

describe('Governance usage screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockSearchParams = {};
    mockUseUsageEvents.mockImplementation((filters = undefined, page = 1) => ({
      data: { ...usagePayload, meta: { ...usagePayload.meta, limit: (page as number) * 20 } },
      isLoading: false,
      isFetching: false,
      error: null,
    }));
  });

  it('shows loading state', () => {
    mockUseUsageEvents.mockReturnValue({ data: undefined, isLoading: true, isFetching: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/governance/usage');
    render(React.createElement(Screen));
    expect(screen.getByTestId('usage-loading')).toBeTruthy();
  });

  it('renders summary card and usage cards', () => {
    const { default: Screen } = require('../../../../app/(tabs)/governance/usage');
    render(React.createElement(Screen));
    expect(screen.getByTestId('usage-screen')).toBeTruthy();
    expect(screen.getByTestId('usage-summary-card')).toBeTruthy();
    expect(screen.getByTestId('usage-event-0-card')).toBeTruthy();
  });

  it('shows empty state', () => {
    mockUseUsageEvents.mockReturnValue({
      data: { data: [], meta: { total: 0, limit: 20, offset: 0 } },
      isLoading: false,
      isFetching: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/governance/usage');
    render(React.createElement(Screen));
    expect(screen.getByTestId('usage-empty')).toBeTruthy();
  });

  it('prefills run_id from route params', () => {
    mockSearchParams = { run_id: 'run-1' };
    const { default: Screen } = require('../../../../app/(tabs)/governance/usage');
    render(React.createElement(Screen));
    expect(mockUseUsageEvents).toHaveBeenCalledWith({ run_id: 'run-1' }, 1);
    expect(screen.getByTestId('usage-run-filter')).toBeTruthy();
  });

  it('requests more events when the list reaches the end', () => {
    mockUseUsageEvents.mockImplementation((filters = undefined, page = 1) => ({
      data: {
        data: buildUsageEvents((page as number) * 20),
        meta: { total: (page as number) * 20, limit: (page as number) * 20, offset: 0 },
      },
      isLoading: false,
      isFetching: false,
      error: null,
    }));
    const { default: Screen } = require('../../../../app/(tabs)/governance/usage');
    render(React.createElement(Screen));
    fireEvent(screen.getByTestId('usage-screen'), 'endReached');
    expect(mockUseUsageEvents).toHaveBeenLastCalledWith(undefined, 2);
  });
});
