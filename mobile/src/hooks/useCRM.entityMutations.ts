import { useMutation, useQueryClient } from '@tanstack/react-query';
import { crmApi } from '../services/api';
import { queryKeys, useWorkspaceId } from './useCRM.keys';

function invalidateMany(queryClient: ReturnType<typeof useQueryClient>, keys: readonly (readonly unknown[])[]) {
  keys.forEach((queryKey) => queryClient.invalidateQueries({ queryKey }));
}

function entityTimelineKey(workspaceId: string | null, entityType?: string, entityId?: string) {
  return entityType && entityId ? [queryKeys.entityTimeline(workspaceId ?? '', entityType, entityId)] : [];
}

export function useCreateActivity() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.createActivity,
    onSuccess: (_result, data) =>
      invalidateMany(queryClient, [queryKeys.activities(workspaceId ?? ''), ...entityTimelineKey(workspaceId, data.entityType, data.entityId)]),
  });
}

export function useUpdateActivity() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof crmApi.updateActivity>[1] }) =>
      crmApi.updateActivity(id, data),
    onSuccess: (_result, vars) =>
      invalidateMany(queryClient, [queryKeys.activities(workspaceId ?? ''), queryKeys.activity(workspaceId ?? '', vars.id)]),
  });
}

export function useDeleteActivity() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteActivity,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.activities(workspaceId ?? ''), queryKeys.activity(workspaceId ?? '', id)]),
  });
}

export function useCreateNote() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.createNote,
    onSuccess: (_result, data) =>
      invalidateMany(queryClient, [queryKeys.notes(workspaceId ?? ''), ...entityTimelineKey(workspaceId, data.entityType, data.entityId)]),
  });
}

export function useUpdateNote() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof crmApi.updateNote>[1] }) => crmApi.updateNote(id, data),
    onSuccess: (_result, vars) =>
      invalidateMany(queryClient, [queryKeys.notes(workspaceId ?? ''), queryKeys.note(workspaceId ?? '', vars.id)]),
  });
}

export function useDeleteNote() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteNote,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.notes(workspaceId ?? ''), queryKeys.note(workspaceId ?? '', id)]),
  });
}

export function useCreateAttachment() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.createAttachment,
    onSuccess: (_result, data) =>
      invalidateMany(queryClient, [queryKeys.attachments(workspaceId ?? ''), ...entityTimelineKey(workspaceId, data.entityType, data.entityId)]),
  });
}

export function useDeleteAttachment() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: crmApi.deleteAttachment,
    onSuccess: (_result, id) =>
      invalidateMany(queryClient, [queryKeys.attachments(workspaceId ?? ''), queryKeys.attachment(workspaceId ?? '', id)]),
  });
}
