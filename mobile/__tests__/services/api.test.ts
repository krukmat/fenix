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
import { apiClient, authApi, crmApi, agentApi } from '../../src/services/api';
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
      expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/accounts', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/contacts', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/deals', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(4, '/bff/api/v1/cases', { params: { workspace_id: 'ws-1', page: 1, limit: 50 } });
      expect(getSpy).toHaveBeenNthCalledWith(5, '/bff/accounts/a1/full');
      expect(getSpy).toHaveBeenNthCalledWith(6, '/bff/deals/d1/full');
      expect(getSpy).toHaveBeenNthCalledWith(7, '/bff/cases/c1/full');
      expect(getSpy).toHaveBeenNthCalledWith(8, '/bff/api/v1/contacts/ct1');
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
  });
});
