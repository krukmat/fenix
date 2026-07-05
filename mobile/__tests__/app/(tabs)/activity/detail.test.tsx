// Activity log detail screen tests — W5-T2
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react-native';

const mockRouterPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockRouterPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'run-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseAgentRun = jest.fn();
const mockUseRunUsage = jest.fn();
const mockUseHandoffPackage = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useAgentRun: () => mockUseAgentRun(),
}));
jest.mock('../../../../src/hooks/useWedge', () => ({
  useRunUsage: (_runId: unknown, _enabled?: unknown) => mockUseRunUsage(),
}));
jest.mock('../../../../src/hooks/useAgentSpec', () => ({
  useHandoffPackage: (_runId: unknown, _caseId: unknown, _enabled?: unknown) => mockUseHandoffPackage(),
}));

jest.mock('react-native-paper', () => ({
  Text: ({ children, ...props }: { children?: React.ReactNode }) => {
    const React = require('react');
    const { Text } = require('react-native');
    return React.createElement(Text, props, children);
  },
  useTheme: () => ({
    colors: {
      primary: '#E53935', surface: '#fff', onSurface: '#000',
      onSurfaceVariant: '#666', background: '#fff', error: '#B00020',
      surfaceVariant: '#f2f2f2', outline: '#ddd', outlineVariant: '#eee',
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

jest.mock('../../../../src/components/copilot/EvidenceCard', () => ({
  EvidenceCard: ({ testIDPrefix }: { testIDPrefix: string }) => {
    const React = require('react');
    const { View } = require('react-native');
    return React.createElement(View, { testID: `${testIDPrefix}-card` });
  },
}));

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
    trace_id: 'trace-1',
    created_at: '2026-04-07T10:00:00Z',
    started_at: '2026-04-07T10:00:00Z',
    latency_ms: 1200,
    cost_euros: 0.05,
    rejection_reason: undefined,
  },
};

const usageEvents = [
  {
    id: 'u-1',
    workspaceId: 'ws-1',
    actorType: 'agent',
    runId: 'run-1',
    toolName: 'create_task',
    modelName: 'gpt-5.4',
    inputUnits: 1500,
    outputUnits: 120,
    estimatedCost: 0.0134,
    latencyMs: 842,
    createdAt: '2026-04-07T10:00:01Z',
  },
];

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('Activity log detail screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAgentRun.mockReturnValue({ data: fullRun, isLoading: false, error: null });
    mockUseRunUsage.mockReturnValue({ data: usageEvents, isLoading: false, error: null });
    mockUseHandoffPackage.mockReturnValue({ data: undefined, isLoading: false, error: null });
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
    expect(screen.getByText('Runtime: Success')).toBeTruthy();
  });

  it('renders evidence section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-evidence')).toBeTruthy();
  });

  it('renders reasoning trace section when present', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-reasoning')).toBeTruthy();
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
    expect(screen.getByTestId('activity-usage-item-0-card')).toBeTruthy();
  });

  it('shows trace id and opens the audit trail filtered by trace_id', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-trace-id')).toBeTruthy();
    fireEvent.press(screen.getByTestId('activity-detail-open-audit'));
    expect(mockRouterPush).toHaveBeenCalledWith({
      pathname: '/governance/audit',
      params: { trace_id: 'trace-1' },
    });
  });

  it('hides the audit trail link entirely when the run has no trace_id', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...fullRun.data, trace_id: undefined } },
      isLoading: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('activity-detail-trace-id')).toBeNull();
    expect(screen.queryByTestId('activity-detail-open-audit')).toBeNull();
  });

  it('navigates to governance usage drilldown with run filter', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    fireEvent.press(screen.getByTestId('activity-view-full-usage'));
    expect(mockRouterPush).toHaveBeenCalledWith('/governance/usage?run_id=run-1');
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
    mockUseHandoffPackage.mockReturnValue({
      data: {
        run_id: 'run-1',
        reason: 'Escalated to human support',
        conversation_context: 'Customer is upset',
        evidence_count: 2,
        entity_type: 'case',
        entity_id: 'case-1',
        created_at: '2026-04-07T10:00:00Z',
      },
      isLoading: false,
      error: null,
    });
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...fullRun.data, status: 'handed_off', runtime_status: 'escalated' } },
      isLoading: false, error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-handoff-payload')).toBeTruthy();
    expect(screen.getByText('Escalated to human support')).toBeTruthy();
  });

  it('does not render handoff banner when run is not handed_off', () => {
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('activity-detail-handoff-payload')).toBeNull();
  });

  it('shows approval linkage when the run awaits approval', () => {
    mockUseAgentRun.mockReturnValue({
      data: {
        data: {
          ...fullRun.data,
          status: 'awaiting_approval',
          output: { approval_id: 'apr-1', action: 'pending_approval' },
        },
      },
      isLoading: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/activity/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('activity-detail-approval-linkage')).toBeTruthy();
    expect(screen.getByTestId('activity-detail-approval-id')).toBeTruthy();
    fireEvent.press(screen.getByTestId('activity-detail-open-inbox'));
    expect(mockRouterPush).toHaveBeenCalledWith('/inbox');
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
