import { describe, it, expect, beforeEach, jest } from '@jest/globals';

import {
  queryKeys,
  useAccounts,
  useAccount,
  useContacts,
  useContact,
  useDeals,
  useDeal,
  useCases,
  useCase,
  useAgentRuns,
  useAgentRun,
} from '../../src/hooks/useCRM';

const mockUseQuery = jest.fn();
const mockUseAuthStore = jest.fn();

jest.mock('@tanstack/react-query', () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
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
    getCases: jest.fn(),
    getCaseFull: jest.fn(),
  },
  agentApi: {
    getRuns: jest.fn(),
    getRun: jest.fn(),
  },
}));

describe('useCRM hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseQuery.mockReturnValue({ data: [] });
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
      expect(queryKeys.cases('ws')).toEqual(['cases', 'ws']);
      expect(queryKeys.case('ws', 'id-4')).toEqual(['case', 'ws', 'id-4']);
      expect(queryKeys.agentRuns('ws')).toEqual(['agent-runs', 'ws']);
      expect(queryKeys.agentRun('ws', 'id-5')).toEqual(['agent-run', 'ws', 'id-5']);
    });
  });

  it('useAccounts should configure list query', () => {
    useAccounts();
    expect(mockUseQuery).toHaveBeenCalledWith(
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

    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['contacts', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ queryKey: ['contact', 'ws-1', 'c-1'], enabled: true })
    );
  });

  it('useDeals/useDeal should configure deal queries', () => {
    useDeals();
    useDeal('d-1');

    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['deals', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ queryKey: ['deal', 'ws-1', 'd-1'], enabled: true })
    );
  });

  it('useCases/useCase should configure case queries', () => {
    useCases();
    useCase('k-1');

    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['cases', 'ws-1'], enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ queryKey: ['case', 'ws-1', 'k-1'], enabled: true })
    );
  });

  it('useAgentRuns/useAgentRun should configure agent queries', () => {
    useAgentRuns();
    useAgentRun('r-1');

    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['agent-runs', 'ws-1'], staleTime: 15000, enabled: true })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ queryKey: ['agent-run', 'ws-1', 'r-1'], staleTime: 15000, enabled: true })
    );
  });

  it('should disable queries when workspaceId is missing', () => {
    mockUseAuthStore.mockImplementation((selector: unknown) =>
      (selector as (state: { workspaceId: string | null }) => unknown)({ workspaceId: null })
    );

    useAccounts();
    useAccount('a-1');

    expect(mockUseQuery).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ queryKey: ['accounts', ''], enabled: false })
    );
    expect(mockUseQuery).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ queryKey: ['account', '', 'a-1'], enabled: false })
    );
  });
});
