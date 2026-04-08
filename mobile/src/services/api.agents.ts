// Agent API — extracted from api.ts to keep it under 300 lines
// W1-T1: uses AgentRunPublicStatus for public filters
import { apiClient } from './api.client';
import { normalizeHandoffPackage } from './api.handoff';
import type { AgentRunPublicStatus, SalesBrief, UsageEvent } from './api.types';

const isE2E = process.env.EXPO_PUBLIC_E2E_MODE === '1';

function buildE2ESalesBriefMock(entityType: string, entityId: string): SalesBrief {
  if (entityType === 'account') {
    return {
      outcome: 'abstained',
      entityType,
      entityId,
      confidence: 'low',
      abstentionReason: 'insufficient_evidence',
      summary: 'No grounded sales brief available for this account in E2E mode.',
      risks: [],
      nextBestActions: [],
      evidencePack: {
        schema_version: 'v1',
        query: `entity_type:${entityType} entity_id:${entityId} latest updates timeline next steps`,
        sources: [],
        source_count: 0,
        dedup_count: 0,
        filtered_count: 0,
        confidence: 'low',
        warnings: ['e2e sales brief mock'],
        retrieval_methods_used: [],
        built_at: new Date().toISOString(),
      },
    };
  }

  return {
    outcome: 'completed',
    entityType,
    entityId,
    confidence: 'medium',
    summary: `Grounded ${entityType} summary ready for ${entityId}.`,
    risks: [
      'Legal review could slip by three business days.',
      'Procurement needs revised pricing language.',
    ],
    nextBestActions: [
      {
        title: 'Send Security Addendum',
        description: 'Send the requested security addendum to Procurement today.',
        tool: 'create_task',
        params: { entity_type: entityType, entity_id: entityId },
        confidence_score: 0.85,
        confidence_level: 'high',
      },
      {
        title: 'Follow up with Procurement',
        description: 'Follow up with Procurement tomorrow regarding the revised pricing language.',
        tool: 'create_task',
        params: { entity_type: entityType, entity_id: entityId },
        confidence_score: 0.85,
        confidence_level: 'high',
      },
    ],
    evidencePack: {
      schema_version: 'v1',
      query: `entity_type:${entityType} entity_id:${entityId} latest updates timeline next steps`,
      sources: [
        {
          id: `${entityType}-${entityId}-mock-source`,
          snippet: 'E2E sales brief mock evidence source.',
          score: 0.9,
          timestamp: new Date().toISOString(),
          source_type: 'e2e_mock',
        },
      ],
      source_count: 1,
      dedup_count: 0,
      filtered_count: 0,
      confidence: 'medium',
      warnings: ['e2e sales brief mock'],
      retrieval_methods_used: ['e2e_mock'],
      built_at: new Date().toISOString(),
    },
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
      params: { run_id: runId },
    });
    return response.data as UsageEvent[];
  },
};

// W1-T1: Sales Brief API — dedicated contract for POST /api/v1/copilot/sales-brief
export const salesBriefApi = {
  getSalesBrief: async (entityType: string, entityId: string) => {
    if (isE2E) {
      return buildE2ESalesBriefMock(entityType, entityId);
    }

    const response = await apiClient.post('/bff/api/v1/copilot/sales-brief', {
      entityType,
      entityId,
    });
    return response.data as SalesBrief;
  },
};
