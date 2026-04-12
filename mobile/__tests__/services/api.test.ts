/**
 * Task 4.2 — FR-300: Tests TDD para api.ts
 *
 * Tests:
 * 1. creates Axios with BFF base URL
 * 2. attaches Authorization header from auth store token
 * 3. does not add Authorization header when no token
 * 4. calls logout() when receiving a 401 response
 * 5. does not call logout() for non-401 errors
 */

import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { InternalAxiosRequestConfig } from 'axios';
import {
  apiClient,
  authApi,
  crmApi,
  agentApi,
  signalApi,
  approvalApi,
  inboxApi,
  copilotApi,
  salesBriefApi,
} from '../../src/services/api';
import { useAuthStore } from '../../src/stores/authStore';

const mockLogout = jest.fn<() => Promise<void>>().mockResolvedValue(undefined);

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: {
    getState: jest.fn(),
  },
}));

// Minimal mock shape — cast via unknown avoids having to satisfy full AuthState
type MockAuthState = { token: string | null; logout: jest.Mock<() => Promise<void>> };

// Cast via unknown to access Axios internal handlers array (not part of public Axios API)
type InterceptorHandlers<T> = {
  handlers: ({ fulfilled: (v: T) => T | Promise<T>; rejected: (e: unknown) => unknown } | null)[];
};

describe('api.ts', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (useAuthStore.getState as jest.MockedFunction<() => unknown>).mockReturnValue(
      { token: 'test-token', logout: mockLogout } satisfies MockAuthState
    );
  });

  describe('apiClient base URL', () => {
    it('should have BFF base URL configured', () => {
      expect(apiClient.defaults.baseURL).toBeDefined();
      expect(apiClient.defaults.baseURL).toContain('3000');
    });
  });

  describe('Request interceptor — Authorization header', () => {
    it('should attach Authorization header from auth store token', async () => {
      const manager = apiClient.interceptors.request as unknown as InterceptorHandlers<InternalAxiosRequestConfig>;
      const handler = manager.handlers.find(Boolean);
      expect(handler).toBeDefined();

      const config = { headers: { 'Content-Type': 'application/json' } } as InternalAxiosRequestConfig;
      const result = await handler!.fulfilled(config) as InternalAxiosRequestConfig;

      expect(result.headers['Authorization']).toBe('Bearer test-token');
    });

    it('should not add Authorization header when no token', async () => {
      (useAuthStore.getState as jest.MockedFunction<() => unknown>).mockReturnValueOnce(
        { token: null, logout: mockLogout } satisfies MockAuthState
      );

      const manager = apiClient.interceptors.request as unknown as InterceptorHandlers<InternalAxiosRequestConfig>;
      const handler = manager.handlers.find(Boolean);
      const config = { headers: { 'Content-Type': 'application/json' } } as InternalAxiosRequestConfig;
      const result = await handler!.fulfilled(config) as InternalAxiosRequestConfig;

      expect(result.headers['Authorization']).toBeUndefined();
    });
  });

  describe('Response interceptor — 401 handling', () => {
    it('should call logout() when receiving a 401 response', async () => {
      const manager = apiClient.interceptors.response as unknown as InterceptorHandlers<unknown>;
      const handler = manager.handlers.find(Boolean);
      expect(handler).toBeDefined();

      const error401 = { response: { status: 401, data: {} }, isAxiosError: true };
      await expect(handler!.rejected(error401)).rejects.toBeDefined();

      expect(mockLogout).toHaveBeenCalledTimes(1);
    });

    it('should not call logout() for non-401 errors', async () => {
      const manager = apiClient.interceptors.response as unknown as InterceptorHandlers<unknown>;
      const handler = manager.handlers.find(Boolean);

      const error500 = { response: { status: 500, data: {} }, isAxiosError: true };
      await expect(handler!.rejected(error500)).rejects.toBeDefined();

      expect(mockLogout).not.toHaveBeenCalled();
    });
  });

  describe('API wrappers', () => {
    it('authApi.login should call POST /bff/auth/login', async () => {
      const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { ok: true } } as never);

      const result = await authApi.login('a@b.com', 'secret');

      expect(postSpy).toHaveBeenCalledWith('/bff/auth/login', {
        email: 'a@b.com',
        password: 'secret',
      });
      expect(result).toEqual({ ok: true });
    });

    it('authApi.register should call POST /bff/auth/register', async () => {
      const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { created: true } } as never);

      const result = await authApi.register('User', 'u@b.com', 'pass', 'WS');

      expect(postSpy).toHaveBeenCalledWith('/bff/auth/register', {
        displayName: 'User',
        email: 'u@b.com',
        password: 'pass',
        workspaceName: 'WS',
      });
      expect(result).toEqual({ created: true });
    });

    it('crmApi methods should call expected GET endpoints', async () => {
      const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValue({ data: { ok: true } } as never);

      await crmApi.getAccounts('ws-1');
      await crmApi.getContacts('ws-1');
      await crmApi.getDeals('ws-1');
      await crmApi.getCases('ws-1');
      await crmApi.getAccountFull('a1');
      await crmApi.getDealFull('d1');
      await crmApi.getCaseFull('c1');
      await crmApi.getContact('ct1');

      expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/accounts', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/contacts', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/deals', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(4, '/bff/api/v1/cases', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(5, '/bff/api/v1/accounts/a1');
      expect(getSpy).toHaveBeenNthCalledWith(6, '/bff/api/v1/accounts/a1/contacts', undefined);
      expect(getSpy).toHaveBeenNthCalledWith(7, '/bff/api/v1/deals', { params: { account_id: 'a1', limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(8, '/bff/api/v1/timeline/account/a1', undefined);
      expect(getSpy).toHaveBeenNthCalledWith(9, '/bff/api/v1/deals/d1');
      expect(getSpy).toHaveBeenNthCalledWith(10, '/bff/api/v1/activities', { params: { deal_id: 'd1', limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(11, '/bff/api/v1/cases/c1');
      expect(getSpy).toHaveBeenNthCalledWith(12, '/bff/api/v1/activities', { params: { case_id: 'c1', limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(13, '/bff/api/v1/contacts/ct1');
    });

    it('crmApi deal/case mutations should call expected POST/PUT endpoints', async () => {
      const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValue({ data: { ok: true } } as never);
      const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValue({ data: { ok: true } } as never);

      await crmApi.createDeal({
        accountId: 'acc-1',
        pipelineId: 'pipe-1',
        stageId: 'stage-1',
        ownerId: 'owner-1',
        title: 'Deal A',
      });
      await crmApi.updateDeal('deal-1', { status: 'won' });
      await crmApi.createCase({ ownerId: 'owner-1', subject: 'Case A' });
      await crmApi.updateCase('case-1', { status: 'in_progress' });

      expect(postSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/deals', {
        accountId: 'acc-1',
        pipelineId: 'pipe-1',
        stageId: 'stage-1',
        ownerId: 'owner-1',
        title: 'Deal A',
      });
      expect(putSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/deals/deal-1', { status: 'won' });
      expect(postSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/cases', {
        ownerId: 'owner-1',
        subject: 'Case A',
      });
      expect(putSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/cases/case-1', { status: 'in_progress' });
    });

    it('agentApi methods should call expected GET endpoints', async () => {
      const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValue({ data: { ok: true } } as never);
      const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValue({ data: { ok: true } } as never);

      await agentApi.getRuns('ws-1');
      await agentApi.getRun('run-1');
      await agentApi.getDefinitions('ws-1');
      await agentApi.triggerRun('agent-1', { entity_type: 'case', entity_id: 'c1' });

      expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/agents/runs', { params: { workspace_id: 'ws-1', page: 1, limit: 25 } });
      expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/agents/runs/run-1');
      expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/agents/definitions', { params: { workspace_id: 'ws-1' } });
      expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/agents/trigger', {
        agent_id: 'agent-1',
        entity_type: 'case',
        entity_id: 'c1',
      });
    });

    it('agentApi.getHandoff should normalize handoff payloads with case context fields', async () => {
      const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({
        data: {
          data: {
            runId: 'run-1',
            reason: 'Escalated to human',
            caseId: 'case-1',
            triggerContext: { entity_type: 'case', entity_id: 'case-1' },
            evidencePack: { source_count: 2 },
            startedAt: '2026-04-08T10:00:00Z',
          },
        },
      } as never);

      const result = await agentApi.getHandoff('run-1');

      expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/agents/runs/run-1/handoff', { params: undefined });
      expect(result).toMatchObject({
        run_id: 'run-1',
        entity_type: 'case',
        entity_id: 'case-1',
        evidence_count: 2,
        caseId: 'case-1',
      });
    });

    it('agentApi filtered run helpers should pass entity and status filters', async () => {
      const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValue({ data: { data: [] } } as never);

      await agentApi.getRuns('ws-1', { page: 2, limit: 10 }, { status: 'delegated' });
      await agentApi.getRunsByEntity('ws-1', 'case', 'case-1', { page: 3, limit: 5 }, { status: 'rejected' });
      await agentApi.getRunsByStatus('ws-1', 'accepted', { page: 5, limit: 7 }, { entity_type: 'case' });

      expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/agents/runs', {
        params: { workspace_id: 'ws-1', page: 2, limit: 10, status: 'delegated' },
      });
      expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/agents/runs', {
        params: { workspace_id: 'ws-1', page: 3, limit: 5, status: 'rejected', entity_type: 'case', entity_id: 'case-1' },
      });
      expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/agents/runs', {
        params: { workspace_id: 'ws-1', page: 5, limit: 7, entity_type: 'case', status: 'accepted' },
      });
    });

    it('inboxApi.getInbox should normalize nested handoff payloads', async () => {
      jest.spyOn(apiClient, 'get').mockResolvedValueOnce({
        data: {
          approvals: [],
          handoffs: [
            {
              run_id: 'run-1',
              handoff: {
                data: {
                  runId: 'run-1',
                  reason: 'Escalated to human',
                  finalOutput: { entity_type: 'deal', entity_id: 'deal-1' },
                  evidencePack: { source_count: 1 },
                  completedAt: '2026-04-08T10:00:00Z',
                },
              },
            },
          ],
          signals: [],
        },
      } as never);

      const result = await inboxApi.getInbox('ws-1');

      expect(result.handoffs[0]).toMatchObject({
        run_id: 'run-1',
        handoff: {
          run_id: 'run-1',
          entity_type: 'deal',
          entity_id: 'deal-1',
          evidence_count: 1,
        },
      });
    });

    // --- Mobile P1.1: signalApi, approvalApi, copilotApi ---

    describe('signalApi', () => {
      it('getSignals should call GET /bff/api/v1/signals with workspace_id and pagination defaults', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await signalApi.getSignals('ws-1');

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/signals', {
          params: { workspace_id: 'ws-1', page: 1, limit: 50 },
        });
      });

      it('getSignals should respect custom pagination', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await signalApi.getSignals('ws-1', undefined, { page: 3, limit: 20 });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/signals', {
          params: { workspace_id: 'ws-1', page: 3, limit: 20 },
        });
      });

      it('getSignals should pass status and entity filters', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await signalApi.getSignals('ws-1', { status: 'active', entity_type: 'deal', entity_id: 'd-1' });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/signals', {
          params: { workspace_id: 'ws-1', page: 1, limit: 50, status: 'active', entity_type: 'deal', entity_id: 'd-1' },
        });
      });

      it('getSignals should return empty array when no signals exist', async () => {
        jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        const result = await signalApi.getSignals('ws-1');

        expect(result).toEqual([]);
      });

      it('getSignals should pass only entity_type filter without entity_id', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await signalApi.getSignals('ws-1', { entity_type: 'contact' });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/signals', {
          params: { workspace_id: 'ws-1', page: 1, limit: 50, entity_type: 'contact' },
        });
      });

      it('getSignals uses default page when only limit is provided', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await signalApi.getSignals('ws-1', undefined, { limit: 10 });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/signals', {
          params: { workspace_id: 'ws-1', page: 1, limit: 10 },
        });
      });

      it('getSignals uses default limit when only page is provided', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await signalApi.getSignals('ws-1', undefined, { page: 2 });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/signals', {
          params: { workspace_id: 'ws-1', page: 2, limit: 50 },
        });
      });

      it('getSignals with empty filters object does not add extra params', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await signalApi.getSignals('ws-1', {});

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/signals', {
          params: { workspace_id: 'ws-1', page: 1, limit: 50 },
        });
      });

      it('dismissSignal should call PUT /bff/api/v1/signals/{id}/dismiss', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { id: 'sig-1', status: 'dismissed' } } as never);

        const result = await signalApi.dismissSignal('sig-1');

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/signals/sig-1/dismiss');
        expect(result).toEqual({ id: 'sig-1', status: 'dismissed' });
      });

    });

    describe('approvalApi', () => {
      it('getPendingApprovals should call GET /bff/api/v1/approvals with workspace_id', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await approvalApi.getPendingApprovals('ws-1');

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/approvals', {
          params: { workspace_id: 'ws-1' },
        });
      });

      it('getPendingApprovals should return empty array when no pending approvals', async () => {
        jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        const result = await approvalApi.getPendingApprovals('ws-1');

        expect(result).toEqual([]);
      });

      it('decideApproval should call PUT /bff/api/v1/approvals/{id} with approve decision', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { ok: true } } as never);

        await approvalApi.decideApproval('apr-1', { decision: 'approve' });

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/approvals/apr-1', { decision: 'approve' });
      });

      it('decideApproval should call PUT /bff/api/v1/approvals/{id} with reject decision and reason', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { ok: true } } as never);

        await approvalApi.decideApproval('apr-1', { decision: 'reject', reason: 'not needed' });

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/approvals/apr-1', { decision: 'reject', reason: 'not needed' });
      });

      it('decideApproval should call PUT without reason when not provided', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { ok: true } } as never);

        await approvalApi.decideApproval('apr-1', { decision: 'reject' });

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/approvals/apr-1', { decision: 'reject' });
      });

    });

    describe('copilotApi extensions', () => {
      it('suggestActions should call POST /bff/api/v1/copilot/suggest-actions', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { actions: [] } } as never);

        const result = await copilotApi.suggestActions('deal', 'd-1');

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/copilot/suggest-actions', {
          entity_type: 'deal',
          entity_id: 'd-1',
        });
        expect(result).toEqual({ actions: [] });
      });

      it('suggestActions should return empty actions when no suggestions available', async () => {
        jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { actions: [] } } as never);

        const result = await copilotApi.suggestActions('contact', 'ct-1');

        expect(result).toEqual({ actions: [] });
      });

      it('suggestActions should support all entity types', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValue({ data: { actions: [] } } as never);

        await copilotApi.suggestActions('account', 'a-1');
        await copilotApi.suggestActions('contact', 'ct-1');
        await copilotApi.suggestActions('lead', 'l-1');
        await copilotApi.suggestActions('case', 'cs-1');

        expect(postSpy).toHaveBeenCalledTimes(4);
        expect(postSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/copilot/suggest-actions', { entity_type: 'account', entity_id: 'a-1' });
        expect(postSpy).toHaveBeenNthCalledWith(4, '/bff/api/v1/copilot/suggest-actions', { entity_type: 'case', entity_id: 'cs-1' });
      });

      it('summarize should call POST /bff/api/v1/copilot/summarize', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { summary: 'text' } } as never);

        const result = await copilotApi.summarize('case', 'c-1');

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/copilot/summarize', {
          entity_type: 'case',
          entity_id: 'c-1',
        });
        expect(result).toEqual({ summary: 'text' });
      });

    });

    describe('salesBriefApi', () => {
      it('getSalesBrief should call POST /bff/api/v1/copilot/sales-brief with camelCase entity fields', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({
          data: {
            outcome: 'completed',
            summary: 'Healthy pipeline',
            nextBestActions: [{
              title: 'Update deal',
              description: 'Capture the latest procurement status.',
              tool: 'update_deal',
              params: { deal_id: 'deal-1' },
            }],
          },
        } as never);

        const result = await salesBriefApi.getSalesBrief('account', 'acc-1');

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/copilot/sales-brief', {
          entityType: 'account',
          entityId: 'acc-1',
        });
        expect(result).toEqual({
          outcome: 'completed',
          summary: 'Healthy pipeline',
          nextBestActions: [{
            title: 'Update deal',
            description: 'Capture the latest procurement status.',
            tool: 'update_deal',
            params: { deal_id: 'deal-1' },
          }],
        });
      });

      it('always calls POST regardless of EXPO_PUBLIC_E2E_MODE env var', async () => {
        const saved = process.env.EXPO_PUBLIC_E2E_MODE;
        process.env.EXPO_PUBLIC_E2E_MODE = '1';

        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({
          data: { outcome: 'completed', summary: 'test' },
        } as never);

        await salesBriefApi.getSalesBrief('deal', 'deal-1');

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/copilot/sales-brief', {
          entityType: 'deal',
          entityId: 'deal-1',
        });

        process.env.EXPO_PUBLIC_E2E_MODE = saved;
      });
    });
  });
});
