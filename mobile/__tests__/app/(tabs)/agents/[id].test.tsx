import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { render, screen } from '@testing-library/react-native';
import AgentsDetailScreen from '../../../../app/(tabs)/agents/[id]';

const mockUseAgentRun = jest.fn();
const mockUseHandoffPackage = jest.fn();

jest.mock('../../../../src/hooks/useAgentSpec', () => ({
  useHandoffPackage: (...args: unknown[]) => mockUseHandoffPackage(...args),
}));

jest.mock('../../../../src/hooks/useCRM', () => ({
  useAgentRun: (...args: unknown[]) => mockUseAgentRun(...args),
}));

const mockRouterPush = jest.fn();
jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockRouterPush }),
  useLocalSearchParams: () => ({ id: 'run-1' }),
  Stack: {
    Screen: () => null,
  },
}));

interface AgentRun {
  id: string;
  agent_id: string;
  agent_name: string;
  status: 'running' | 'success' | 'failed' | 'abstained' | 'partial' | 'escalated' | 'accepted' | 'rejected' | 'delegated';
  triggered_by: string;
  trigger_type: 'manual' | 'event' | 'schedule';
  inputs: Record<string, unknown>;
  evidence_retrieved: Array<{ source_id: string; score: number; snippet: string }>;
  reasoning_trace: string[];
  tool_calls: Array<{
    tool_name: string;
    params: Record<string, unknown>;
    result: Record<string, unknown>;
    latency_ms: number;
  }>;
  output?: string;
  audit_events: Array<{
    actor_id: string;
    action: string;
    timestamp: string;
    outcome: 'success' | 'denied' | 'error';
  }>;
  created_at: string;
  started_at: string;
  completed_at?: string;
  latency_ms: number;
  cost_euros: number;
  handoff_status?: string;
  rejection_reason?: string;
}

describe('Agent Run Detail Screen', () => {
  const baseRun: AgentRun = {
    id: 'run-1',
    agent_id: 'agent-support-001',
    agent_name: 'Support Agent',
    status: 'success',
    triggered_by: 'user-123',
    trigger_type: 'manual',
    inputs: { query: 'How do I reset my password?' },
    evidence_retrieved: [
      { source_id: 'kb-001', score: 0.95, snippet: 'To reset your password...' },
      { source_id: 'kb-002', score: 0.88, snippet: 'Password recovery steps...' },
    ],
    reasoning_trace: [
      'User asked about password reset',
      'Retrieved KB articles about password recovery',
      'Synthesized response from retrieved information',
    ],
    tool_calls: [],
    output: 'You can reset your password by clicking the "Forgot Password" link on the login page.',
    audit_events: [
      { actor_id: 'agent-support-001', action: 'retrieval', timestamp: '2026-02-15T10:00:00Z', outcome: 'success' },
      { actor_id: 'agent-support-001', action: 'generation', timestamp: '2026-02-15T10:00:01Z', outcome: 'success' },
    ],
    created_at: '2026-02-15T10:00:00Z',
    started_at: '2026-02-15T10:00:00Z',
    completed_at: '2026-02-15T10:00:02Z',
    latency_ms: 2000,
    cost_euros: 0.02,
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseHandoffPackage.mockReturnValue({ data: null, isLoading: false });
  });

  it('renders agent run details', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: baseRun },
      isLoading: false,
      error: null,
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText('Support Agent')).toBeDefined();
    expect(screen.getByText('2.0s')).toBeDefined();
    expect(screen.getByText(/0\.0200/)).toBeDefined();
  });

  it('shows loading state', () => {
    mockUseAgentRun.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText('Loading agent run...')).toBeDefined();
  });

  it('shows error state when run is not found', () => {
    mockUseAgentRun.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Agent run not found'),
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText('Agent run not found')).toBeDefined();
  });

  it('renders evidence, reasoning, output and audit sections', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: baseRun },
      isLoading: false,
      error: null,
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText(/Evidence Retrieved/i)).toBeDefined();
    expect(screen.getByText(/Reasoning Trace/i)).toBeDefined();
    expect(screen.getByText(/Output/i)).toBeDefined();
    expect(screen.getByText(/Audit Events/i)).toBeDefined();
    expect(screen.getByText(/To reset your password/i)).toBeDefined();
  });

  it('renders accepted state correctly', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...baseRun, status: 'accepted' } },
      isLoading: false,
      error: null,
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText('Accepted')).toBeDefined();
  });

  it('shows rejection reason for rejected runs', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...baseRun, status: 'rejected', rejection_reason: 'Policy threshold not met' } },
      isLoading: false,
      error: null,
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText('Rejected')).toBeDefined();
    expect(screen.getByText(/Rejection Reason/i)).toBeDefined();
    expect(screen.getByText('Policy threshold not met')).toBeDefined();
  });

  it('renders delegated state without handoff banner', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...baseRun, status: 'delegated' } },
      isLoading: false,
      error: null,
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText('Delegated')).toBeDefined();
    expect(screen.getByText('Delegated to another agent. This is not a human handoff.')).toBeDefined();
    expect(screen.queryByTestId('agent-run-handoff-banner')).toBeNull();
  });

  it('shows handoff banner only when status is escalated', () => {
    mockUseAgentRun.mockReturnValue({
      data: { data: { ...baseRun, status: 'escalated', handoff_status: 'escalated' } },
      isLoading: false,
      error: null,
    });

    render(<AgentsDetailScreen />);

    expect(screen.getByText(/Escalated/i)).toBeDefined();
  });
});
