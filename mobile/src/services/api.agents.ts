// Agent API — extracted from api.ts to keep it under 300 lines
// W1-T1: uses AgentRunPublicStatus for public filters
import { apiClient } from './api.client';
import { normalizeHandoffPackage } from './api.handoff';
import type {
  AgentRunPublicStatus,
  PaginatedResponse,
  QueuedAgentTriggerResponse,
  SalesBrief,
  UsageEvent,
} from './api.types';

function normalizeQueuedTriggerResponse(data: unknown): QueuedAgentTriggerResponse {
  const payload = (data ?? null) as Record<string, unknown> | null;
  return {
    runId: String(payload?.run_id ?? ''),
    status: String(payload?.status ?? ''),
    agent: String(payload?.agent ?? ''),
  };
}

export const agentApi = {
  // W1-T1: status filter accepts public outcomes for user-facing lists
  getRuns: async (
    workspaceId: string,
    pagination?: { page?: number; limit?: number },
    filters?: { status?: AgentRunPublicStatus; entity_type?: string; entity_id?: string }
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
    filters?: { status?: AgentRunPublicStatus }
  ) => {
    return agentApi.getRuns(workspaceId, pagination, {
      ...filters,
      entity_type: entityType,
      entity_id: entityId,
    });
  },

  // W1-T1: filter by normalized public status
  getRunsByStatus: async (
    workspaceId: string,
    status: AgentRunPublicStatus,
    pagination?: { page?: number; limit?: number },
    filters?: { entity_type?: string; entity_id?: string }
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

  // W1-T1: support-specific trigger — resolves the support agent by agentType internally
  triggerSupportRun: async (context: { entity_type: string; entity_id: string }) => {
    const response = await apiClient.post('/bff/api/v1/agents/support/trigger', context);
    return response.data;
  },

  triggerProspectingRun: async (context: { lead_id: string; language?: string }) => {
    const response = await apiClient.post('/bff/api/v1/agents/prospecting/trigger', context);
    return normalizeQueuedTriggerResponse(response.data);
  },

  triggerKBRun: async (context: { case_id: string; language?: string }) => {
    const response = await apiClient.post('/bff/api/v1/agents/kb/trigger', context);
    return normalizeQueuedTriggerResponse(response.data);
  },

  triggerInsightsRun: async (context: {
    query: string;
    date_from?: string;
    date_to?: string;
    language?: string;
  }) => {
    const response = await apiClient.post('/bff/api/v1/agents/insights/trigger', context);
    return normalizeQueuedTriggerResponse(response.data);
  },

  triggerDealRiskRun: async (context: { deal_id: string; language?: string }) => {
    const response = await apiClient.post('/bff/api/v1/agents/deal-risk/trigger', context);
    return normalizeQueuedTriggerResponse(response.data);
  },

  // Task Mobile P1.8 — FR-232/UC-A7: handoff package for escalated runs
  getHandoff: async (runId: string, caseId?: string) => {
    const response = await apiClient.get(`/bff/api/v1/agents/runs/${runId}/handoff`, {
      params: caseId ? { case_id: caseId } : undefined,
    });
    return normalizeHandoffPackage(response.data, runId);
  },

  // W1-T1: per-run usage events for activity detail diagnostics
  getRunUsage: async (runId: string) => {
    const response = await apiClient.get(`/bff/api/v1/usage`, {
      params: { run_id: runId, limit: 100 },
    });
    return (response.data as PaginatedResponse<UsageEvent>).data ?? [];
  },
};

// W1-T1: Sales Brief API — dedicated contract for POST /api/v1/copilot/sales-brief
// LLM-backed endpoint: timeout extended to 90s to accommodate local Ollama latency (~35s observed)
export const salesBriefApi = {
  getSalesBrief: async (entityType: string, entityId: string) => {
    const response = await apiClient.post(
      '/bff/api/v1/copilot/sales-brief',
      { entityType, entityId },
      { timeout: 90000 },
    );
    return response.data as SalesBrief;
  },
};
