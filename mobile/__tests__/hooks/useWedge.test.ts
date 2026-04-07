// W1-T6 (mobile_wedge_harmonization_plan): tests for useWedge hook layer
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import {
  useInbox,
  useApproveApproval,
  useRejectApproval,
  useSalesBrief,
  useRunUsage,
  useAgentRuns,
  useTriggerSupportAgent,
  useGovernanceSummary,
  wedgeQueryKeys,
} from '../../src/hooks/useWedge';

const mockUseQuery = jest.fn();
const mockUseMutation = jest.fn();
const mockUseQueryClient = jest.fn();
const mockUseAuthStore = jest.fn();
const mockInvalidateQueries = jest.fn();

jest.mock('@tanstack/react-query', () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: (...args: unknown[]) => mockUseMutation(...args),
  useQueryClient: () => mockUseQueryClient(),
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: (...args: unknown[]) => mockUseAuthStore(...args),
}));

jest.mock('../../src/services/api', () => ({
  inboxApi: { getInbox: jest.fn() },
  approvalApi: {
    approve: jest.fn(),
    reject: jest.fn(),
    getPendingApprovals: jest.fn(),
    decideApproval: jest.fn(),
  },
  salesBriefApi: { getSalesBrief: jest.fn() },
  agentApi: {
    getRuns: jest.fn(),
    triggerSupportRun: jest.fn(),
    getRunUsage: jest.fn(),
  },
  governanceApi: { getSummary: jest.fn() },
}));

const WORKSPACE_ID = 'ws-test-123';

beforeEach(() => {
  jest.clearAllMocks();
  mockUseAuthStore.mockImplementation((selector: (s: { workspaceId: string }) => string) =>
    selector({ workspaceId: WORKSPACE_ID })
  );
  mockUseQueryClient.mockReturnValue({ invalidateQueries: mockInvalidateQueries });
});

describe('wedgeQueryKeys', () => {
  it('inbox key contains workspaceId', () => {
    expect(wedgeQueryKeys.inbox('ws-1')).toEqual(['inbox', 'ws-1']);
  });
  it('salesBrief key contains entityType and entityId', () => {
    expect(wedgeQueryKeys.salesBrief('account', 'acc-1')).toEqual(['sales-brief', 'account', 'acc-1']);
  });
  it('runUsage key contains runId', () => {
    expect(wedgeQueryKeys.runUsage('run-1')).toEqual(['run-usage', 'run-1']);
  });
  it('governanceSummary key contains workspaceId', () => {
    expect(wedgeQueryKeys.governanceSummary('ws-1')).toEqual(['governance-summary', 'ws-1']);
  });
});

describe('useInbox', () => {
  it('calls useQuery with inbox queryKey and enabled=true', () => {
    mockUseQuery.mockReturnValue({ data: null, isLoading: false });
    useInbox();
    const [opts] = mockUseQuery.mock.calls[0] as [{ queryKey: unknown[]; enabled: boolean }];
    expect(opts.queryKey).toEqual(wedgeQueryKeys.inbox(WORKSPACE_ID));
    expect(opts.enabled).toBe(true);
  });

  it('disabled when workspaceId is null', () => {
    mockUseAuthStore.mockImplementation((selector: (s: { workspaceId: null }) => null) =>
      selector({ workspaceId: null })
    );
    mockUseQuery.mockReturnValue({ data: null });
    useInbox();
    const [opts] = mockUseQuery.mock.calls[0] as [{ enabled: boolean }];
    expect(opts.enabled).toBe(false);
  });
});

describe('useApproveApproval', () => {
  it('calls useMutation and invalidates inbox + pending-approvals on success', () => {
    mockUseMutation.mockReturnValue({ mutate: jest.fn() });
    useApproveApproval();
    const [opts] = mockUseMutation.mock.calls[0] as [{ onSuccess: () => void }];
    opts.onSuccess();
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: wedgeQueryKeys.inbox(WORKSPACE_ID) });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['pending-approvals', WORKSPACE_ID] });
  });
});

describe('useRejectApproval', () => {
  it('invalidates inbox and pending-approvals on success', () => {
    mockUseMutation.mockReturnValue({ mutate: jest.fn() });
    useRejectApproval();
    const [opts] = mockUseMutation.mock.calls[0] as [{ onSuccess: () => void }];
    opts.onSuccess();
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: wedgeQueryKeys.inbox(WORKSPACE_ID) });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['pending-approvals', WORKSPACE_ID] });
  });
});

describe('useSalesBrief', () => {
  it('calls useQuery with salesBrief queryKey and enabled=true', () => {
    mockUseQuery.mockReturnValue({ data: null });
    useSalesBrief('account', 'acc-1');
    const [opts] = mockUseQuery.mock.calls[0] as [{ queryKey: unknown[]; enabled: boolean }];
    expect(opts.queryKey).toEqual(wedgeQueryKeys.salesBrief('account', 'acc-1'));
    expect(opts.enabled).toBe(true);
  });

  it('disabled when enabled=false is passed', () => {
    mockUseQuery.mockReturnValue({ data: null });
    useSalesBrief('account', 'acc-1', false);
    const [opts] = mockUseQuery.mock.calls[0] as [{ enabled: boolean }];
    expect(opts.enabled).toBe(false);
  });
});

describe('useRunUsage', () => {
  it('calls useQuery with runUsage queryKey', () => {
    mockUseQuery.mockReturnValue({ data: [] });
    useRunUsage('run-42');
    const [opts] = mockUseQuery.mock.calls[0] as [{ queryKey: unknown[]; enabled: boolean }];
    expect(opts.queryKey).toEqual(wedgeQueryKeys.runUsage('run-42'));
    expect(opts.enabled).toBe(true);
  });

  it('disabled when runId is undefined', () => {
    mockUseQuery.mockReturnValue({ data: [] });
    useRunUsage(undefined);
    const [opts] = mockUseQuery.mock.calls[0] as [{ enabled: boolean }];
    expect(opts.enabled).toBe(false);
  });
});

describe('useAgentRuns', () => {
  it('calls useQuery with agentRuns queryKey and workspace filter', () => {
    mockUseQuery.mockReturnValue({ data: null });
    useAgentRuns({ status: 'completed' });
    const [opts] = mockUseQuery.mock.calls[0] as [{ queryKey: unknown[]; enabled: boolean }];
    expect(opts.queryKey).toEqual(wedgeQueryKeys.agentRuns(WORKSPACE_ID, { status: 'completed' }));
    expect(opts.enabled).toBe(true);
  });
});

describe('useTriggerSupportAgent', () => {
  it('invalidates agent runs on success', () => {
    mockUseMutation.mockReturnValue({ mutate: jest.fn() });
    useTriggerSupportAgent();
    const [opts] = mockUseMutation.mock.calls[0] as [{ onSuccess: () => void }];
    opts.onSuccess();
    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: wedgeQueryKeys.agentRuns(WORKSPACE_ID),
    });
  });
});

describe('useGovernanceSummary', () => {
  it('calls useQuery with governanceSummary queryKey', () => {
    mockUseQuery.mockReturnValue({ data: null });
    useGovernanceSummary();
    const [opts] = mockUseQuery.mock.calls[0] as [{ queryKey: unknown[]; enabled: boolean }];
    expect(opts.queryKey).toEqual(wedgeQueryKeys.governanceSummary(WORKSPACE_ID));
    expect(opts.enabled).toBe(true);
  });
});
