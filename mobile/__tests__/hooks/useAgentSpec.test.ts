import { beforeEach, describe, expect, it, jest } from '@jest/globals';

import {
  agentSpecQueryKeys,
  useAgentRunsByEntity,
  useDecideApproval,
  useDismissSignal,
  useHandoffPackage,
  usePendingApprovals,
  useSignals,
  useSignalsByEntity,
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
  approvalApi: {
    getPendingApprovals: jest.fn(),
    decideApproval: jest.fn(),
  },
  agentApi: {
    getRunsByEntity: jest.fn(),
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
    it('keeps signal and entity keys stable', () => {
      expect(agentSpecQueryKeys.signals('ws')).toEqual(['signals', 'ws', {}]);
      expect(agentSpecQueryKeys.signalsByEntity('ws', 'deal', 'd-1')).toEqual([
        'signals',
        'ws',
        { entity_type: 'deal', entity_id: 'd-1' },
      ]);
    });

    it('keeps agent run entity key stable', () => {
      expect(agentSpecQueryKeys.agentRunsByEntity('ws', 'case', 'c-1', { status: 'failed' })).toEqual([
        'agent-runs',
        'ws',
        'entity',
        'case',
        'c-1',
        { status: 'failed' },
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

  describe('filtered agent run queries', () => {
    it('configures entity-scoped run query with workspace-safe key', () => {
      mockUseInfiniteQuery.mockReturnValueOnce({ data: { pages: [{ data: [], meta: { total: 0 } }] } });
      useAgentRunsByEntity('case', 'c-1', { status: 'failed' });

      expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['agent-runs', 'ws-1', 'entity', 'case', 'c-1', { status: 'failed' }],
          enabled: true,
          staleTime: 15_000,
        })
      );
    });

    it('disables entity-scoped run query when entity id is empty', () => {
      useAgentRunsByEntity('case', '');
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
