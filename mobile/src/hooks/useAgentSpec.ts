import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  agentApi,
  approvalApi,
  signalApi,
  type AgentRunPublicStatus,
  type SignalStatus,
} from '../services/api';
import { useAuthStore } from '../stores/authStore';

const SIGNAL_PAGE_SIZE = 50;
const AGENT_RUN_PAGE_SIZE = 25;

export const agentSpecQueryKeys = {
  signals: (workspaceId: string, filters?: { status?: SignalStatus; entity_type?: string; entity_id?: string }) =>
    ['signals', workspaceId, filters ?? {}] as const,
  signalsByEntity: (workspaceId: string, entityType: string, entityId: string) =>
    ['signals', workspaceId, { entity_type: entityType, entity_id: entityId }] as const,
  agentRunsByEntity: (
    workspaceId: string,
    entityType: string,
    entityId: string,
    filters?: { status?: AgentRunPublicStatus }
  ) => ['agent-runs', workspaceId, 'entity', entityType, entityId, filters ?? {}] as const,
  pendingApprovals: (workspaceId: string) => ['pending-approvals', workspaceId] as const,
  handoffPackage: (runId: string) => ['handoff-package', runId] as const,
};

function useWorkspaceId(): string | null {
  return useAuthStore((state) => state.workspaceId);
}

export function useSignals(filters?: { status?: SignalStatus; entity_type?: string; entity_id?: string }) {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: agentSpecQueryKeys.signals(workspaceId ?? '', filters),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      signalApi.getSignals(workspaceId!, filters, { page: pageParam, limit: SIGNAL_PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (_lastPage, allPages) => {
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

export function useAgentRunsByEntity(
  entityType: string,
  entityId: string,
  filters?: { status?: AgentRunPublicStatus }
) {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: agentSpecQueryKeys.agentRunsByEntity(workspaceId ?? '', entityType, entityId, filters),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      agentApi.getRunsByEntity(workspaceId!, entityType, entityId, { page: pageParam, limit: AGENT_RUN_PAGE_SIZE }, filters),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) => {
      const total = lastPage.meta?.total ?? 0;
      const loaded = allPages.flatMap((page) => page.data ?? []).length;
      return loaded < total ? allPages.length + 1 : undefined;
    },
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId && !!entityType && !!entityId,
  });
}

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
    // W1-T1: 'deny' replaced by 'reject' per normalized approval contract
    mutationFn: ({ id, decision }: { id: string; decision: { decision: 'approve' | 'reject'; reason?: string } }) =>
      approvalApi.decideApproval(id, decision),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.pendingApprovals(workspaceId ?? '') });
    },
  });
}

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
