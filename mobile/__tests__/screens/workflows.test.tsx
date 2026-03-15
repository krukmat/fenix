// Workflows list + detail screens — filter, activate/execute/verify, DSL viewer
// FR-302 (Workflows), UC-A4: workflow list and detail


import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import type { Workflow } from '../../src/services/api';

// ─── Mocks ───────────────────────────────────────────────────────────────────

const mockUseWorkflows = jest.fn();
const mockUseWorkflow = jest.fn();
const mockUseActivateWorkflow = jest.fn();
const mockUseExecuteWorkflow = jest.fn();
const mockPush = jest.fn();

jest.mock('../../src/hooks/useAgentSpec', () => ({
  useWorkflows: (...args: unknown[]) => mockUseWorkflows(...args),
  useWorkflow: (...args: unknown[]) => mockUseWorkflow(...args),
  useActivateWorkflow: () => mockUseActivateWorkflow(),
  useExecuteWorkflow: () => mockUseExecuteWorkflow(),
  useSignals: jest.fn().mockReturnValue({ data: { pages: [] }, isLoading: false, refetch: jest.fn() }),
  usePendingApprovals: jest.fn().mockReturnValue({ data: [], isLoading: false, refetch: jest.fn() }),
  useDismissSignal: jest.fn().mockReturnValue({ mutate: jest.fn() }),
  useDecideApproval: jest.fn().mockReturnValue({ mutate: jest.fn() }),
}));

jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useLocalSearchParams: () => ({ id: 'wf-1' }),
  Stack: {
    Screen: ({ children }: { children?: React.ReactNode }) => children ?? null,
  },
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: (sel: (s: { workspaceId: string }) => unknown) => sel({ workspaceId: 'ws-1' }),
}));

jest.mock('../../src/services/api', () => ({
  workflowApi: { verifyWorkflow: jest.fn() },
}));

// ─── Fixtures ────────────────────────────────────────────────────────────────

const activeWorkflow: Workflow = {
  id: 'wf-1',
  workspace_id: 'ws-1',
  name: 'Lead Nurture',
  dsl_source: 'workflow:\n  name: lead_nurture',
  version: 1,
  status: 'active',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

const draftWorkflow: Workflow = { ...activeWorkflow, id: 'wf-2', status: 'draft' };

function setupListMock(workflows: Workflow[]) {
  mockUseWorkflows.mockReturnValue({
    data: { pages: [workflows] },
    isLoading: false,
    isRefetching: false,
    isFetchingNextPage: false,
    hasNextPage: false,
    fetchNextPage: jest.fn(),
    refetch: jest.fn(),
  });
}

// ─── Workflows list ───────────────────────────────────────────────────────────

describe('WorkflowsListScreen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders workflow cards for each workflow', () => {
    setupListMock([activeWorkflow, draftWorkflow]);
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/index').default;
    const { getByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    expect(getByTestId(`workflow-${activeWorkflow.id}`)).toBeTruthy();
    expect(getByTestId(`workflow-${draftWorkflow.id}`)).toBeTruthy();
  });

  it('shows empty state when no workflows', () => {
    setupListMock([]);
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/index').default;
    const { getByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    expect(getByTestId('workflows-empty')).toBeTruthy();
  });

  it('renders status filter chips', () => {
    setupListMock([]);
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/index').default;
    const { getByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    expect(getByTestId('workflows-chip-all')).toBeTruthy();
    expect(getByTestId('workflows-chip-active')).toBeTruthy();
    expect(getByTestId('workflows-chip-draft')).toBeTruthy();
  });

  it('passes status filter to useWorkflows when chip is selected', () => {
    setupListMock([activeWorkflow]);
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/index').default;
    const { getByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    fireEvent.press(getByTestId('workflows-chip-active'));
    expect(mockUseWorkflows).toHaveBeenCalledWith({ status: 'active' });
  });

  it('navigates to workflow detail on card press', () => {
    setupListMock([activeWorkflow]);
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/index').default;
    const { getByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    fireEvent.press(getByTestId(`workflow-${activeWorkflow.id}`));
    expect(mockPush).toHaveBeenCalledWith(`/workflows/${activeWorkflow.id}`);
  });
});

// ─── Workflow detail ──────────────────────────────────────────────────────────

describe('WorkflowDetailScreen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseActivateWorkflow.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseExecuteWorkflow.mockReturnValue({ mutate: jest.fn(), isPending: false });
  });

  it('renders name, version, status and DSL', () => {
    mockUseWorkflow.mockReturnValue({ data: activeWorkflow, isLoading: false, refetch: jest.fn() });
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/[id]').default;
    const { getByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    expect(getByTestId('workflow-detail-name').props.children).toBe('Lead Nurture');
    expect(getByTestId('workflow-detail-version').props.children).toBe('v1');
    expect(within(getByTestId('workflow-detail-status')).getByText('active')).toBeTruthy();
    expect(getByTestId('workflow-detail-dsl-code').props.children).toContain('lead_nurture');
  });

  it('shows Execute button for active workflow, not Activate', () => {
    mockUseWorkflow.mockReturnValue({ data: activeWorkflow, isLoading: false, refetch: jest.fn() });
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/[id]').default;
    const { getByTestId, queryByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    expect(getByTestId('workflow-execute-btn')).toBeTruthy();
    expect(queryByTestId('workflow-activate-btn')).toBeNull();
  });

  it('shows Activate and Verify buttons for draft workflow, not Execute', () => {
    mockUseWorkflow.mockReturnValue({ data: draftWorkflow, isLoading: false, refetch: jest.fn() });
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/[id]').default;
    const { getByTestId, queryByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    expect(getByTestId('workflow-activate-btn')).toBeTruthy();
    expect(getByTestId('workflow-verify-btn')).toBeTruthy();
    expect(queryByTestId('workflow-execute-btn')).toBeNull();
  });

  it('calls activateMutation on Activate press', () => {
    const mutateFn = jest.fn();
    mockUseActivateWorkflow.mockReturnValue({ mutate: mutateFn, isPending: false });
    mockUseWorkflow.mockReturnValue({ data: draftWorkflow, isLoading: false, refetch: jest.fn() });
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/[id]').default;
    const { getByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    fireEvent.press(getByTestId('workflow-activate-btn'));
    expect(mutateFn).toHaveBeenCalledWith('wf-1', expect.any(Object));
  });

  it('shows loading state when isLoading is true', () => {
    mockUseWorkflow.mockReturnValue({ data: null, isLoading: true, refetch: jest.fn() });
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Screen = require('../../app/(tabs)/workflows/[id]').default;
    const { queryByTestId } = render(
      <PaperProvider>
        <Screen />
      </PaperProvider>
    );
    expect(queryByTestId('workflow-detail')).toBeNull();
  });
});