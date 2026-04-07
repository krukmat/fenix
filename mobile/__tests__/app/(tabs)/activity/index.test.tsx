// Activity log list screen tests — W5-T1
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseAgentRuns = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useAgentRuns: (filters?: unknown) => mockUseAgentRuns(filters),
}));

jest.mock('react-native-paper', () => ({
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
}));

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const makeData = (items: object[]) => ({ data: items, total: items.length });

const runA = { id: 'run-1', agent_name: 'Support Agent', status: 'completed', started_at: '2026-04-07T10:00:00Z', latency_ms: 1200, cost_euros: 0.05 };
const runB = { id: 'run-2', agent_name: 'Sales Agent', status: 'awaiting_approval', started_at: '2026-04-07T09:00:00Z', latency_ms: 800, cost_euros: 0.02 };
const runC = { id: 'run-3', agent_name: 'Support Agent', status: 'failed', started_at: '2026-04-07T08:00:00Z', latency_ms: 300, cost_euros: 0.01 };

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Activity log list screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAgentRuns.mockReturnValue({
      data: makeData([runA, runB, runC]),
      isLoading: false,
      error: null,
      fetchNextPage: jest.fn(),
      hasNextPage: false,
    });
  });

  it('renders the activity log screen', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-log-screen')).toBeTruthy();
  });

  it('renders filter chips for normalized public outcomes', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('filter-all')).toBeTruthy();
    expect(screen.getByTestId('filter-completed')).toBeTruthy();
    expect(screen.getByTestId('filter-awaiting_approval')).toBeTruthy();
    expect(screen.getByTestId('filter-failed')).toBeTruthy();
  });

  it('renders all runs when All filter is active', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-run-item-run-1')).toBeTruthy();
    expect(screen.getByTestId('activity-run-item-run-2')).toBeTruthy();
    expect(screen.getByTestId('activity-run-item-run-3')).toBeTruthy();
  });

  it('navigates to activity detail when a run is pressed', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('activity-run-item-run-1'));
    expect(mockPush).toHaveBeenCalledWith('/activity/run-1');
  });

  it('filters runs by status when a filter chip is pressed', () => {
    mockUseAgentRuns.mockImplementation((filters: { status?: string } | undefined) => {
      const all = [runA, runB, runC];
      const items = filters?.status ? all.filter(r => r.status === filters.status) : all;
      return { data: makeData(items), isLoading: false, error: null };
    });
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('filter-completed'));
    expect(screen.getByTestId('activity-run-item-run-1')).toBeTruthy();
    expect(screen.queryByTestId('activity-run-item-run-2')).toBeNull();
  });

  it('shows loading state', () => {
    mockUseAgentRuns.mockReturnValue({ data: undefined, isLoading: true, error: null,  });
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-log-loading')).toBeTruthy();
  });

  it('shows empty state when no runs', () => {
    mockUseAgentRuns.mockReturnValue({ data: makeData([]), isLoading: false, error: null,  });
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-log-empty')).toBeTruthy();
  });

  it('shows public status badge using normalized status', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/index');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-status-run-2')).toBeTruthy();
  });
});
