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
import { apiClient } from '../../src/services/api';
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
});
