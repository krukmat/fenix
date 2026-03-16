import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { render, screen } from '@testing-library/react-native';
import AgentsListScreen from '../../../../app/(tabs)/agents/index';

const mockUseAgentRuns = jest.fn();
const mockUseAgentDefinitions = jest.fn();

jest.mock('../../../../src/hooks/useCRM', () => ({
  useAgentRuns: (...args: unknown[]) => mockUseAgentRuns(...args),
  useAgentDefinitions: () => mockUseAgentDefinitions(),
}));

jest.mock('../../../../src/components/agents/TriggerAgentButton', () => {
  const React = require('react');
  return {
    __esModule: true,
    default: () => React.createElement('View', { testID: 'trigger-agent-button' }),
  };
});

interface AgentRun {
  id: string;
  agent_name: string;
  status: 'running' | 'success' | 'failed' | 'abstained' | 'escalated' | 'accepted' | 'rejected' | 'delegated';
  started_at: string;
  latency_ms: number;
  cost_euros: number;
  rejection_reason?: string;
}

describe('Agent Runs List Screen', () => {
  const mockRuns: AgentRun[] = [
    {
      id: 'run-1',
      agent_name: 'Support Agent',
      status: 'success',
      started_at: '2026-02-15T10:00:00Z',
      latency_ms: 2500,
      cost_euros: 0.02,
    },
    {
      id: 'run-2',
      agent_name: 'Prospecting Agent',
      status: 'failed',
      started_at: '2026-02-15T11:30:00Z',
      latency_ms: 5000,
      cost_euros: 0.05,
    },
    {
      id: 'run-3',
      agent_name: 'Support Agent',
      status: 'running',
      started_at: '2026-02-15T12:00:00Z',
      latency_ms: 1000,
      cost_euros: 0.01,
    },
    {
      id: 'run-4',
      agent_name: 'Workflow Agent',
      status: 'accepted',
      started_at: '2026-02-15T13:00:00Z',
      latency_ms: 1800,
      cost_euros: 0.03,
    },
    {
      id: 'run-5',
      agent_name: 'Policy Agent',
      status: 'rejected',
      started_at: '2026-02-15T14:00:00Z',
      latency_ms: 900,
      cost_euros: 0.01,
      rejection_reason: 'Policy threshold not met',
    },
    {
      id: 'run-6',
      agent_name: 'Coordinator Agent',
      status: 'delegated',
      started_at: '2026-02-15T15:00:00Z',
      latency_ms: 1200,
      cost_euros: 0.02,
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders agent runs including accepted, rejected and delegated states', () => {
    mockUseAgentRuns.mockReturnValue({
      data: { pages: [{ data: mockRuns }] },
      isLoading: false,
      isFetching: false,
      error: null,
      refetch: jest.fn(),
    });

    render(<AgentsListScreen />);

    expect(screen.getByTestId('agent-runs-list')).toBeDefined();
    expect(screen.getAllByText('Support Agent').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('Workflow Agent')).toBeDefined();
    expect(screen.getByText('Policy Agent')).toBeDefined();
    expect(screen.getByText('Coordinator Agent')).toBeDefined();
    expect(screen.getByText('Accepted')).toBeDefined();
    expect(screen.getByText('Rejected')).toBeDefined();
    expect(screen.getByText('Delegated')).toBeDefined();
    expect(screen.getByText('Policy threshold not met')).toBeDefined();
  });

  it('shows loading state', () => {
    mockUseAgentRuns.mockReturnValue({
      data: undefined,
      isLoading: true,
      isFetching: false,
      error: null,
      refetch: jest.fn(),
    });

    render(<AgentsListScreen />);

    expect(screen.getByText('Loading agent runs...')).toBeDefined();
  });

  it('shows error state', () => {
    mockUseAgentRuns.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      error: new Error('Failed to load'),
      refetch: jest.fn(),
    });

    render(<AgentsListScreen />);

    expect(screen.getByText('Failed to load')).toBeDefined();
  });

  it('shows empty state when no runs', () => {
    mockUseAgentRuns.mockReturnValue({
      data: { pages: [{ data: [] }] },
      isLoading: false,
      isFetching: false,
      error: null,
      refetch: jest.fn(),
    });

    render(<AgentsListScreen />);

    expect(screen.getByText('No agent runs found')).toBeDefined();
    expect(screen.getByText('Trigger an agent to get started')).toBeDefined();
  });

  it('renders latency and cost metrics for list items', () => {
    mockUseAgentRuns.mockReturnValue({
      data: { pages: [{ data: mockRuns }] },
      isLoading: false,
      isFetching: false,
      error: null,
      refetch: jest.fn(),
    });

    render(<AgentsListScreen />);

    expect(screen.getByText('2.5s')).toBeDefined();
    expect(screen.getByText('5.0s')).toBeDefined();
    expect(screen.getByText('900ms')).toBeDefined();
    expect(screen.getAllByText(/0\.02/).length).toBeGreaterThanOrEqual(1);
  });
});
