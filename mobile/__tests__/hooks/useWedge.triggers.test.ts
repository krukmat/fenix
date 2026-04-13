import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import {
  useTriggerInsightsAgent,
  useTriggerKBAgent,
  useTriggerProspectingAgent,
  wedgeQueryKeys,
} from '../../src/hooks/useWedge';

const mockUseMutation = jest.fn();
const mockUseQueryClient = jest.fn();
const mockUseAuthStore = jest.fn();
const mockInvalidateQueries = jest.fn();

const mockTriggerProspectingRun = jest.fn();
const mockTriggerKBRun = jest.fn();
const mockTriggerInsightsRun = jest.fn();

jest.mock('@tanstack/react-query', () => ({
  useMutation: (...args: unknown[]) => mockUseMutation(...args),
  useQueryClient: () => mockUseQueryClient(),
  useQuery: jest.fn(),
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: (...args: unknown[]) => mockUseAuthStore(...args),
}));

jest.mock('../../src/services/api', () => ({
  inboxApi: { getInbox: jest.fn() },
  approvalApi: { approve: jest.fn(), reject: jest.fn() },
  salesBriefApi: { getSalesBrief: jest.fn() },
  governanceApi: { getSummary: jest.fn(), getAuditEvents: jest.fn(), getUsageEvents: jest.fn() },
  agentApi: {
    getRuns: jest.fn(),
    getRunUsage: jest.fn(),
    triggerSupportRun: jest.fn(),
    triggerProspectingRun: (...args: unknown[]) => mockTriggerProspectingRun(...args),
    triggerKBRun: (...args: unknown[]) => mockTriggerKBRun(...args),
    triggerInsightsRun: (...args: unknown[]) => mockTriggerInsightsRun(...args),
  },
}));

const WORKSPACE_ID = 'ws-test-123';

describe('useWedge trigger hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAuthStore.mockImplementation((selector: (s: { workspaceId: string }) => string) =>
      selector({ workspaceId: WORKSPACE_ID })
    );
    mockUseQueryClient.mockReturnValue({ invalidateQueries: mockInvalidateQueries });
    mockUseMutation.mockImplementation((opts: object) => opts);
  });

  it('useTriggerProspectingAgent calls the typed API with lead_id and invalidates agent runs', async () => {
    mockTriggerProspectingRun.mockResolvedValueOnce({ runId: 'run-1', status: 'queued', agent: 'prospecting' });

    const hook = useTriggerProspectingAgent() as {
      mutationFn: (vars: { leadId: string; language?: string }) => Promise<unknown>;
      onSuccess: () => void;
    };

    await expect(hook.mutationFn({ leadId: 'lead-1', language: 'es' })).resolves.toEqual({
      runId: 'run-1',
      status: 'queued',
      agent: 'prospecting',
    });
    expect(mockTriggerProspectingRun).toHaveBeenCalledWith({ lead_id: 'lead-1', language: 'es' });

    hook.onSuccess();
    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: wedgeQueryKeys.agentRuns(WORKSPACE_ID),
    });
  });

  it('useTriggerKBAgent calls the typed API with case_id and invalidates agent runs', async () => {
    mockTriggerKBRun.mockResolvedValueOnce({ runId: 'run-2', status: 'queued', agent: 'kb' });

    const hook = useTriggerKBAgent() as {
      mutationFn: (vars: { caseId: string; language?: string }) => Promise<unknown>;
      onSuccess: () => void;
    };

    await expect(hook.mutationFn({ caseId: 'case-1' })).resolves.toEqual({
      runId: 'run-2',
      status: 'queued',
      agent: 'kb',
    });
    expect(mockTriggerKBRun).toHaveBeenCalledWith({ case_id: 'case-1', language: undefined });

    hook.onSuccess();
    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: wedgeQueryKeys.agentRuns(WORKSPACE_ID),
    });
  });

  it('useTriggerInsightsAgent passes analytical query payloads through and invalidates agent runs', async () => {
    mockTriggerInsightsRun.mockResolvedValueOnce({ runId: 'run-3', status: 'queued', agent: 'insights' });

    const hook = useTriggerInsightsAgent() as {
      mutationFn: (vars: {
        query: string;
        date_from?: string;
        date_to?: string;
        language?: string;
      }) => Promise<unknown>;
      onSuccess: () => void;
    };

    await expect(hook.mutationFn({
      query: 'show unsupported conclusion',
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-04-13T23:59:59Z',
    })).resolves.toEqual({
      runId: 'run-3',
      status: 'queued',
      agent: 'insights',
    });
    expect(mockTriggerInsightsRun).toHaveBeenCalledWith({
      query: 'show unsupported conclusion',
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-04-13T23:59:59Z',
    });

    hook.onSuccess();
    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: wedgeQueryKeys.agentRuns(WORKSPACE_ID),
    });
  });
});
