import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { crmApi } from '../services/api';
import { queryKeys, useWorkspaceId } from './useCRM.keys';

const PAGE_SIZE = 50;
const DETAIL_STALE_TIME = 60_000;
const LIST_STALE_TIME = 30_000;

function useWorkspaceList<TPage extends { total?: number; data?: unknown[] }>(
  buildKey: (workspaceId: string) => readonly unknown[],
  fetchPage: (workspaceId: string, page: number) => Promise<TPage>,
) {
  const workspaceId = useWorkspaceId();
  return useInfiniteQuery({
    queryKey: buildKey(workspaceId ?? ''),
    queryFn: ({ pageParam }: { pageParam: number }) => fetchPage(workspaceId ?? '', pageParam),
    initialPageParam: 1,
    getNextPageParam: (lastPage, pages) => {
      const total = lastPage.total ?? 0;
      const loaded = pages.flatMap((p) => p.data ?? []).length;
      return loaded < total ? pages.length + 1 : undefined;
    },
    staleTime: LIST_STALE_TIME,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}

function invalidateMany(queryClient: ReturnType<typeof useQueryClient>, keys: readonly (readonly unknown[])[]) {
  keys.forEach((queryKey) => queryClient.invalidateQueries({ queryKey }));
}

function useDetailQuery(key: readonly unknown[], enabled: boolean, queryFn: () => Promise<unknown>) {
  return useQuery({
    queryKey: key,
    queryFn,
    staleTime: DETAIL_STALE_TIME,
    enabled,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

export function usePipelines() {
  return useWorkspaceList(queryKeys.pipelines, (ws, page) => crmApi.getPipelines(ws, { page, limit: PAGE_SIZE }));
}

export function usePipeline(id: string) {
  const workspaceId = useWorkspaceId();
  return useDetailQuery(queryKeys.pipeline(workspaceId ?? '', id), !!workspaceId && !!id, () => crmApi.getPipeline(id));
}

export function usePipelineStages(pipelineId: string) {
  const workspaceId = useWorkspaceId();
  return useQuery({
    queryKey: queryKeys.pipelineStages(workspaceId ?? '', pipelineId),
    queryFn: () => crmApi.getPipelineStages(pipelineId),
    staleTime: DETAIL_STALE_TIME,
    enabled: !!workspaceId && !!pipelineId,
    retry: 1,
    refetchOnWindowFocus: false,
  });
}

export function useActivities() {
  return useWorkspaceList(queryKeys.activities, (ws, page) =>
    crmApi.getActivities(ws, { limit: PAGE_SIZE, offset: (page - 1) * PAGE_SIZE }));
}

export function useActivity(id: string) {
  const workspaceId = useWorkspaceId();
  return useDetailQuery(queryKeys.activity(workspaceId ?? '', id), !!workspaceId && !!id, () => crmApi.getActivity(id));
}

export function useNotes() {
  return useWorkspaceList(queryKeys.notes, (ws, page) =>
    crmApi.getNotes(ws, { limit: PAGE_SIZE, offset: (page - 1) * PAGE_SIZE }));
}

export function useNote(id: string) {
  const workspaceId = useWorkspaceId();
  return useDetailQuery(queryKeys.note(workspaceId ?? '', id), !!workspaceId && !!id, () => crmApi.getNote(id));
}

export function useAttachments() {
  return useWorkspaceList(queryKeys.attachments, (ws, page) =>
    crmApi.getAttachments(ws, { limit: PAGE_SIZE, offset: (page - 1) * PAGE_SIZE }));
}

export function useAttachment(id: string) {
  const workspaceId = useWorkspaceId();
  return useDetailQuery(queryKeys.attachment(workspaceId ?? '', id), !!workspaceId && !!id, () => crmApi.getAttachment(id));
}

export function useTimeline() {
  return useWorkspaceList(queryKeys.timeline, (ws, page) =>
    crmApi.getTimeline(ws, { limit: PAGE_SIZE, offset: (page - 1) * PAGE_SIZE }));
}

export function useEntityTimeline(entityType: string, entityId: string) {
  const workspaceId = useWorkspaceId();
  return useDetailQuery(
    queryKeys.entityTimeline(workspaceId ?? '', entityType, entityId),
    !!workspaceId && !!entityType && !!entityId,
    () => crmApi.getTimelineByEntity(entityType, entityId),
  );
}

export function useCreateAccount() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.createAccount,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.accounts(workspaceId ?? '') }),
  });
}

export function useUpdateAccount() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof crmApi.updateAccount>[1] }) =>
      crmApi.updateAccount(id, data),
    onSuccess: (_result, vars) =>
      invalidateMany(queryClient, [queryKeys.accounts(workspaceId ?? ''), queryKeys.account(workspaceId ?? '', vars.id)]),
  });
}

export function useDeleteAccount() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteAccount,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.accounts(workspaceId ?? ''), queryKeys.account(workspaceId ?? '', id)]),
  });
}

export function useCreateContact() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.createContact,
    onSuccess: (_result, data) => {
      const keys: (readonly unknown[])[] = [queryKeys.contacts(workspaceId ?? '')];
      if (data.accountId) {
        keys.push(queryKeys.accountContacts(workspaceId ?? '', data.accountId));
      }
      invalidateMany(queryClient, keys);
    },
  });
}

export function useUpdateContact() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof crmApi.updateContact>[1] }) =>
      crmApi.updateContact(id, data),
    onSuccess: (_result, vars) =>
      invalidateMany(queryClient, [queryKeys.contacts(workspaceId ?? ''), queryKeys.contact(workspaceId ?? '', vars.id)]),
  });
}

export function useDeleteContact() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteContact,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.contacts(workspaceId ?? ''), queryKeys.contact(workspaceId ?? '', id)]),
  });
}

export function useCreateLead() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.createLead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.leads(workspaceId ?? '') }),
  });
}

export function useUpdateLead() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof crmApi.updateLead>[1] }) => crmApi.updateLead(id, data),
    onSuccess: (_result, vars) =>
      invalidateMany(queryClient, [queryKeys.leads(workspaceId ?? ''), queryKeys.lead(workspaceId ?? '', vars.id)]),
  });
}

export function useDeleteLead() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteLead,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.leads(workspaceId ?? ''), queryKeys.lead(workspaceId ?? '', id)]),
  });
}

export function useDeleteDeal() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteDeal,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.deals(workspaceId ?? ''), queryKeys.deal(workspaceId ?? '', id)]),
  });
}

export function useDeleteCase() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteCase,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.cases(workspaceId ?? ''), queryKeys.case(workspaceId ?? '', id)]),
  });
}
