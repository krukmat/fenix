// Agent API — extracted from api.ts to keep it under 300 lines
import { apiClient } from './api.client';
import type { AgentRunStatus, HandoffPackage } from './api.types';

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
