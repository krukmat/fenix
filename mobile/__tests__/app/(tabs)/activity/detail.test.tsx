// Activity log detail screen tests — W5-T2
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'run-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseAgentRun = jest.fn();
const mockUseRunUsage = jest.fn();
const mockHandoffBanner = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useAgentRun: () => mockUseAgentRun(),
}));
jest.mock('../../../../src/hooks/useWedge', () => ({
  useRunUsage: (_runId: unknown, _enabled?: unknown) => mockUseRunUsage(),
}));

jest.mock('react-native-paper', () => ({
  useTheme: () => ({
    colors: {
      primary: '#E53935', surface: '#fff', onSurface: '#000',
      onSurfaceVariant: '#666', background: '#fff', error: '#B00020',
    },
  }),
}));

jest.mock('../../../../src/components/agents/HandoffBanner', () => {
  const mockReact = require('react');
  const { View } = require('react-native');
  return {
    HandoffBanner: ({ runId, caseId, testIDPrefix }: { runId: string; caseId?: string; testIDPrefix: string }) => {
      mockHandoffBanner({ runId, caseId, testIDPrefix });
      return mockReact.createElement(View, { testID: `${testIDPrefix}-banner`, accessibilityLabel: runId });
    },
  };
});

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const fullRun = {
  data: {
    id: 'run-1',
    agent_name: 'Support Agent',
    status: 'completed',
    runtime_status: 'success',
    triggered_by: 'user-1',
    trigger_type: 'manual',
    inputs: { case_id: 'case-1' },
    evidence_retrieved: [{ source_id: 'src-1', score: 0.95, snippet: 'Customer reported...' }],
    reasoning_trace: ['Step 1: Retrieved evidence'],
    tool_calls: [{ tool_name: 'create_task', params: {}, result: {}, latency_ms: 120 }],
    output: 'Case resolved with KB article.',
    audit_events: [{ actor_id: 'user-1', action: 'trigger', timestamp: '2026-04-07T10:00:00Z', outcome: 'success' }],
    created_at: '2026-04-07T10:00:00Z',
    started_at: '2026-04-07T10:00:00Z',
    latency_ms: 1200,
    cost_euros: 0.05,
    rejection_reason: undefined,
  },
};

const usageEvents = [
  { id: 'u-1', run_id: 'run-1', metric_name: 'tokens', value: 1500, recorded_at: '2026-04-07T10:00:01Z' },
];

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Activity log detail screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAgentRun.mockReturnValue({ data: fullRun, isLoading: false, error: null });
    mockUseRunUsage.mockReturnValue({ data: usageEvents, isLoading: false, error: null });
  });

  it('renders the detail screen', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-run-detail-screen')).toBeTruthy();
  });

  it('shows public status chip', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-public-status')).toBeTruthy();
  });

  it('shows runtime status in diagnostics section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-runtime-status')).toBeTruthy();
  });

  it('renders evidence section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-evidence')).toBeTruthy();
  });

  it('renders audit events section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-audit')).toBeTruthy();
  });

  it('renders tool calls section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-tool-calls')).toBeTruthy();
  });

  it('renders output section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-output')).toBeTruthy();
  });

  it('renders object output payloads without crashing', () => {
    mockUseAgentRun.mockReturnValue({
      data: {
        data: {
          ...fullRun.data,
          status: 'handed_off',
          output: {
            agent_name: 'Support Agent',
            entity_type: 'case',
            entity_id: 'case-1',
            rejection_reason: '',
          },
        },
      },
      isLoading: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-output')).toBeTruthy();
    expect(screen.getByText(/"entity_type": "case"/)).toBeTruthy();
  });

  it('renders per-run usage section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-usage')).toBeTruthy();
  });

  it('shows rejection reason section only when status is denied_by_policy', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...fullRun.data, status: 'denied_by_policy', rejection_reason: 'Over quota' } },
      isLoading: false, error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-rejection-reason')).toBeTruthy();
  });

  it('does NOT show rejection reason when status is not denied_by_policy', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('activity-detail-rejection-reason')).toBeNull();
  });

  it('shows handoff banner when status is handed_off', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...fullRun.data, status: 'handed_off', runtime_status: 'escalated' } },
      isLoading: false, error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-handoff-banner')).toBeTruthy();
    expect(screen.getByTestId('activity-detail-handoff-banner').props.accessibilityLabel).toBe('run-1');
    expect(mockHandoffBanner).toHaveBeenCalledWith({
      runId: 'run-1',
      caseId: undefined,
      testIDPrefix: 'activity-detail-handoff',
    });
  });

  it('does not render handoff banner when run is not handed_off', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('activity-detail-handoff-banner')).toBeNull();
  });

  it('shows loading state', () => {
    mockUseAgentRun.mockReturnValue({ data: undefined, isLoading: true, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-run-detail-loading')).toBeTruthy();
  });

  it('shows error state', () => {
    mockUseAgentRun.mockReturnValue({ data: null, isLoading: false, error: new Error('Not found') });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-run-detail-error')).toBeTruthy();
  });
});
