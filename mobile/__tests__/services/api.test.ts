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
import { apiClient, authApi, crmApi, agentApi, signalApi, workflowApi, approvalApi, copilotApi } from '../../src/services/api';
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

      // STEP 8: list endpoints now use params object + pagination (page, limit)
      expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/accounts', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/contacts', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/deals', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(4, '/bff/cases', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(5, '/bff/accounts/a1/full');
      expect(getSpy).toHaveBeenNthCalledWith(6, '/bff/deals/d1/full');
      expect(getSpy).toHaveBeenNthCalledWith(7, '/bff/cases/c1/full');
      expect(getSpy).toHaveBeenNthCalledWith(8, '/bff/api/v1/contacts/ct1');
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

    it('agentApi filtered run helpers should pass entity, workflow and status filters', async () => {
      const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValue({ data: { data: [] } } as never);

      await agentApi.getRuns('ws-1', { page: 2, limit: 10 }, { status: 'delegated', workflow_id: 'wf-1' });
      await agentApi.getRunsByEntity('ws-1', 'case', 'case-1', { page: 3, limit: 5 }, { status: 'rejected' });
      await agentApi.getRunsByWorkflow('ws-1', 'wf-2', { page: 4, limit: 6 }, { entity_type: 'deal', entity_id: 'deal-7' });
      await agentApi.getRunsByStatus('ws-1', 'accepted', { page: 5, limit: 7 }, { workflow_id: 'wf-3' });

      expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/agents/runs', {
        params: { workspace_id: 'ws-1', page: 2, limit: 10, status: 'delegated', workflow_id: 'wf-1' },
      });
      expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/agents/runs', {
        params: { workspace_id: 'ws-1', page: 3, limit: 5, status: 'rejected', entity_type: 'case', entity_id: 'case-1' },
      });
      expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/agents/runs', {
        params: { workspace_id: 'ws-1', page: 4, limit: 6, entity_type: 'deal', entity_id: 'deal-7', workflow_id: 'wf-2' },
      });
      expect(getSpy).toHaveBeenNthCalledWith(4, '/bff/api/v1/agents/runs', {
        params: { workspace_id: 'ws-1', page: 5, limit: 7, workflow_id: 'wf-3', status: 'accepted' },
      });
    });

    // --- Mobile P1.1: signalApi, workflowApi, approvalApi, copilotApi ---

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

    describe('workflowApi', () => {
      it('getWorkflows should call GET /bff/api/v1/workflows with workspace_id and pagination defaults', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await workflowApi.getWorkflows('ws-1');

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/workflows', {
          params: { workspace_id: 'ws-1', page: 1, limit: 50 },
        });
      });

      it('getWorkflows should respect custom pagination', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await workflowApi.getWorkflows('ws-1', undefined, { page: 2, limit: 10 });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/workflows', {
          params: { workspace_id: 'ws-1', page: 2, limit: 10 },
        });
      });

      it('getWorkflows should pass status filter', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await workflowApi.getWorkflows('ws-1', { status: 'active' });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/workflows', {
          params: { workspace_id: 'ws-1', page: 1, limit: 50, status: 'active' },
        });
      });

      it('getWorkflows should return empty array when no workflows exist', async () => {
        jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        const result = await workflowApi.getWorkflows('ws-1');

        expect(result).toEqual([]);
      });

      it('getWorkflows uses default page when only limit is provided', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await workflowApi.getWorkflows('ws-1', undefined, { limit: 10 });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/workflows', {
          params: { workspace_id: 'ws-1', page: 1, limit: 10 },
        });
      });

      it('getWorkflows uses default limit when only page is provided', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [] } as never);

        await workflowApi.getWorkflows('ws-1', undefined, { page: 3 });

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/workflows', {
          params: { workspace_id: 'ws-1', page: 3, limit: 50 },
        });
      });

      it('getWorkflow should call GET /bff/api/v1/workflows/{id}', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: { id: 'wf-1' } } as never);

        const result = await workflowApi.getWorkflow('wf-1');

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-1');
        expect(result).toEqual({ id: 'wf-1' });
      });

      it('create should call POST /bff/api/v1/workflows', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { id: 'wf-new' } } as never);

        const result = await workflowApi.create({
          name: 'Lead Qualifier',
          description: 'Draft workflow',
          dsl_source: 'ON lead.created',
          spec_source: 'spec text',
        });

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/workflows', {
          name: 'Lead Qualifier',
          description: 'Draft workflow',
          dsl_source: 'ON lead.created',
          spec_source: 'spec text',
        });
        expect(result).toEqual({ id: 'wf-new' });
      });

      it('update should call PUT /bff/api/v1/workflows/{id}', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { id: 'wf-1', description: 'updated' } } as never);

        const result = await workflowApi.update('wf-1', {
          agent_definition_id: 'agent-7',
          description: 'updated',
          dsl_source: 'ON lead.updated',
        });

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-1', {
          agent_definition_id: 'agent-7',
          description: 'updated',
          dsl_source: 'ON lead.updated',
        });
        expect(result).toEqual({ id: 'wf-1', description: 'updated' });
      });

      it('getVersions should call GET /bff/api/v1/workflows/{id}/versions', async () => {
        const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValueOnce({ data: [{ id: 'wf-1', version: 1 }] } as never);

        const result = await workflowApi.getVersions('wf-1');

        expect(getSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-1/versions');
        expect(result).toEqual([{ id: 'wf-1', version: 1 }]);
      });

      it('newVersion should call POST /bff/api/v1/workflows/{id}/new-version', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { id: 'wf-2', version: 2 } } as never);

        const result = await workflowApi.newVersion('wf-1');

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-1/new-version');
        expect(result).toEqual({ id: 'wf-2', version: 2 });
      });

      it('rollback should call PUT /bff/api/v1/workflows/{id}/rollback', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { id: 'wf-0', status: 'active' } } as never);

        const result = await workflowApi.rollback('wf-0');

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-0/rollback');
        expect(result).toEqual({ id: 'wf-0', status: 'active' });
      });


      it('activateWorkflow should call PUT /bff/api/v1/workflows/{id}/activate', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { id: 'wf-1', status: 'active' } } as never);

        const result = await workflowApi.activateWorkflow('wf-1');

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-1/activate');
        expect(result).toEqual({ id: 'wf-1', status: 'active' });
      });


      it('executeWorkflow should call POST /bff/api/v1/workflows/{id}/execute', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { run_id: 'run-1' } } as never);

        const result = await workflowApi.executeWorkflow('wf-1');

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-1/execute');
        expect(result).toEqual({ run_id: 'run-1' });
      });


      it('verifyWorkflow should call POST /bff/api/v1/workflows/{id}/verify with passed: true', async () => {
        const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { passed: true } } as never);

        const result = await workflowApi.verifyWorkflow('wf-1');

        expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/workflows/wf-1/verify');
        expect(result).toEqual({ passed: true });
      });

      it('verifyWorkflow should return passed: false with violations list', async () => {
        const violations = [{ check: 'CHECK1', message: 'missing ON event', line: 1 }];
        jest.spyOn(apiClient, 'post').mockResolvedValueOnce({ data: { passed: false, violations } } as never);

        const result = await workflowApi.verifyWorkflow('wf-invalid');

        expect(result).toEqual({ passed: false, violations });
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

      it('decideApproval should call PUT /bff/api/v1/approvals/{id} with deny decision and reason', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { ok: true } } as never);

        await approvalApi.decideApproval('apr-1', { decision: 'deny', reason: 'not needed' });

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/approvals/apr-1', { decision: 'deny', reason: 'not needed' });
      });

      it('decideApproval should call PUT without reason when not provided', async () => {
        const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValueOnce({ data: { ok: true } } as never);

        await approvalApi.decideApproval('apr-1', { decision: 'deny' });

        expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/approvals/apr-1', { decision: 'deny' });
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
  });
});
