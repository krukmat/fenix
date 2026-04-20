// Task 4.2 — FR-300: TanStack Query hooks para entidades CRM
// Task 4.3.td — STEP 8: list hooks migrated to useInfiniteQuery

import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { crmApi, agentApi } from '../services/api';
import { queryKeys, useWorkspaceId } from './useCRM.keys';

const PAGE_SIZE = 50;

export { queryKeys } from './useCRM.keys';
export * from './useCRM.full';
export * from './useCRM.entityMutations';

function useInfiniteWorkspaceList<TPage extends { total?: number; data?: unknown[] }>(
  buildQueryKey: (workspaceId: string) => readonly unknown[],
  workspaceId: string | null,
  fetchPage: (workspaceId: string, page: number) => Promise<TPage>,
  staleTime = 30_000,
) {
  return useInfiniteQuery({
    queryKey: buildQueryKey(workspaceId ?? ''),
    queryFn: ({ pageParam }: { pageParam: number }) => fetchPage(workspaceId ?? '', pageParam),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) => {
      const total = lastPage.total ?? 0;
      const loaded = allPages.flatMap((p) => p.data ?? []).length;
      return loaded < total ? allPages.length + 1 : undefined;
    },
    staleTime,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}

// Accounts
export function useAccounts() {
  const workspaceId = useWorkspaceId();

  return useInfiniteWorkspaceList(
    queryKeys.accounts,
    workspaceId,
    (ws, page) => crmApi.getAccounts(ws, { page, limit: PAGE_SIZE }),
  );
}

export function useAccount(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: queryKeys.account(workspaceId ?? '', id),
    queryFn: () => crmApi.getAccountFull(id),
    staleTime: 60_000, // 60s for details
    enabled: !!workspaceId && !!id,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

// Contacts
export function useContacts() {
  const workspaceId = useWorkspaceId();

  return useInfiniteWorkspaceList(
    queryKeys.contacts,
    workspaceId,
    (ws, page) => crmApi.getContacts(ws, { page, limit: PAGE_SIZE }),
  );
}

export function useContact(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: queryKeys.contact(workspaceId ?? '', id),
    queryFn: () => crmApi.getContact(id),
    staleTime: 60_000,
    enabled: !!workspaceId && !!id,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

// Deals
export function useDeals() {
  const workspaceId = useWorkspaceId();

  return useInfiniteWorkspaceList(
    queryKeys.deals,
    workspaceId,
    (ws, page) => crmApi.getDeals(ws, { page, limit: PAGE_SIZE }),
  );
}

export function useDeal(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: queryKeys.deal(workspaceId ?? '', id),
    queryFn: () => crmApi.getDealFull(id),
    staleTime: 60_000,
    enabled: !!workspaceId && !!id,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

// Leads
export function useLeads() {
  const workspaceId = useWorkspaceId();

  return useInfiniteWorkspaceList(
    queryKeys.leads,
    workspaceId,
    (ws, page) => crmApi.getLeads(ws, { page, limit: PAGE_SIZE }),
  );
}

export function useLead(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: queryKeys.lead(workspaceId ?? '', id),
    queryFn: () => crmApi.getLead(id),
    staleTime: 60_000,
    enabled: !!workspaceId && !!id,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

// Cases
export function useCases() {
  const workspaceId = useWorkspaceId();

  return useInfiniteWorkspaceList(
    queryKeys.cases,
    workspaceId,
    (ws, page) => crmApi.getCases(ws, { page, limit: PAGE_SIZE }),
  );
}

export function useCase(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: queryKeys.case(workspaceId ?? '', id),
    queryFn: () => crmApi.getCaseFull(id),
    staleTime: 60_000,
    enabled: !!workspaceId && !!id,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

// Deal mutations
export function useCreateDeal() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: crmApi.createDeal,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.deals(workspaceId ?? '') });
    },
  });
}

export function useUpdateDeal() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof crmApi.updateDeal>[1] }) => crmApi.updateDeal(id, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.deals(workspaceId ?? '') });
      queryClient.invalidateQueries({ queryKey: queryKeys.deal(workspaceId ?? '', variables.id) });
    },
  });
}

// Case mutations
export function useCreateCase() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: crmApi.createCase,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.cases(workspaceId ?? '') });
    },
  });
}

export function useUpdateCase() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof crmApi.updateCase>[1] }) => crmApi.updateCase(id, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.cases(workspaceId ?? '') });
      queryClient.invalidateQueries({ queryKey: queryKeys.case(workspaceId ?? '', variables.id) });
    },
  });
}

// Agent Runs
export function useAgentRuns() {
  const workspaceId = useWorkspaceId();

  return useInfiniteWorkspaceList(
    queryKeys.agentRuns,
    workspaceId,
    (ws, page) => agentApi.getRuns(ws, { page, limit: PAGE_SIZE }),
    15_000,
  );
}

export function useAgentRun(id: string) {
  const workspaceId = useWorkspaceId();

  return useQuery({
    queryKey: queryKeys.agentRun(workspaceId ?? '', id),
    queryFn: () => agentApi.getRun(id),
    staleTime: 15_000,
    enabled: !!workspaceId && !!id,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

// Agent Definitions
export function useAgentDefinitions() {
  const workspaceId = useWorkspaceId();
  return useQuery({
    queryKey: queryKeys.agentDefinitions(workspaceId ?? ''),
    queryFn: () => agentApi.getDefinitions(workspaceId ?? ''),
    staleTime: 5 * 60_000, // 5 minutes - definitions don't change often
    gcTime: 30 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}
