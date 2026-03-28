// Task 4.2 — FR-300: Axios API Client hacia BFF
// Task Mobile P1.1 — FR-300, UC-A4/A5/A7: signalApi, workflowApi, approvalApi, copilotApi extensions
//
// This file is the public API surface for the services layer.
// Implementation is split across api.client.ts, api.types.ts, api.agents.ts, api.secondary.ts
// to keep each file under the 300-line architecture gate.

import { apiClient } from './api.client';

export { apiClient, BFF_URL } from './api.client';

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

async function getOrNull(path: string, params?: Record<string, string | number | undefined>) {
  try {
    const response = await apiClient.get(path, params ? { params } : undefined);
    return response.data;
  } catch {
    return null;
  }
}

function extractId(obj: Record<string, unknown> | null, camel: string, snake: string): string | undefined {
  return (obj?.[camel] as string | undefined) ?? (obj?.[snake] as string | undefined);
}

function extractSignalCount(obj: Record<string, unknown> | null): number {
  return typeof obj?.active_signal_count === 'number' ? (obj.active_signal_count as number) : 0;
}

// CRM API - Generic fetch helpers
export const crmApi = {
  // Lists
  getAccounts: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/api/v1/accounts', {
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
    const [account, contacts, deals, timeline] = await Promise.all([
      apiClient.get(`/bff/api/v1/accounts/${id}`).then((response) => response.data),
      getOrNull(`/bff/api/v1/accounts/${id}/contacts`),
      getOrNull('/bff/api/v1/deals', { account_id: id, limit: 50 }),
      getOrNull(`/bff/api/v1/timeline/account/${id}`),
    ]);
    return {
      account,
      contacts,
      deals,
      timeline,
      active_signal_count:
        typeof account?.active_signal_count === 'number' ? account.active_signal_count : 0,
    };
  },

  getDealFull: async (id: string) => {
    const deal = await apiClient.get(`/bff/api/v1/deals/${id}`).then((response) => response.data);
    const d = deal as Record<string, unknown> | null;
    const accountId = extractId(d, 'accountId', 'account_id');
    const contactId = extractId(d, 'contactId', 'contact_id');
    const [account, contact, activities] = await Promise.all([
      accountId ? getOrNull(`/bff/api/v1/accounts/${accountId}`) : Promise.resolve(null),
      contactId ? getOrNull(`/bff/api/v1/contacts/${contactId}`) : Promise.resolve(null),
      getOrNull('/bff/api/v1/activities', { deal_id: id, limit: 50 }),
    ]);
    return { deal, account, contact, activities, active_signal_count: extractSignalCount(d) };
  },

  getCaseFull: async (id: string) => {
    const caseData = await apiClient.get(`/bff/api/v1/cases/${id}`).then((response) => response.data);
    const c = caseData as Record<string, unknown> | null;
    const accountId = extractId(c, 'accountId', 'account_id');
    const contactId = extractId(c, 'contactId', 'contact_id');
    const handoffStatus = extractId(c, 'handoffStatus', 'handoff_status');
    const [account, contact, activities] = await Promise.all([
      accountId ? getOrNull(`/bff/api/v1/accounts/${accountId}`) : Promise.resolve(null),
      contactId ? getOrNull(`/bff/api/v1/contacts/${contactId}`) : Promise.resolve(null),
      getOrNull('/bff/api/v1/activities', { case_id: id, limit: 50 }),
    ]);
    return {
      case: caseData,
      account,
      contact,
      activities,
      handoff: handoffStatus ? { status: handoffStatus } : null,
      active_signal_count: extractSignalCount(c),
    };
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
