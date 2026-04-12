// Signal, Tool and Approval APIs — extracted from api.ts to keep it under 300 lines
import { apiClient } from './api.client';
import { normalizeInboxResponse } from './api.handoff';
import type {
  Signal,
  SignalStatus,
  ApprovalRequest,
  GovernanceSummary,
  AuditEvent,
  AuditFilters,
  PaginatedResponse,
  UsageEvent,
  UsageFilters,
} from './api.types';

function normalizeSignalsResponse(data: unknown): Signal[] {
  if (Array.isArray(data)) {
    return data as Signal[];
  }
  if (Array.isArray((data as { data?: unknown[] } | null)?.data)) {
    return (data as { data: Signal[] }).data;
  }
  return [];
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
    return normalizeSignalsResponse(response.data);
  },

  dismissSignal: async (id: string) => {
    const response = await apiClient.put(`/bff/api/v1/signals/${id}/dismiss`);
    return response.data as Signal;
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
// W1-T1: decisions use 'approve'/'reject' only — 'deny' is legacy and must not be sent
export const approvalApi = {
  getPendingApprovals: async (workspaceId: string) => {
    const response = await apiClient.get('/bff/api/v1/approvals', {
      params: { workspace_id: workspaceId },
    });
    return response.data as ApprovalRequest[];
  },

  // W1-T1: BFF alias routes POST /approvals/{id}/approve and /reject map to this handler
  approve: async (id: string, reason?: string) => {
    const response = await apiClient.post(`/bff/api/v1/approvals/${id}/approve`, { reason });
    return response.data;
  },

  reject: async (id: string, reason?: string) => {
    const response = await apiClient.post(`/bff/api/v1/approvals/${id}/reject`, { reason });
    return response.data;
  },

  // Legacy — kept for backwards compat while old screens are rewritten; do not use in new code
  decideApproval: async (id: string, decision: { decision: 'approve' | 'reject'; reason?: string }) => {
    const response = await apiClient.put(`/bff/api/v1/approvals/${id}`, decision);
    return response.data;
  },
};

// W1-T1: Inbox API — BFF aggregation route for approvals, handoffs, signals, and rejected runs
export const inboxApi = {
  getInbox: async (workspaceId: string) => {
    const response = await apiClient.get('/bff/api/v1/mobile/inbox', {
      params: { workspace_id: workspaceId },
    });
    return normalizeInboxResponse(response.data);
  },
};

// W1-T1: Governance API — BFF proxy for GET /api/v1/governance/summary
export const governanceApi = {
  getSummary: async (workspaceId: string) => {
    const response = await apiClient.get('/bff/api/v1/governance/summary', {
      params: { workspace_id: workspaceId },
    });
    return response.data as GovernanceSummary;
  },

  getAuditEvents: async (
    workspaceId: string,
    filters?: AuditFilters,
    pagination?: { page?: number; limit?: number }
  ) => {
    const limit = pagination?.limit ?? 20;
    const offset = ((pagination?.page ?? 1) - 1) * limit;
    const response = await apiClient.get('/bff/api/v1/audit/events', {
      params: { workspace_id: workspaceId, limit, offset, ...filters },
    });
    return response.data as PaginatedResponse<AuditEvent>;
  },

  getAuditEventById: async (workspaceId: string, id: string) => {
    const response = await apiClient.get(`/bff/api/v1/audit/events/${id}`, {
      params: { workspace_id: workspaceId },
    });
    return response.data as AuditEvent;
  },

  getUsageEvents: async (
    workspaceId: string,
    filters?: UsageFilters,
    pagination?: { page?: number; limit?: number }
  ) => {
    const page = pagination?.page ?? 1;
    const pageSize = pagination?.limit ?? 20;
    // /usage ignores offset, so loading "more" means requesting a larger limit.
    const limit = page * pageSize;
    const response = await apiClient.get('/bff/api/v1/usage', {
      params: { workspace_id: workspaceId, limit, ...filters },
    });
    return response.data as PaginatedResponse<UsageEvent>;
  },
};
