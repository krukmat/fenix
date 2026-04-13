import React from 'react';
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render } from '@testing-library/react-native';
import { PaperProvider } from 'react-native-paper';
import { AgentActivitySection } from '../../../src/components/agents/AgentActivitySection';

const mockUseAgentRunsByEntity = jest.fn();
const mockPush = jest.fn();

jest.mock('../../../src/hooks/useAgentSpec', () => ({
  useAgentRunsByEntity: (...args: unknown[]) => mockUseAgentRunsByEntity(...args),
}));

jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush }),
}));

describe('AgentActivitySection', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders recent agent runs and navigates to run detail', () => {
    mockUseAgentRunsByEntity.mockReturnValue({
      data: {
        pages: [
          {
            data: [
              { id: 'run-1', agent_name: 'Workflow Agent', status: 'delegated', started_at: '2026-03-16T10:00:00Z', latency_ms: 1200 },
              { id: 'run-2', agent_name: 'Policy Agent', status: 'rejected', started_at: '2026-03-16T10:02:00Z', latency_ms: 900 },
            ],
          },
        ],
      },
      isLoading: false,
    });

    const { getByText, getByTestId } = render(
      <PaperProvider>
        <AgentActivitySection entityType="deal" entityId="deal-1" />
      </PaperProvider>
    );

    expect(getByText('Agent Activity')).toBeTruthy();
    expect(getByText('Workflow Agent')).toBeTruthy();
    expect(getByText('Delegated')).toBeTruthy();

    fireEvent.press(getByTestId('agent-activity-item-run-1'));
    expect(mockPush).toHaveBeenCalledWith('/agents/run-1');
  });

  it('returns null when no activity exists', () => {
    mockUseAgentRunsByEntity.mockReturnValue({
      data: { pages: [{ data: [] }] },
      isLoading: false,
    });

    const { queryByTestId } = render(
      <PaperProvider>
        <AgentActivitySection entityType="account" entityId="acc-1" />
      </PaperProvider>
    );

    expect(queryByTestId('agent-activity-section')).toBeNull();
  });

  it('accepts lead as an entity type and queries by that entity', () => {
    mockUseAgentRunsByEntity.mockReturnValue({
      data: {
        pages: [
          {
            data: [
              { id: 'run-lead-1', agent_name: 'Prospecting Agent', status: 'completed', started_at: '2026-04-13T10:00:00Z' },
            ],
          },
        ],
      },
      isLoading: false,
    });

    const { getByText } = render(
      <PaperProvider>
        <AgentActivitySection entityType="lead" entityId="lead-1" />
      </PaperProvider>
    );

    expect(mockUseAgentRunsByEntity).toHaveBeenCalledWith('lead', 'lead-1');
    expect(getByText('Prospecting Agent')).toBeTruthy();
  });
});
