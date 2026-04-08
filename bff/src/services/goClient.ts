// Task 4.1 — FR-301: Axios instance pre-configured for Go backend calls
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import { config } from '../config';

// Copilot and agent endpoints can take materially longer than CRUD calls when a local LLM is involved.
const GO_TIMEOUT_MS = 120000;
const GO_HEALTH_TIMEOUT_MS = 2000;

export function createGoClient(token?: string): AxiosInstance {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = token;
  }

  return axios.create({
    baseURL: config.backendUrl,
    timeout: GO_TIMEOUT_MS,
    headers,
  });
}

// Ping Go /readyz — used by BFF health route
export async function pingGoBackend(): Promise<{ reachable: boolean; latencyMs: number }> {
  const start = Date.now();
  try {
    await axios.get(`${config.backendUrl}/readyz`, {
      timeout: GO_HEALTH_TIMEOUT_MS,
    } satisfies AxiosRequestConfig);
    return { reachable: true, latencyMs: Date.now() - start };
  } catch {
    return { reachable: false, latencyMs: Date.now() - start };
  }
}
