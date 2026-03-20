// Axios client instance — extracted from api.ts to keep it under 300 lines
import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { useAuthStore } from '../stores/authStore';

// BFF URL from environment variables
// EXPO_PUBLIC_ prefix is required for Expo SDK 52+
export const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL || 'http://10.0.2.2:3000';

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
