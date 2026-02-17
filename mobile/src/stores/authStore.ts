// Task 4.2 — FR-300: Zustand + expo-secure-store Auth Store

import { create } from 'zustand';
import * as SecureStore from 'expo-secure-store';

const AUTH_TOKEN_KEY = 'fenixcrm_token';

export interface AuthData {
  token: string;
  userId: string;
  workspaceId: string;
}

interface AuthState {
  token: string | null;
  userId: string | null;
  workspaceId: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (data: AuthData) => Promise<void>;
  logout: () => Promise<void>;
  loadStoredToken: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  userId: null,
  workspaceId: null,
  isAuthenticated: false,
  isLoading: false,

  login: async (data: AuthData) => {
    try {
      // Persist to SecureStore
      await SecureStore.setItemAsync(AUTH_TOKEN_KEY, JSON.stringify(data));
      
      set({
        token: data.token,
        userId: data.userId,
        workspaceId: data.workspaceId,
        isAuthenticated: true,
      });
    } catch (error) {
      console.error('Failed to login:', error);
      throw error;
    }
  },

  logout: async () => {
    // Clear in-memory state first - always must succeed
    set({
      token: null,
      userId: null,
      workspaceId: null,
      isAuthenticated: false,
    });
    
    try {
      // Try to delete from SecureStore
      await SecureStore.deleteItemAsync(AUTH_TOKEN_KEY);
    } catch (error) {
      // Log but don't throw - logout must always complete from user perspective
      console.error('Failed to delete token from SecureStore:', error);
    }
  },

  loadStoredToken: async () => {
    set({ isLoading: true });
    
    try {
      const stored = await SecureStore.getItemAsync(AUTH_TOKEN_KEY);
      
      if (stored) {
        const data: AuthData = JSON.parse(stored);
        set({
          token: data.token,
          userId: data.userId,
          workspaceId: data.workspaceId,
          isAuthenticated: true,
          isLoading: false,
        });
      } else {
        set({ isLoading: false });
      }
    } catch (error) {
      console.error('Failed to load stored token:', error);
      set({ isLoading: false });
    }
  },
}));
