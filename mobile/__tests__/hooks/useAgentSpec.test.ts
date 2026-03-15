// useAgentSpec hooks — query keys, enabled guards, staleTime, mutation invalidation
// FR-301 (Signals), FR-302 (Workflows), FR-071 (Approvals), FR-232 (Handoff), UC-A4/A5/A6/A7


import { describe, it, expect, beforeEach, jest } from '@jest/globals';

import {
  agentSpecQueryKeys,
  useSignals,
  useSignalsByEntity,
  useDismissSignal,
  useWorkflows,
  useWorkflow,
  useActivateWorkflow,
  useExecuteWorkflow,
  usePendingApprovals,
  useDecideApproval,
  useHandoffPackage,
} from '../../src/hooks/useAgentSpec';

const mockUseQuery = jest.fn();
const mockUseInfiniteQuery = jest.fn();
const mockUseMutation = jest.fn();
const mockUseQueryClient = jest.fn();
const mockUseAuthStore = jest.fn();
const mockInvalidateQueries = jest.fn();

jest.mock('@tanstack/react-query', () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useInfiniteQuery: (...args: unknown[]) => mockUseInfiniteQuery(...args),
  useMutation: (...args: unknown[]) => mockUseMutation(...args),
  useQueryClient: () => mockUseQueryClient(),
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: (...args: unknown[]) => mockUseAuthStore(...args),
}));

jest.mock('../../src/services/api', () => ({
  signalApi: {
    getSignals: jest.fn(),
    dismissSignal: jest.fn(),
  },
  workflowApi: {
    getWorkflows: jest.fn(),
    getWorkflow: jest.fn(),
    activateWorkflow: jest.fn(),
    executeWorkflow: jest.fn(),
  },
  approvalApi: {
    getPendingApprovals: jest.fn(),
    decideApproval: jest.fn(),
  },
  agentApi: {
    getHandoff: jest.fn(),
  },
}));

describe('useAgentSpec hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseQuery.mockReturnValue({ data: [] });
    mockUseInfiniteQuery.mockReturnValue({ data: { pages: [] } });
    mockUseMutation.mockReturnValue({ mutate: jest.fn() });
    mockUseQueryClient.mockReturnValue({ invalidateQueries: mockInvalidateQueries });
    mockInvalidateQueries.mockReset();
    mockUseAuthStore.mockImplementation((selector: unknown) =>
      (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: 'ws-1' })
    );
  });

  // ─── agentSpecQueryKeys ───────────────────────────────────────────────────

  describe('agentSpecQueryKeys', () => {
    it('signals key includes workspaceId and empty filter object by default', () => {
      expect(agentSpecQueryKeys.signals('ws')).toEqual(['signals', 'ws', {}]);
    });

    it('signals key includes explicit filters when provided', () => {
      expect(agentSpecQueryKeys.signals('ws', { status: 'active' })).toEqual([
        'signals',
        'ws',
        { status: 'active' },
      ]);
    });

    it('signalsByEntity key encodes entity type and id into filter shape', () => {
      expect(agentSpecQueryKeys.signalsByEntity('ws', 'deal', 'd-1')).toEqual([
        'signals',
        'ws',
        { entity_type: 'deal', entity_id: 'd-1' },
      ]);
    });

    it('workflows key includes workspaceId and empty filter object by default', () => {
      expect(agentSpecQueryKeys.workflows('ws')).toEqual(['workflows', 'ws', {}]);
    });

    it('workflows key includes status filter when provided', () => {
      expect(agentSpecQueryKeys.workflows('ws', { status: 'active' })).toEqual([
        'workflows',
        'ws',
        { status: 'active' },
      ]);
    });

    it('workflow detail key includes workspaceId and id', () => {
      expect(agentSpecQueryKeys.workflow('ws', 'wf-1')).toEqual(['workflow', 'ws', 'wf-1']);
    });

    it('pendingApprovals key includes workspaceId', () => {
      expect(agentSpecQueryKeys.pendingApprovals('ws')).toEqual(['pending-approvals', 'ws']);
    });
  });

  // ─── Signals ─────────────────────────────────────────────────────────────

  describe('useSignals', () => {
    it('configures infinite query with correct key and staleTime', () => {
      useSignals();
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['signals', 'ws-1', {}],
          staleTime: 15_000,
          enabled: true,
        })
      );
    });

    it('passes status filter into query key', () => {
      useSignals({ status: 'active' });
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['signals', 'ws-1', { status: 'active' }],
        })
      );
    });

    it('passes partial entity filter into query key', () => {
      useSignals({ entity_type: 'account' });
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['signals', 'ws-1', { entity_type: 'account' }],
        })
      );
    });

    it('is disabled when workspaceId is null', () => {
      mockUseAuthStore.mockImplementation((selector: unknown) =>
        (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
      );
      useSignals();
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });
  });

  describe('useSignalsByEntity', () => {
    it('configures query with entity-scoped key', () => {
      useSignalsByEntity('deal', 'd-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['signals', 'ws-1', { entity_type: 'deal', entity_id: 'd-1' }],
          staleTime: 15_000,
          enabled: true,
        })
      );
    });

    it('is disabled when entityType is empty', () => {
      useSignalsByEntity('', 'd-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });

    it('is disabled when entityId is empty', () => {
      useSignalsByEntity('deal', '');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });

    it('is disabled when workspaceId is null', () => {
      mockUseAuthStore.mockImplementation((selector: unknown) =>
        (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
      );
      useSignalsByEntity('deal', 'd-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });
  });

  describe('useDismissSignal', () => {
    it('invalidates all signals for the workspace on success', () => {
      useDismissSignal();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['signals', 'ws-1'] });
    });

    it('uses null-safe workspace key when workspaceId is missing', () => {
      mockUseAuthStore.mockImplementation((selector: unknown) =>
        (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
      );
      useDismissSignal();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['signals', ''] });
    });
  });

  // ─── Workflows ───────────────────────────────────────────────────────────

  describe('useWorkflows', () => {
    it('configures infinite query with correct key and staleTime', () => {
      useWorkflows();
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['workflows', 'ws-1', {}],
          staleTime: 60_000,
          enabled: true,
        })
      );
    });

    it('passes status filter into query key', () => {
      useWorkflows({ status: 'active' });
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['workflows', 'ws-1', { status: 'active' }],
        })
      );
    });

    it('is disabled when workspaceId is null', () => {
      mockUseAuthStore.mockImplementation((selector: unknown) =>
        (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
      );
      useWorkflows();
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });
  });

  describe('useWorkflow', () => {
    it('configures query with correct detail key', () => {
      useWorkflow('wf-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['workflow', 'ws-1', 'wf-1'],
          staleTime: 60_000,
          enabled: true,
        })
      );
    });

    it('is disabled when id is empty', () => {
      useWorkflow('');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });

    it('is disabled when workspaceId is null', () => {
      mockUseAuthStore.mockImplementation((selector: unknown) =>
        (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
      );
      useWorkflow('wf-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });
  });

  describe('useActivateWorkflow', () => {
    it('invalidates workflows list and workflow detail on success', () => {
      useActivateWorkflow();
      const options = mockUseMutation.mock.calls[0][0] as {
        onSuccess?: (_result: unknown, id: string) => void;
      };
      options.onSuccess?.({}, 'wf-1');

      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['workflows', 'ws-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['workflow', 'ws-1', 'wf-1'] });
    });
  });

  describe('useExecuteWorkflow', () => {
    it('invalidates agent-runs list on success (execution creates a new run)', () => {
      useExecuteWorkflow();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();

      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['agent-runs', 'ws-1'] });
    });
  });

  // ─── Approvals ───────────────────────────────────────────────────────────

  describe('usePendingApprovals', () => {
    it('configures query with staleTime of 15s for near-realtime badge count', () => {
      usePendingApprovals();
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['pending-approvals', 'ws-1'],
          staleTime: 15_000,
          enabled: true,
        })
      );
    });

    it('is disabled when workspaceId is null', () => {
      mockUseAuthStore.mockImplementation((selector: unknown) =>
        (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
      );
      usePendingApprovals();
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });
  });

  describe('useDecideApproval', () => {
    it('invalidates pending approvals on approve decision', () => {
      useDecideApproval();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();

      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: ['pending-approvals', 'ws-1'],
      });
    });

    it('invalidates pending approvals on deny decision', () => {
      useDecideApproval();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();

      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: ['pending-approvals', 'ws-1'],
      });
    });

    it('uses null-safe workspace key when workspaceId is missing', () => {
      mockUseAuthStore.mockImplementation((selector: unknown) =>
        (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
      );
      useDecideApproval();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();

      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: ['pending-approvals', ''],
      });
    });
  });

  // ─── Handoff ─────────────────────────────────────────────────────────────

  describe('agentSpecQueryKeys.handoffPackage', () => {
    it('includes runId in the query key', () => {
      expect(agentSpecQueryKeys.handoffPackage('run-1')).toEqual(['handoff-package', 'run-1']);
    });
  });

  describe('useHandoffPackage', () => {
    it('is enabled when runId is provided and enabled=true', () => {
      useHandoffPackage('run-1', true);
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['handoff-package', 'run-1'],
          enabled: true,
        })
      );
    });

    it('is disabled when enabled=false (run is not escalated)', () => {
      useHandoffPackage('run-1', false);
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });

    it('is disabled when runId is undefined', () => {
      useHandoffPackage(undefined, true);
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: false })
      );
    });

    it('uses staleTime of 60s', () => {
      useHandoffPackage('run-1', true);
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ staleTime: 60_000 })
      );
    });
  });
});