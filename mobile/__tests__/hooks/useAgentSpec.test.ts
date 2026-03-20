import { beforeEach, describe, expect, it, jest } from '@jest/globals';

import {
  agentSpecQueryKeys,
  useActivateWorkflow,
  useAgentRunsByEntity,
  useAgentRunsByWorkflow,
  useCreateWorkflow,
  useDecideApproval,
  useDismissSignal,
  useExecuteWorkflow,
  useHandoffPackage,
  useNewVersion,
  usePendingApprovals,
  useRollback,
  useSignals,
  useSignalsByEntity,
  useUpdateWorkflow,
  useWorkflow,
  useWorkflowVersions,
  useWorkflows,
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
    getVersions: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    activateWorkflow: jest.fn(),
    newVersion: jest.fn(),
    rollback: jest.fn(),
    executeWorkflow: jest.fn(),
  },
  approvalApi: {
    getPendingApprovals: jest.fn(),
    decideApproval: jest.fn(),
  },
  agentApi: {
    getRunsByEntity: jest.fn(),
    getRunsByWorkflow: jest.fn(),
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

  describe('agentSpecQueryKeys', () => {
    it('keeps existing signal and workflow keys stable', () => {
      expect(agentSpecQueryKeys.signals('ws')).toEqual(['signals', 'ws', {}]);
      expect(agentSpecQueryKeys.signalsByEntity('ws', 'deal', 'd-1')).toEqual([
        'signals',
        'ws',
        { entity_type: 'deal', entity_id: 'd-1' },
      ]);
      expect(agentSpecQueryKeys.workflows('ws')).toEqual(['workflows', 'ws', {}]);
      expect(agentSpecQueryKeys.workflow('ws', 'wf-1')).toEqual(['workflow', 'ws', 'wf-1']);
    });

    it('adds workflow versions and filtered run keys', () => {
      expect(agentSpecQueryKeys.workflowVersions('ws', 'wf-1')).toEqual(['workflow-versions', 'ws', 'wf-1']);
      expect(agentSpecQueryKeys.agentRunsByEntity('ws', 'case', 'c-1', { status: 'rejected' })).toEqual([
        'agent-runs',
        'ws',
        'entity',
        'case',
        'c-1',
        { status: 'rejected' },
      ]);
      expect(agentSpecQueryKeys.agentRunsByWorkflow('ws', 'wf-1', { entity_type: 'deal' })).toEqual([
        'agent-runs',
        'ws',
        'workflow',
        'wf-1',
        { entity_type: 'deal' },
      ]);
    });
  });

  describe('signals hooks', () => {
    it('configures signal list and entity queries', () => {
      useSignals({ status: 'active' });
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: ['signals', 'ws-1', { status: 'active' }], enabled: true })
      );

      useSignalsByEntity('deal', 'd-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['signals', 'ws-1', { entity_type: 'deal', entity_id: 'd-1' }],
          enabled: true,
        })
      );
    });

    it('invalidates workspace signal queries after dismiss', () => {
      useDismissSignal();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['signals', 'ws-1'] });
    });
  });

  describe('workflow queries', () => {
    it('configures workflow list, detail and versions queries', () => {
      useWorkflows({ status: 'active' });
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: ['workflows', 'ws-1', { status: 'active' }], enabled: true })
      );

      useWorkflow('wf-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: ['workflow', 'ws-1', 'wf-1'], enabled: true })
      );

      useWorkflowVersions('wf-1');
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: ['workflow-versions', 'ws-1', 'wf-1'], enabled: true })
      );
    });

    it('disables workflow version query when id is empty', () => {
      useWorkflowVersions('');
      expect(mockUseQuery).toHaveBeenCalledWith(expect.objectContaining({ enabled: false }));
    });
  });

  describe('workflow mutations', () => {
    it('useCreateWorkflow invalidates list, detail and versions for the created workflow', () => {
      useCreateWorkflow();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: (result: { id: string }) => void };
      options.onSuccess?.({ id: 'wf-2' });

      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['workflows', 'ws-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['workflow', 'ws-1', 'wf-2'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(3, { queryKey: ['workflow-versions', 'ws-1', 'wf-2'] });
    });

    it('useUpdateWorkflow invalidates list, detail and versions', () => {
      useUpdateWorkflow();
      const options = mockUseMutation.mock.calls[0][0] as {
        onSuccess?: (_result: unknown, variables: { id: string }) => void;
      };
      options.onSuccess?.({}, { id: 'wf-1' });

      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['workflows', 'ws-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['workflow', 'ws-1', 'wf-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(3, { queryKey: ['workflow-versions', 'ws-1', 'wf-1'] });
    });

    it('useActivateWorkflow invalidates list, detail and versions', () => {
      useActivateWorkflow();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, id: string) => void };
      options.onSuccess?.({}, 'wf-1');

      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['workflows', 'ws-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['workflow', 'ws-1', 'wf-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(3, { queryKey: ['workflow-versions', 'ws-1', 'wf-1'] });
    });

    it('useNewVersion invalidates list, detail and versions of the source workflow', () => {
      useNewVersion();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, id: string) => void };
      options.onSuccess?.({}, 'wf-1');

      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['workflows', 'ws-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['workflow', 'ws-1', 'wf-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(3, { queryKey: ['workflow-versions', 'ws-1', 'wf-1'] });
    });

    it('useRollback invalidates list, detail, versions and agent runs', () => {
      useRollback();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, id: string) => void };
      options.onSuccess?.({}, 'wf-1');

      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['workflows', 'ws-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['workflow', 'ws-1', 'wf-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(3, { queryKey: ['workflow-versions', 'ws-1', 'wf-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(4, { queryKey: ['agent-runs', 'ws-1'] });
    });

    it('useExecuteWorkflow invalidates workflow detail, versions and agent runs', () => {
      useExecuteWorkflow();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, id: string) => void };
      options.onSuccess?.({}, 'wf-1');

      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['workflow', 'ws-1', 'wf-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['workflow-versions', 'ws-1', 'wf-1'] });
      expect(mockInvalidateQueries).toHaveBeenNthCalledWith(3, { queryKey: ['agent-runs', 'ws-1'] });
    });
  });

  describe('filtered agent run queries', () => {
    it('configures entity-scoped run query with workspace-safe key', () => {
      mockUseInfiniteQuery.mockReturnValueOnce({ data: { pages: [{ data: [], meta: { total: 0 } }] } });
      useAgentRunsByEntity('case', 'c-1', { status: 'rejected' });

      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['agent-runs', 'ws-1', 'entity', 'case', 'c-1', { status: 'rejected' }],
          enabled: true,
          staleTime: 15_000,
        })
      );
    });

    it('configures workflow-scoped run query with workspace-safe key', () => {
      mockUseInfiniteQuery.mockReturnValueOnce({ data: { pages: [{ data: [], meta: { total: 0 } }] } });
      useAgentRunsByWorkflow('wf-1', { entity_type: 'deal' });

      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['agent-runs', 'ws-1', 'workflow', 'wf-1', { entity_type: 'deal' }],
          enabled: true,
          staleTime: 15_000,
        })
      );
    });

    it('disables entity-scoped run query when entity id is empty', () => {
      useAgentRunsByEntity('case', '');
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(expect.objectContaining({ enabled: false }));
    });

    it('disables workflow-scoped run query when workflow id is empty', () => {
      useAgentRunsByWorkflow('');
      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(expect.objectContaining({ enabled: false }));
    });
  });

  describe('approvals and handoff', () => {
    it('keeps pending approvals invalidation stable', () => {
      usePendingApprovals();
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: ['pending-approvals', 'ws-1'], enabled: true })
      );

      useDecideApproval();
      const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
      options.onSuccess?.();
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['pending-approvals', 'ws-1'] });
    });

    it('keeps handoff query behavior stable', () => {
      expect(agentSpecQueryKeys.handoffPackage('run-1')).toEqual(['handoff-package', 'run-1']);
      useHandoffPackage('run-1', true);
      expect(mockUseQuery).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: ['handoff-package', 'run-1'], enabled: true, staleTime: 60_000 })
      );
    });
  });
});
