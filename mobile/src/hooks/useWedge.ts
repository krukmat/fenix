// W1-T6 (mobile_wedge_harmonization_plan): mobile service and hook layer for wedge surfaces
// Covers: inbox, approval aliases, sales brief, activity run usage, governance summary
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  inboxApi,
  approvalApi,
  salesBriefApi,
  agentApi,
  governanceApi,
  type AgentRunPublicStatus,
  type AuditFilters,
  type UsageFilters,
} from '../services/api';
import { useAuthStore } from '../stores/authStore';

function useWorkspaceId(): string | null {
  return useAuthStore((state) => state.workspaceId);
}

// ─── Query keys ───────────────────────────────────────────────────────────────

export const wedgeQueryKeys = {
  inbox: (workspaceId: string) => ['inbox', workspaceId] as const,
  salesBrief: (entityType: string, entityId: string) => ['sales-brief', entityType, entityId] as const,
  runUsage: (runId: string) => ['run-usage', runId] as const,
  governanceSummary: (workspaceId: string) => ['governance-summary', workspaceId] as const,
  auditEvents: (workspaceId: string, filters?: AuditFilters, page?: number) =>
    ['audit-events', workspaceId, filters ?? {}, page ?? 1] as const,
  usageEvents: (workspaceId: string, filters?: UsageFilters, page?: number) =>
    ['usage-events', workspaceId, filters ?? {}, page ?? 1] as const,
  agentRuns: (workspaceId: string, filters?: { status?: AgentRunPublicStatus }) =>
    ['agent-runs', workspaceId, filters ?? {}] as const,
};

// ─── Inbox ────────────────────────────────────────────────────────────────────

/** Fetches unified inbox: approvals + handoffs + signals + rejected runs. Stale after 15s. */
export function useInbox() {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: wedgeQueryKeys.inbox(workspaceId ?? ''),
    queryFn: () => inboxApi.getInbox(workspaceId!),
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: true,
    enabled: !!workspaceId,
  });
}

// ─── Approval aliases ────────────────────────────────────────────────────────

/** Approve an approval request. Invalidates inbox and pending approvals on success. */
export function useApproveApproval() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      approvalApi.approve(id, reason),
    onSuccess: () => {
      if (workspaceId) {
        queryClient.invalidateQueries({ queryKey: wedgeQueryKeys.inbox(workspaceId) });
        queryClient.invalidateQueries({ queryKey: ['pending-approvals', workspaceId] });
      }
    },
  });
}

/** Reject an approval request. Invalidates inbox and pending approvals on success. */
export function useRejectApproval() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      approvalApi.reject(id, reason),
    onSuccess: () => {
      if (workspaceId) {
        queryClient.invalidateQueries({ queryKey: wedgeQueryKeys.inbox(workspaceId) });
        queryClient.invalidateQueries({ queryKey: ['pending-approvals', workspaceId] });
      }
    },
  });
}

// ─── Sales Brief ─────────────────────────────────────────────────────────────

/** Fetches a sales brief for an account or deal. Stale after 5 minutes (expensive call). */
export function useSalesBrief(entityType: string, entityId: string, enabled = true) {
  return useQuery({
    queryKey: wedgeQueryKeys.salesBrief(entityType, entityId),
    queryFn: () => salesBriefApi.getSalesBrief(entityType, entityId),
    staleTime: 5 * 60_000,
    gcTime: 10 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: enabled && !!entityType && !!entityId,
  });
}

// ─── Activity run usage ───────────────────────────────────────────────────────

/** Fetches per-run usage events for the activity detail diagnostics section. */
export function useRunUsage(runId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: wedgeQueryKeys.runUsage(runId ?? ''),
    queryFn: () => agentApi.getRunUsage(runId!),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: enabled && !!runId,
  });
}

// ─── Agent runs list (activity log) ──────────────────────────────────────────

/** Fetches agent runs filtered by normalized public status. */
export function useAgentRuns(filters?: { status?: AgentRunPublicStatus }) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: wedgeQueryKeys.agentRuns(workspaceId ?? '', filters),
    queryFn: () => agentApi.getRuns(workspaceId!, { limit: 50 }, filters),
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: true,
    enabled: !!workspaceId,
  });
}

// ─── Support agent trigger ────────────────────────────────────────────────────

/** Triggers the support agent for a case. Invalidates agent runs on success. */
export function useTriggerSupportAgent() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ entityType, entityId }: { entityType: string; entityId: string }) =>
      agentApi.triggerSupportRun({ entity_type: entityType, entity_id: entityId }),
    onSuccess: () => {
      if (workspaceId) {
        queryClient.invalidateQueries({ queryKey: wedgeQueryKeys.agentRuns(workspaceId) });
      }
    },
  });
}

// ─── Governance ───────────────────────────────────────────────────────────────

/** Fetches governance summary: recent usage + enriched quota states. Stale after 60s. */
export function useGovernanceSummary() {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: wedgeQueryKeys.governanceSummary(workspaceId ?? ''),
    queryFn: () => governanceApi.getSummary(workspaceId!),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}

/** Fetches paginated audit events with server-side filters. */
export function useAuditEvents(filters?: AuditFilters, page = 1) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: wedgeQueryKeys.auditEvents(workspaceId ?? '', filters, page),
    queryFn: () => governanceApi.getAuditEvents(workspaceId!, filters, { page, limit: 20 }),
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}

/** Fetches usage events for governance drilldown. "Page" increases requested limit because /usage has no offset. */
export function useUsageEvents(filters?: UsageFilters, page = 1) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: wedgeQueryKeys.usageEvents(workspaceId ?? '', filters, page),
    queryFn: () => governanceApi.getUsageEvents(workspaceId!, filters, { page, limit: 20 }),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}
