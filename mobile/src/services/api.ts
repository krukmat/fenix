// Task 4.2 — FR-300: Axios API Client hacia BFF
// Task Mobile P1.1 — FR-300, UC-A4/A5/A7: signalApi, workflowApi, approvalApi, copilotApi extensions
//
// This file is the public API surface for the services layer.
// Implementation is split across api.client.ts, api.types.ts, api.agents.ts, api.secondary.ts
// to keep each file under the 300-line architecture gate.

import { apiClient } from './api.client';

export { BFF_URL } from './api.client';

// Re-export all types so existing imports (import type { X } from 'services/api') keep working
export type {
  SignalStatus,
  Signal,
  WorkflowStatus,
  Workflow,
  WorkflowVersion,
  CreateWorkflowInput,
  UpdateWorkflowInput,
  AgentRunStatus,
  AgentRun,
  AgentRunListResponse,
  AgentRunResponse,
  ApprovalStatus,
  ApprovalRequest,
  HandoffPackage,
} from './api.types';

// Re-export secondary APIs
export { agentApi } from './api.agents';
export { signalApi, workflowApi, toolApi, approvalApi } from './api.secondary';

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
    const response = await apiClient.get('/bff/accounts', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  // Create account (Task 4.8 — GAP 2)
  createAccount: async (data: { name: string; industry?: string }) => {
    const response = await apiClient.post('/bff/api/v1/accounts', data);
    return response.data;
  },

  getContacts: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/api/v1/contacts', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  getDeals: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/deals', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  getCases: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/cases', {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  // Mutations: deals
  createDeal: async (data: {
    accountId: string;
    contactId?: string;
    pipelineId: string;
    stageId: string;
    ownerId: string;
    title: string;
    amount?: number;
    currency?: string;
    expectedClose?: string;
    status?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.post('/bff/api/v1/deals', data);
    return response.data;
  },

  updateDeal: async (id: string, data: {
    accountId?: string;
    contactId?: string;
    pipelineId?: string;
    stageId?: string;
    ownerId?: string;
    title?: string;
    amount?: number;
    currency?: string;
    expectedClose?: string;
    status?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.put(`/bff/api/v1/deals/${id}`, data);
    return response.data;
  },

  // Mutations: cases
  createCase: async (data: {
    accountId?: string;
    contactId?: string;
    pipelineId?: string;
    stageId?: string;
    ownerId: string;
    subject: string;
    description?: string;
    priority?: string;
    status?: string;
    channel?: string;
    slaConfig?: string;
    slaDeadline?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.post('/bff/api/v1/cases', data);
    return response.data;
  },

  updateCase: async (id: string, data: {
    accountId?: string;
    contactId?: string;
    pipelineId?: string;
    stageId?: string;
    ownerId?: string;
    subject?: string;
    description?: string;
    priority?: string;
    status?: string;
    channel?: string;
    slaConfig?: string;
    slaDeadline?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.put(`/bff/api/v1/cases/${id}`, data);
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

// Copilot API
export const copilotApi = {
  buildChatUrl: (): string => `${process.env.EXPO_PUBLIC_BFF_URL || 'http://10.0.2.2:3000'}/bff/copilot/chat`,

  suggestActions: async (entityType: string, entityId: string) => {
    const response = await apiClient.post('/bff/api/v1/copilot/suggest-actions', {
      entity_type: entityType,
      entity_id: entityId,
    });
    return response.data;
  },

  summarize: async (entityType: string, entityId: string) => {
    const response = await apiClient.post('/bff/api/v1/copilot/summarize', {
      entity_type: entityType,
      entity_id: entityId,
    });
    return response.data;
  },
};
