// Signal, Workflow, Tool and Approval APIs — extracted from api.ts to keep it under 300 lines
import { apiClient } from './api.client';
import type {
  Signal,
  SignalStatus,
  Workflow,
  WorkflowStatus,
  WorkflowVersion,
  CreateWorkflowInput,
  UpdateWorkflowInput,
  ApprovalRequest,
} from './api.types';

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

// Tool API
export const toolApi = {
  execute: async (tool: string, params: Record<string, unknown>) => {
    const response = await apiClient.post(`/bff/api/v1/tools/${tool}`, params);
    return response.data;
  },
};

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
