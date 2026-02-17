/**
 * Task 4.2 — FR-300: Tests TDD para authStore
 * 
 * Tests:
 * 1. initial state unauthenticated
 * 2. login() sets state + persists to SecureStore
 * 3. logout() clears state + deletes from SecureStore
 * 4. loadStoredToken() restores state
 * 5. loadStoredToken() stays unauthenticated when no token
 */

import * as SecureStore from 'expo-secure-store';
import { describe, it, expect, beforeEach, jest } from '@jest/globals';

// Import the store after mocking
// eslint-disable-next-line @typescript-eslint/no-var-requires
const { useAuthStore } = require('../../src/stores/authStore');

describe('authStore', () => {
  beforeEach(() => {
    // Reset store state directly — avoids async side effects of calling logout()
    useAuthStore.setState({
      token: null,
      userId: null,
      workspaceId: null,
      isAuthenticated: false,
      isLoading: false,
    });
    jest.clearAllMocks();
  });

  describe('initial state', () => {
    it('should have unauthenticated initial state', () => {
      const state = useAuthStore.getState();
      expect(state.token).toBeNull();
      expect(state.userId).toBeNull();
      expect(state.workspaceId).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isLoading).toBe(false);
    });
  });

  describe('login()', () => {
    it('should set state and persist to SecureStore', async () => {
      const testData = {
        token: 'test-jwt-token',
        userId: 'user-123',
        workspaceId: 'workspace-456',
      };

      await useAuthStore.getState().login(testData);

      // Check state is updated
      const state = useAuthStore.getState();
      expect(state.token).toBe(testData.token);
      expect(state.userId).toBe(testData.userId);
      expect(state.workspaceId).toBe(testData.workspaceId);
      expect(state.isAuthenticated).toBe(true);

      // Check SecureStore was called
      expect(SecureStore.setItemAsync).toHaveBeenCalledWith(
        'fenixcrm_token',
        JSON.stringify(testData)
      );
    });
  });

  describe('logout()', () => {
    it('should clear state and delete from SecureStore', async () => {
      // First login
      const testData = {
        token: 'test-jwt-token',
        userId: 'user-123',
        workspaceId: 'workspace-456',
      };
      await useAuthStore.getState().login(testData);

      // Then logout
      await useAuthStore.getState().logout();

      // Check state is cleared
      const state = useAuthStore.getState();
      expect(state.token).toBeNull();
      expect(state.userId).toBeNull();
      expect(state.workspaceId).toBeNull();
      expect(state.isAuthenticated).toBe(false);

      // Check SecureStore delete was called
      expect(SecureStore.deleteItemAsync).toHaveBeenCalledWith('fenixcrm_token');
    });
  });

  describe('loadStoredToken()', () => {
    it('should restore state from SecureStore', async () => {
      const testData = {
        token: 'test-jwt-token',
        userId: 'user-123',
        workspaceId: 'workspace-456',
      };

      // Mock SecureStore to return stored data
      (SecureStore.getItemAsync as jest.MockedFunction<typeof SecureStore.getItemAsync>).mockResolvedValueOnce(
        JSON.stringify(testData)
      );

      await useAuthStore.getState().loadStoredToken();

      const state = useAuthStore.getState();
      expect(state.token).toBe(testData.token);
      expect(state.userId).toBe(testData.userId);
      expect(state.workspaceId).toBe(testData.workspaceId);
      expect(state.isAuthenticated).toBe(true);
    });

    it('should stay unauthenticated when no token exists', async () => {
      // Mock SecureStore to return null (no token)
      (SecureStore.getItemAsync as jest.MockedFunction<typeof SecureStore.getItemAsync>).mockResolvedValueOnce(null);

      await useAuthStore.getState().loadStoredToken();

      const state = useAuthStore.getState();
      expect(state.token).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });
});
