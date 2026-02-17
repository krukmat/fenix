// Task 4.2 — FR-300: Axios API Client hacia BFF

import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { useAuthStore } from '../stores/authStore';

// BFF URL from environment variables
// EXPO_PUBLIC_ prefix is required for Expo SDK 52+
const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL || 'http://10.0.2.2:3000';

// Create axios instance
export const apiClient = axios.create({
  baseURL: BFF_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Apply interceptors inline (called once when module loads)
(function applyInterceptors() {
  // Request interceptor: add Authorization header
  apiClient.interceptors.request.use(
    async (config: InternalAxiosRequestConfig) => {
      const { token } = useAuthStore.getState();
      
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      
      return config;
    },
    (error: AxiosError) => {
      return Promise.reject(error);
    }
  );

  // Response interceptor: handle 401 (no refresh token in MVP -> logout)
  apiClient.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      if (error.response?.status === 401) {
        // No refresh token in MVP - logout directly
        await useAuthStore.getState().logout();
      }
      return Promise.reject(error);
    }
  );
})();

// Auth API
export const authApi = {
  login: async (email: string, password: string) => {
    const response = await apiClient.post('/bff/auth/login', {
      email,
      password,
    });
    return response.data;
  },
  
  register: async (displayName: string, email: string, password: string, workspaceName: string) => {
    const response = await apiClient.post('/bff/auth/register', {
      displayName,
      email,
      password,
      workspaceName,
    });
    return response.data;
  },
};

// CRM API - Generic fetch helpers
export const crmApi = {
  // Lists
  getAccounts: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/api/v1/accounts', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  getContacts: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/api/v1/contacts', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  getDeals: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/api/v1/deals', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  getCases: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/api/v1/cases', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },
  
  // Details (aggregated)
  getAccountFull: async (id: string) => {
    const response = await apiClient.get(`/bff/accounts/${id}/full`);
    return response.data;
  },
  
  getDealFull: async (id: string) => {
    const response = await apiClient.get(`/bff/deals/${id}/full`);
    return response.data;
  },
  
  getCaseFull: async (id: string) => {
    const response = await apiClient.get(`/bff/cases/${id}/full`);
    return response.data;
  },
  
  // Contact (no aggregated endpoint)
  getContact: async (id: string) => {
    const response = await apiClient.get(`/bff/api/v1/contacts/${id}`);
    return response.data;
  },
};

// Agent API
export const agentApi = {
  getRuns: async (workspaceId: string) => {
    const response = await apiClient.get(`/bff/api/v1/agents/runs?workspace_id=${workspaceId}`);
    return response.data;
  },
  
  getRun: async (id: string) => {
    const response = await apiClient.get(`/bff/api/v1/agents/runs/${id}`);
    return response.data;
  },
};
