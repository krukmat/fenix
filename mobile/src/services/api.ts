// Task 4.2 — FR-300: Axios API Client hacia BFF
// Task Mobile P1.1 — FR-300, UC-A4/A5/A7: signalApi, workflowApi, approvalApi, copilotApi extensions

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

// Agent API
export const agentApi = {
  getRuns: async (
    workspaceId: string,
    pagination?: { page?: number; limit?: number },
    filters?: { status?: AgentRunStatus; entity_type?: string; entity_id?: string; workflow_id?: string }
  ) => {
    const response = await apiClient.get('/bff/api/v1/agents/runs', {
      params: {
        workspace_id: workspaceId,
        page: pagination?.page ?? 1,
        limit: pagination?.limit ?? 25,
        ...filters,
      },
    });
    return response.data;
  },

  getRunsByEntity: async (
    workspaceId: string,
    entityType: string,
    entityId: string,
    pagination?: { page?: number; limit?: number },
    filters?: { status?: AgentRunStatus; workflow_id?: string }
  ) => {
    return agentApi.getRuns(workspaceId, pagination, {
      ...filters,
      entity_type: entityType,
      entity_id: entityId,
    });
  },

  getRunsByWorkflow: async (
    workspaceId: string,
    workflowId: string,
    pagination?: { page?: number; limit?: number },
    filters?: { status?: AgentRunStatus; entity_type?: string; entity_id?: string }
  ) => {
    return agentApi.getRuns(workspaceId, pagination, {
      ...filters,
      workflow_id: workflowId,
    });
  },

  getRunsByStatus: async (
    workspaceId: string,
    status: AgentRunStatus,
    pagination?: { page?: number; limit?: number },
    filters?: { entity_type?: string; entity_id?: string; workflow_id?: string }
  ) => {
    return agentApi.getRuns(workspaceId, pagination, {
      ...filters,
      status,
    });
  },

  getRun: async (id: string) => {
    const response = await apiClient.get(`/bff/api/v1/agents/runs/${id}`);
    return response.data;
  },

  getDefinitions: async (workspaceId: string) => {
    const response = await apiClient.get('/bff/api/v1/agents/definitions', {
      params: { workspace_id: workspaceId },
    });
    return response.data;
  },

  triggerRun: async (agentId: string, context: { entity_type?: string; entity_id?: string }) => {
    const response = await apiClient.post(`/bff/api/v1/agents/trigger`, {
      agent_id: agentId,
      ...context,
    });
    return response.data;
  },

  // Task Mobile P1.8 — FR-232/UC-A7: handoff package for escalated runs
  getHandoff: async (runId: string) => {
    const response = await apiClient.get(`/bff/api/v1/agents/runs/${runId}/handoff`);
    return response.data as HandoffPackage;
  },
};

// Copilot API
export const copilotApi = {
  buildChatUrl: (): string => `${BFF_URL}/bff/copilot/chat`,

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

// Tool API
export const toolApi = {
  execute: async (tool: string, params: Record<string, unknown>) => {
    const response = await apiClient.post(`/bff/api/v1/tools/${tool}`, params);
    return response.data;
  },
};

// --- Types ---

export type SignalStatus = 'active' | 'dismissed' | 'expired';

export interface Signal {
  id: string;
  workspace_id: string;
  entity_type: string;
  entity_id: string;
  signal_type: string;
  confidence: number;
  evidence_ids: string[];
  source_type: string;
  source_id: string;
  metadata: Record<string, unknown>;
  status: SignalStatus;
  dismissed_by?: string;
  dismissed_at?: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export type WorkflowStatus = 'draft' | 'testing' | 'active' | 'archived';

export interface Workflow {
  id: string;
  workspace_id: string;
  agent_definition_id?: string;
  parent_version_id?: string;
  name: string;
  description?: string;
  dsl_source: string;
  spec_source?: string;
  version: number;
  status: WorkflowStatus;
  created_by_user_id?: string;
  archived_at?: string;
  created_at: string;
  updated_at: string;
}

export type WorkflowVersion = Workflow;

export interface CreateWorkflowInput {
  agent_definition_id?: string;
  name: string;
  description?: string;
  dsl_source: string;
  spec_source?: string;
}

export interface UpdateWorkflowInput {
  agent_definition_id?: string;
  description?: string;
  dsl_source: string;
  spec_source?: string;
}

export type AgentRunStatus =
  | 'running'
  | 'success'
  | 'failed'
  | 'abstained'
  | 'partial'
  | 'escalated'
  | 'accepted'
  | 'rejected'
  | 'delegated';

export interface AgentRun {
  id: string;
  workspaceId: string;
  agentDefinitionId: string;
  triggeredByUserId?: string;
  triggerType: string;
  status: AgentRunStatus;
  inputs?: Record<string, unknown> | null;
  output?: Record<string, unknown> | null;
  toolCalls?: unknown[] | Record<string, unknown> | null;
  reasoningTrace?: unknown[] | Record<string, unknown> | null;
  totalTokens?: number;
  totalCost?: number;
  latencyMs?: number;
  traceId?: string;
  workflow_id?: string;
  entity_type?: string;
  entity_id?: string;
  rejection_reason?: string;
  startedAt: string;
  completedAt?: string;
  createdAt: string;
}

export interface AgentRunListResponse {
  data: AgentRun[];
  meta?: {
    total?: number;
    limit?: number;
    offset?: number;
  };
}

export interface AgentRunResponse {
  data: AgentRun;
}

export type ApprovalStatus = 'pending' | 'approved' | 'denied' | 'expired';

export interface ApprovalRequest {
  id: string;
  workspace_id: string;
  requested_by: string;
  approver_id: string;
  decided_by?: string;
  action: string;
  resource_type?: string;
  resource_id?: string;
  payload: Record<string, unknown>;
  reason?: string;
  status: ApprovalStatus;
  expires_at: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
}

// Signal API
export const signalApi = {
  getSignals: async (
    workspaceId: string,
    filters?: { status?: SignalStatus; entity_type?: string; entity_id?: string },
    pagination?: { page?: number; limit?: number }
  ) => {
    const response = await apiClient.get('/bff/api/v1/signals', {
      params: {
        workspace_id: workspaceId,
        page: pagination?.page ?? 1,
        limit: pagination?.limit ?? 50,
        ...filters,
      },
    });
    return response.data as Signal[];
  },

  dismissSignal: async (id: string) => {
    const response = await apiClient.put(`/bff/api/v1/signals/${id}/dismiss`);
    return response.data as Signal;
  },
};

// Workflow API
export const workflowApi = {
  getWorkflows: async (
    workspaceId: string,
    filters?: { status?: WorkflowStatus },
    pagination?: { page?: number; limit?: number }
  ) => {
    const response = await apiClient.get('/bff/api/v1/workflows', {
      params: {
        workspace_id: workspaceId,
        page: pagination?.page ?? 1,
        limit: pagination?.limit ?? 50,
        ...filters,
      },
    });
    return response.data as Workflow[];
  },

  getWorkflow: async (id: string) => {
    const response = await apiClient.get(`/bff/api/v1/workflows/${id}`);
    return response.data as Workflow;
  },

  create: async (data: CreateWorkflowInput) => {
    const response = await apiClient.post('/bff/api/v1/workflows', data);
    return response.data as Workflow;
  },

  update: async (id: string, data: UpdateWorkflowInput) => {
    const response = await apiClient.put(`/bff/api/v1/workflows/${id}`, data);
    return response.data as Workflow;
  },

  getVersions: async (id: string) => {
    const response = await apiClient.get(`/bff/api/v1/workflows/${id}/versions`);
    return response.data as WorkflowVersion[];
  },

  newVersion: async (id: string) => {
    const response = await apiClient.post(`/bff/api/v1/workflows/${id}/new-version`);
    return response.data as Workflow;
  },

  rollback: async (id: string) => {
    const response = await apiClient.put(`/bff/api/v1/workflows/${id}/rollback`);
    return response.data as Workflow;
  },

  activateWorkflow: async (id: string) => {
    const response = await apiClient.put(`/bff/api/v1/workflows/${id}/activate`);
    return response.data as Workflow;
  },

  executeWorkflow: async (id: string) => {
    const response = await apiClient.post(`/bff/api/v1/workflows/${id}/execute`);
    return response.data;
  },

  verifyWorkflow: async (id: string) => {
    const response = await apiClient.post(`/bff/api/v1/workflows/${id}/verify`);
    return response.data;
  },
};

// Task Mobile P1.8 — FR-232/UC-A7: handoff package type
export interface HandoffPackage {
  run_id: string;
  reason: string;
  conversation_context: string;
  evidence_count: number;
  entity_type?: string;
  entity_id?: string;
  created_at: string;
}

// Approval API
export const approvalApi = {
  getPendingApprovals: async (workspaceId: string) => {
    const response = await apiClient.get('/bff/api/v1/approvals', {
      params: { workspace_id: workspaceId },
    });
    return response.data as ApprovalRequest[];
  },

  decideApproval: async (id: string, decision: { decision: 'approve' | 'deny'; reason?: string }) => {
    const response = await apiClient.put(`/bff/api/v1/approvals/${id}`, decision);
    return response.data;
  },
};
