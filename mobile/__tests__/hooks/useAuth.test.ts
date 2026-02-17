import { describe, it, expect, beforeEach, jest } from '@jest/globals';

import { useAuth } from '../../src/hooks/useAuth';

const mockStore = {
  token: 'jwt-token',
  userId: 'user-1',
  workspaceId: 'ws-1',
  isAuthenticated: true,
  isLoading: false,
  login: jest.fn(async () => undefined),
  logout: jest.fn(async () => undefined),
  loadStoredToken: jest.fn(async () => undefined),
};

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: () => mockStore,
}));

describe('useAuth', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockStore.token = 'jwt-token';
    mockStore.userId = 'user-1';
    mockStore.workspaceId = 'ws-1';
    mockStore.isAuthenticated = true;
    mockStore.isLoading = false;
  });

  it('should expose auth state from store', () => {
    const result = useAuth();

    expect(result.token).toBe('jwt-token');
    expect(result.userId).toBe('user-1');
    expect(result.workspaceId).toBe('ws-1');
    expect(result.isAuthenticated).toBe(true);
    expect(result.isLoading).toBe(false);
  });

  it('should call wrapped async actions', async () => {
    const result = useAuth();

    await result.login({ token: 'new-token', userId: 'user-2', workspaceId: 'ws-2' });
    await result.logout();
    await result.loadStoredToken();

    expect(mockStore.login).toHaveBeenCalledWith({
      token: 'new-token',
      userId: 'user-2',
      workspaceId: 'ws-2',
    });
    expect(mockStore.logout).toHaveBeenCalledTimes(1);
    expect(mockStore.loadStoredToken).toHaveBeenCalledTimes(1);
  });
});
