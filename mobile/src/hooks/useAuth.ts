// Task 4.2 — FR-300: Wrapper hook para authStore

import { useAuthStore, AuthData } from '../stores/authStore';

export function useAuth() {
  const {
    token,
    userId,
    workspaceId,
    isAuthenticated,
    isLoading,
    login,
    logout,
    loadStoredToken,
  } = useAuthStore();

  return {
    token,
    userId,
    workspaceId,
    isAuthenticated,
    isLoading,
    login: async (data: AuthData) => login(data),
    logout: async () => logout(),
    loadStoredToken: async () => loadStoredToken(),
  };
}
