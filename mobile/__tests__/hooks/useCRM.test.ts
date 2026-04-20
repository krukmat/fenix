import { describe, it, expect, beforeEach, jest } from '@jest/globals';

import {
  queryKeys,
  useAccounts,
  useAccount,
  useContacts,
  useContact,
  useDeals,
  useDeal,
  useLeads,
  useLead,
  useCases,
  useCase,
  useCreateDeal,
  useUpdateDeal,
  useCreateCase,
  useUpdateCase,
  usePipelines,
  usePipelineStages,
  useActivities,
  useEntityTimeline,
  useCreateAccount,
  useUpdateAccount,
  useCreateContact,
  useDeleteDeal,
  useCreateNote,
  useDeleteAttachment,
  useAgentRuns,
  useAgentRun,
  useAgentDefinitions,
} from '../../src/hooks/useCRM';

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
  crmApi: {
    getAccounts: jest.fn(),
    getAccountFull: jest.fn(),
    getContacts: jest.fn(),
    getContact: jest.fn(),
    getDeals: jest.fn(),
    getDealFull: jest.fn(),
    getLeads: jest.fn(),
    getLead: jest.fn(),
    getCases: jest.fn(),
    getCaseFull: jest.fn(),
    createDeal: jest.fn(),
    updateDeal: jest.fn(),
    deleteDeal: jest.fn(),
    createCase: jest.fn(),
    updateCase: jest.fn(),
    getPipelines: jest.fn(),
    getPipelineStages: jest.fn(),
    getActivities: jest.fn(),
    getTimelineByEntity: jest.fn(),
    createAccount: jest.fn(),
    updateAccount: jest.fn(),
    createContact: jest.fn(),
    createNote: jest.fn(),
    deleteAttachment: jest.fn(),
  },
  agentApi: {
    getRuns: jest.fn(),
    getRun: jest.fn(),
    getDefinitions: jest.fn(),
  },
}));

describe('useCRM hooks', () => {
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

  describe('queryKeys', () => {
    it('should build keys consistently', () => {
      expect(queryKeys.accounts('ws')).toEqual(['accounts', 'ws']);
      expect(queryKeys.account('ws', 'id-1')).toEqual(['account', 'ws', 'id-1']);
      expect(queryKeys.contacts('ws')).toEqual(['contacts', 'ws']);
      expect(queryKeys.contact('ws', 'id-2')).toEqual(['contact', 'ws', 'id-2']);
      expect(queryKeys.deals('ws')).toEqual(['deals', 'ws']);
      expect(queryKeys.deal('ws', 'id-3')).toEqual(['deal', 'ws', 'id-3']);
      expect(queryKeys.leads('ws')).toEqual(['leads', 'ws']);
      expect(queryKeys.lead('ws', 'id-lead')).toEqual(['lead', 'ws', 'id-lead']);
      expect(queryKeys.cases('ws')).toEqual(['cases', 'ws']);
      expect(queryKeys.case('ws', 'id-4')).toEqual(['case', 'ws', 'id-4']);
      expect(queryKeys.pipelines('ws')).toEqual(['pipelines', 'ws']);
      expect(queryKeys.pipelineStages('ws', 'pipe-1')).toEqual(['pipeline-stages', 'ws', 'pipe-1']);
      expect(queryKeys.activities('ws')).toEqual(['activities', 'ws']);
      expect(queryKeys.entityTimeline('ws', 'case', 'case-1')).toEqual(['timeline', 'ws', 'case', 'case-1']);
      expect(queryKeys.agentRuns('ws')).toEqual(['agent-runs', 'ws']);
      expect(queryKeys.agentRun('ws', 'id-5')).toEqual(['agent-run', 'ws', 'id-5']);
      expect(queryKeys.agentDefinitions('ws')).toEqual(['agent-definitions', 'ws']);
    });
  });

  it('useAccounts should configure list query', () => {
    useAccounts();
    expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ['accounts', 'ws-1'],
        staleTime: 30000,
        gcTime: 300000,
        enabled: true,
      })
    );
  });

  it('useAccount should configure detail query', () => {
    useAccount('a-1');
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ['account', 'ws-1', 'a-1'],
        staleTime: 60000,
        enabled: true,
      })
    );
  });

  it('useContacts/useContact should configure contact queries', () => {
    useContacts();
    useContact('c-1');

    expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['contacts', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['contact', 'ws-1', 'c-1'], enabled: true })
    );
  });

  it('useDeals/useDeal should configure deal queries', () => {
    useDeals();
    useDeal('d-1');

    expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['deals', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['deal', 'ws-1', 'd-1'], enabled: true })
    );
  });

  it('useLeads/useLead should configure lead queries', () => {
    useLeads();
    useLead('l-1');

    expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['leads', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['lead', 'ws-1', 'l-1'], enabled: true })
    );
  });

  it('useCases/useCase should configure case queries', () => {
    useCases();
    useCase('k-1');

    expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['cases', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['case', 'ws-1', 'k-1'], enabled: true })
    );
  });

  it('useAgentRuns/useAgentRun should configure agent queries', () => {
    useAgentRuns();
    useAgentRun('r-1');

    // useAgentRuns uses useInfiniteQuery (paginates like CRM list hooks)
    expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['agent-runs', 'ws-1'], staleTime: 15000, enabled: true })
    );
    // useAgentRun uses useQuery (single record)
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['agent-run', 'ws-1', 'r-1'], staleTime: 15000, enabled: true })
    );
  });

  it('useAgentDefinitions should configure query with workspace isolation', () => {
    useAgentDefinitions();
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ['agent-definitions', 'ws-1'],
        staleTime: 300000,
        enabled: true,
      })
    );
  });

  it('new CRM query hooks should use workspace-isolated keys', () => {
    usePipelines();
    usePipelineStages('pipe-1');
    useActivities();
    useEntityTimeline('case', 'case-1');

    expect(mockUseInfiniteQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['pipelines', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['pipeline-stages', 'ws-1', 'pipe-1'], enabled: true })
    );
    expect(mockUseInfiniteQuery).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ queryKey: ['activities', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ queryKey: ['timeline', 'ws-1', 'case', 'case-1'], enabled: true })
    );
  });

  it('should disable queries when workspaceId is missing', () => {
    mockUseAuthStore.mockImplementation((selector: unknown) =>
      (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
    );

    useAccounts();
    useAccount('a-1');

    expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['accounts', ''], enabled: false })
    );
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['account', '', 'a-1'], enabled: false })
    );
  });

  it('useCreateDeal should invalidate deals list on success', () => {
    useCreateDeal();
    const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
    options.onSuccess?.();

    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['deals', 'ws-1'] });
  });

  it('useUpdateDeal should invalidate deals list and deal detail on success', () => {
    useUpdateDeal();
    const options = mockUseMutation.mock.calls[0][0] as {
      onSuccess?: (_result: unknown, vars: { id: string }) => void;
    };
    options.onSuccess?.({}, { id: 'd-1' });

    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['deals', 'ws-1'] });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['deal', 'ws-1', 'd-1'] });
  });

  it('useCreateCase should invalidate cases list on success', () => {
    useCreateCase();
    const options = mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void };
    options.onSuccess?.();

    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['cases', 'ws-1'] });
  });

  it('useUpdateCase should invalidate cases list and case detail on success', () => {
    useUpdateCase();
    const options = mockUseMutation.mock.calls[0][0] as {
      onSuccess?: (_result: unknown, vars: { id: string }) => void;
    };
    options.onSuccess?.({}, { id: 'c-1' });

    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['cases', 'ws-1'] });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['case', 'ws-1', 'c-1'] });
  });

  it('new CRM mutations should invalidate affected query keys', () => {
    useCreateAccount();
    (mockUseMutation.mock.calls[0][0] as { onSuccess?: () => void }).onSuccess?.();
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['accounts', 'ws-1'] });

    jest.clearAllMocks();
    useUpdateAccount();
    (mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, vars: { id: string }) => void })
      .onSuccess?.({}, { id: 'acc-1' });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['accounts', 'ws-1'] });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['account', 'ws-1', 'acc-1'] });

    jest.clearAllMocks();
    useCreateContact();
    (mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, data: { accountId?: string }) => void })
      .onSuccess?.({}, { accountId: 'acc-1' });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['contacts', 'ws-1'] });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['account-contacts', 'ws-1', 'acc-1'] });

    jest.clearAllMocks();
    useDeleteDeal();
    (mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, id: string) => void })
      .onSuccess?.({}, 'deal-1');
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['deals', 'ws-1'] });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['deal', 'ws-1', 'deal-1'] });
  });

  it('entity child mutations should invalidate entity timeline where available', () => {
    useCreateNote();
    (mockUseMutation.mock.calls[0][0] as {
      onSuccess?: (_result: unknown, data: { entityType: string; entityId: string }) => void;
    }).onSuccess?.({}, { entityType: 'case', entityId: 'case-1' });

    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['notes', 'ws-1'] });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['timeline', 'ws-1', 'case', 'case-1'] });

    jest.clearAllMocks();
    useDeleteAttachment();
    (mockUseMutation.mock.calls[0][0] as { onSuccess?: (_result: unknown, id: string) => void })
      .onSuccess?.({}, 'att-1');
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['attachments', 'ws-1'] });
    expect(mockInvalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['attachment', 'ws-1', 'att-1'] });
  });
});
