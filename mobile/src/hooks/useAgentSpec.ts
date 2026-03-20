import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  agentApi,
  approvalApi,
  signalApi,
  workflowApi,
  type AgentRunStatus,
  type CreateWorkflowInput,
  type SignalStatus,
  type UpdateWorkflowInput,
  type WorkflowStatus,
} from '../services/api';
import { useAuthStore } from '../stores/authStore';

const SIGNAL_PAGE_SIZE = 50;
const WORKFLOW_PAGE_SIZE = 50;
const AGENT_RUN_PAGE_SIZE = 25;

export const agentSpecQueryKeys = {
  signals: (workspaceId: string, filters?: { status?: SignalStatus; entity_type?: string; entity_id?: string }) =>
    ['signals', workspaceId, filters ?? {}] as const,
  signalsByEntity: (workspaceId: string, entityType: string, entityId: string) =>
    ['signals', workspaceId, { entity_type: entityType, entity_id: entityId }] as const,
  workflows: (workspaceId: string, filters?: { status?: WorkflowStatus }) =>
    ['workflows', workspaceId, filters ?? {}] as const,
  workflow: (workspaceId: string, id: string) => ['workflow', workspaceId, id] as const,
  workflowVersions: (workspaceId: string, id: string) => ['workflow-versions', workspaceId, id] as const,
  agentRunsByEntity: (
    workspaceId: string,
    entityType: string,
    entityId: string,
    filters?: { status?: AgentRunStatus; workflow_id?: string }
  ) => ['agent-runs', workspaceId, 'entity', entityType, entityId, filters ?? {}] as const,
  agentRunsByWorkflow: (
    workspaceId: string,
    workflowId: string,
    filters?: { status?: AgentRunStatus; entity_type?: string; entity_id?: string }
  ) => ['agent-runs', workspaceId, 'workflow', workflowId, filters ?? {}] as const,
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

export function useWorkflows(filters?: { status?: WorkflowStatus }) {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: agentSpecQueryKeys.workflows(workspaceId ?? '', filters),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      workflowApi.getWorkflows(workspaceId!, filters, { page: pageParam, limit: WORKFLOW_PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (_lastPage, allPages) => {
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

export function useWorkflowVersions(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: agentSpecQueryKeys.workflowVersions(workspaceId ?? '', id),
    queryFn: () => workflowApi.getVersions(id),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId && !!id,
  });
}

export function useCreateWorkflow() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateWorkflowInput) => workflowApi.create(input),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['workflows', workspaceId ?? ''] });
      if (result?.id) {
        queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflow(workspaceId ?? '', result.id) });
        queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflowVersions(workspaceId ?? '', result.id) });
      }
    },
  });
}

export function useUpdateWorkflow() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateWorkflowInput }) => workflowApi.update(id, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: ['workflows', workspaceId ?? ''] });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflow(workspaceId ?? '', variables.id) });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflowVersions(workspaceId ?? '', variables.id) });
    },
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
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflowVersions(workspaceId ?? '', id) });
    },
  });
}

export function useNewVersion() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => workflowApi.newVersion(id),
    onSuccess: (_result, id) => {
      queryClient.invalidateQueries({ queryKey: ['workflows', workspaceId ?? ''] });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflow(workspaceId ?? '', id) });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflowVersions(workspaceId ?? '', id) });
    },
  });
}

export function useRollback() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => workflowApi.rollback(id),
    onSuccess: (_result, id) => {
      queryClient.invalidateQueries({ queryKey: ['workflows', workspaceId ?? ''] });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflow(workspaceId ?? '', id) });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflowVersions(workspaceId ?? '', id) });
      queryClient.invalidateQueries({ queryKey: ['agent-runs', workspaceId ?? ''] });
    },
  });
}

export function useExecuteWorkflow() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => workflowApi.executeWorkflow(id),
    onSuccess: (_result, id) => {
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflow(workspaceId ?? '', id) });
      queryClient.invalidateQueries({ queryKey: agentSpecQueryKeys.workflowVersions(workspaceId ?? '', id) });
      queryClient.invalidateQueries({ queryKey: ['agent-runs', workspaceId ?? ''] });
    },
  });
}

export function useAgentRunsByEntity(
  entityType: string,
  entityId: string,
  filters?: { status?: AgentRunStatus; workflow_id?: string }
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

export function useAgentRunsByWorkflow(
  workflowId: string,
  filters?: { status?: AgentRunStatus; entity_type?: string; entity_id?: string }
) {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: agentSpecQueryKeys.agentRunsByWorkflow(workspaceId ?? '', workflowId, filters),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      agentApi.getRunsByWorkflow(workspaceId!, workflowId, { page: pageParam, limit: AGENT_RUN_PAGE_SIZE }, filters),
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
    enabled: !!workspaceId && !!workflowId,
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
    mutationFn: ({ id, decision }: { id: string; decision: { decision: 'approve' | 'deny'; reason?: string } }) =>
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
