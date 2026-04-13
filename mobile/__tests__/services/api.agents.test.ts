import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { agentApi } from '../../src/services/api';
import { apiClient } from '../../src/services/api.client';

describe('api.agents', () => {
  beforeEach(() => {
    jest.restoreAllMocks();
  });

  it('triggerProspectingRun posts to the prospecting endpoint and normalizes the response', async () => {
    const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({
      data: { run_id: 'run-pro-1', status: 'queued', agent: 'prospecting' },
    } as never);

    const result = await agentApi.triggerProspectingRun({ lead_id: 'lead-1', language: 'es' });

    expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/agents/prospecting/trigger', {
      lead_id: 'lead-1',
      language: 'es',
    });
    expect(result).toEqual({ runId: 'run-pro-1', status: 'queued', agent: 'prospecting' });
  });

  it('triggerKBRun posts to the kb endpoint and normalizes the response', async () => {
    const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({
      data: { run_id: 'run-kb-1', status: 'queued', agent: 'kb' },
    } as never);

    const result = await agentApi.triggerKBRun({ case_id: 'case-1' });

    expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/agents/kb/trigger', {
      case_id: 'case-1',
    });
    expect(result).toEqual({ runId: 'run-kb-1', status: 'queued', agent: 'kb' });
  });

  it('triggerInsightsRun posts to the insights endpoint and normalizes the response', async () => {
    const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({
      data: { run_id: 'run-ins-1', status: 'queued', agent: 'insights' },
    } as never);

    const result = await agentApi.triggerInsightsRun({
      query: 'show stalled deals',
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-04-13T23:59:59Z',
      language: 'en',
    });

    expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/agents/insights/trigger', {
      query: 'show stalled deals',
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-04-13T23:59:59Z',
      language: 'en',
    });
    expect(result).toEqual({ runId: 'run-ins-1', status: 'queued', agent: 'insights' });
  });
});
