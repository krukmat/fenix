// Task Mobile P1.2 — FR-301/302/071, UC-A4/A5/A6/A7: TanStack Query hooks for Signals, Workflows, Approvals

import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { signalApi, workflowApi, approvalApi, agentApi, type SignalStatus, type WorkflowStatus } from '../services/api';
import { useAuthStore } from '../stores/authStore';

const SIGNAL_PAGE_SIZE = 50;
const WORKFLOW_PAGE_SIZE = 50;

// Query keys (workspace-isolated)
export const agentSpecQueryKeys = {
  signals: (workspaceId: string, filters?: { status?: SignalStatus; entity_type?: string; entity_id?: string }) =>
    ['signals', workspaceId, filters ?? {}] as const,
  signalsByEntity: (workspaceId: string, entityType: string, entityId: string) =>
    ['signals', workspaceId, { entity_type: entityType, entity_id: entityId }] as const,
  workflows: (workspaceId: string, filters?: { status?: WorkflowStatus }) =>
    ['workflows', workspaceId, filters ?? {}] as const,
  workflow: (workspaceId: string, id: string) => ['workflow', workspaceId, id] as const,
  pendingApprovals: (workspaceId: string) => ['pending-approvals', workspaceId] as const,
  handoffPackage: (runId: string) => ['handoff-package', runId] as const,
};

function useWorkspaceId(): string | null {
  return useAuthStore((state) => state.workspaceId);
}

// ─── Signals ──────────────────────────────────────────────────────────────────

export function useSignals(filters?: { status?: SignalStatus; entity_type?: string; entity_id?: string }) {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: agentSpecQueryKeys.signals(workspaceId ?? '', filters),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      signalApi.getSignals(workspaceId!, filters, { page: pageParam, limit: SIGNAL_PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.flat().length;
      return loaded < SIGNAL_PAGE_SIZE * allPages.length ? undefined : allPages.length + 1;
    },
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}

export function useSignalsByEntity(entityType: string, entityId: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: agentSpecQueryKeys.signalsByEntity(workspaceId ?? '', entityType, entityId),
    queryFn: () => signalApi.getSignals(workspaceId!, { entity_type: entityType, entity_id: entityId }),
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId && !!entityType && !!entityId,
  });
}

export function useDismissSignal() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => signalApi.dismissSignal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signals', workspaceId ?? ''] });
    },
  });
}

// ─── Workflows ────────────────────────────────────────────────────────────────

export function useWorkflows(filters?: { status?: WorkflowStatus }) {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: agentSpecQueryKeys.workflows(workspaceId ?? '', filters),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      workflowApi.getWorkflows(workspaceId!, filters, { page: pageParam, limit: WORKFLOW_PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.flat().length;
      return loaded < WORKFLOW_PAGE_SIZE * allPages.length ? undefined : allPages.length + 1;
    },
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}

export function useWorkflow(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: agentSpecQueryKeys.workflow(workspaceId ?? '', id),
    queryFn: () => workflowApi.getWorkflow(id),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId && !!id,
  });
}

export function useActivateWorkflow() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => workflowApi.activateWorkflow(id),
    onSuccess: (_result, id) => {
      queryClient.invalidateQueries({ queryKey: ['workflows', workspaceId ?? ''] });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflow(workspaceId ?? '', id) });
    },
  });
}

export function useExecuteWorkflow() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => workflowApi.executeWorkflow(id),
    onSuccess: () => {
      // Invalidate agent runs — execution creates a new run
      queryClient.invalidateQueries({ queryKey: ['agent-runs', workspaceId ?? ''] });
    },
  });
}

// ─── Approvals ────────────────────────────────────────────────────────────────

export function usePendingApprovals() {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: agentSpecQueryKeys.pendingApprovals(workspaceId ?? ''),
    queryFn: () => approvalApi.getPendingApprovals(workspaceId!),
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}

export function useDecideApproval() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: { decision: 'approve' | 'deny'; reason?: string } }) =>
      approvalApi.decideApproval(id, decision),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.pendingApprovals(workspaceId ?? '') });
    },
  });
}

// ─── Handoff ──────────────────────────────────────────────────────────────────

// Task Mobile P1.8 — FR-232/UC-A7: fetch handoff package, only enabled when run is escalated
export function useHandoffPackage(runId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: agentSpecQueryKeys.handoffPackage(runId ?? ''),
    queryFn: () => agentApi.getHandoff(runId!),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!runId && enabled,
  });
}
