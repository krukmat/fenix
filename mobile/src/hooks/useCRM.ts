// Task 4.2 — FR-300: TanStack Query hooks para entidades CRM
// Task 4.3.td — STEP 8: list hooks migrated to useInfiniteQuery

import { useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { crmApi, agentApi } from '../services/api';
import { useAuthStore } from '../stores/authStore';

const PAGE_SIZE = 50;

// Query keys pattern (workspace isolation)
export const queryKeys = {
  accounts: (workspaceId: string) => ['accounts', workspaceId] as const,
  account: (workspaceId: string, id: string) => ['account', workspaceId, id] as const,
  contacts: (workspaceId: string) => ['contacts', workspaceId] as const,
  contact: (workspaceId: string, id: string) => ['contact', workspaceId, id] as const,
  deals: (workspaceId: string) => ['deals', workspaceId] as const,
  deal: (workspaceId: string, id: string) => ['deal', workspaceId, id] as const,
  cases: (workspaceId: string) => ['cases', workspaceId] as const,
  case: (workspaceId: string, id: string) => ['case', workspaceId, id] as const,
  agentRuns: (workspaceId: string) => ['agent-runs', workspaceId] as const,
  agentRun: (workspaceId: string, id: string) => ['agent-run', workspaceId, id] as const,
  agentDefinitions: (workspaceId: string) => ['agent-definitions', workspaceId] as const,
};

// Hook to get workspaceId from auth store - returns null if not available
function useWorkspaceId(): string | null {
  return useAuthStore((state) => state.workspaceId);
}

// Accounts
export function useAccounts() {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: queryKeys.accounts(workspaceId ?? ''),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      crmApi.getAccounts(workspaceId!, { page: pageParam, limit: PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage: { total?: number; data?: unknown[] }, allPages) => {
      const total = lastPage.total ?? 0;
      const loaded = allPages.flatMap((p) => p.data ?? []).length;
      return loaded < total ? allPages.length + 1 : undefined;
    },
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
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

  return useInfiniteQuery({
    queryKey: queryKeys.contacts(workspaceId ?? ''),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      crmApi.getContacts(workspaceId!, { page: pageParam, limit: PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage: { total?: number; data?: unknown[] }, allPages) => {
      const total = lastPage.total ?? 0;
      const loaded = allPages.flatMap((p) => p.data ?? []).length;
      return loaded < total ? allPages.length + 1 : undefined;
    },
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
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

  return useInfiniteQuery({
    queryKey: queryKeys.deals(workspaceId ?? ''),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      crmApi.getDeals(workspaceId!, { page: pageParam, limit: PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage: { total?: number; data?: unknown[] }, allPages) => {
      const total = lastPage.total ?? 0;
      const loaded = allPages.flatMap((p) => p.data ?? []).length;
      return loaded < total ? allPages.length + 1 : undefined;
    },
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
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

// Cases
export function useCases() {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: queryKeys.cases(workspaceId ?? ''),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      crmApi.getCases(workspaceId!, { page: pageParam, limit: PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage: { total?: number; data?: unknown[] }, allPages) => {
      const total = lastPage.total ?? 0;
      const loaded = allPages.flatMap((p) => p.data ?? []).length;
      return loaded < total ? allPages.length + 1 : undefined;
    },
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
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

// Agent Runs
export function useAgentRuns() {
  const workspaceId = useWorkspaceId();

  return useInfiniteQuery({
    queryKey: queryKeys.agentRuns(workspaceId ?? ''),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      agentApi.getRuns(workspaceId!, { page: pageParam, limit: PAGE_SIZE }),
    initialPageParam: 1,
    getNextPageParam: (lastPage: { total?: number; data?: unknown[] }, allPages) => {
      const total = lastPage.total ?? 0;
      const loaded = allPages.flatMap((p) => p.data ?? []).length;
      return loaded < total ? allPages.length + 1 : undefined;
    },
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
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
    queryFn: () => agentApi.getDefinitions(workspaceId!),
    staleTime: 5 * 60_000, // 5 minutes - definitions don't change often
    gcTime: 30 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}
