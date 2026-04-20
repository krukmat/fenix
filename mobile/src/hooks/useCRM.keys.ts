import { useAuthStore } from '../stores/authStore';

export const queryKeys = {
  accounts: (workspaceId: string) => ['accounts', workspaceId] as const,
  account: (workspaceId: string, id: string) => ['account', workspaceId, id] as const,
  contacts: (workspaceId: string) => ['contacts', workspaceId] as const,
  contact: (workspaceId: string, id: string) => ['contact', workspaceId, id] as const,
  accountContacts: (workspaceId: string, accountId: string) => ['account-contacts', workspaceId, accountId] as const,
  deals: (workspaceId: string) => ['deals', workspaceId] as const,
  deal: (workspaceId: string, id: string) => ['deal', workspaceId, id] as const,
  leads: (workspaceId: string) => ['leads', workspaceId] as const,
  lead: (workspaceId: string, id: string) => ['lead', workspaceId, id] as const,
  cases: (workspaceId: string) => ['cases', workspaceId] as const,
  case: (workspaceId: string, id: string) => ['case', workspaceId, id] as const,
  pipelines: (workspaceId: string) => ['pipelines', workspaceId] as const,
  pipeline: (workspaceId: string, id: string) => ['pipeline', workspaceId, id] as const,
  pipelineStages: (workspaceId: string, pipelineId: string) => ['pipeline-stages', workspaceId, pipelineId] as const,
  activities: (workspaceId: string) => ['activities', workspaceId] as const,
  activity: (workspaceId: string, id: string) => ['activity', workspaceId, id] as const,
  notes: (workspaceId: string) => ['notes', workspaceId] as const,
  note: (workspaceId: string, id: string) => ['note', workspaceId, id] as const,
  attachments: (workspaceId: string) => ['attachments', workspaceId] as const,
  attachment: (workspaceId: string, id: string) => ['attachment', workspaceId, id] as const,
  timeline: (workspaceId: string) => ['timeline', workspaceId] as const,
  entityTimeline: (workspaceId: string, entityType: string, entityId: string) =>
    ['timeline', workspaceId, entityType, entityId] as const,
  agentRuns: (workspaceId: string) => ['agent-runs', workspaceId] as const,
  agentRun: (workspaceId: string, id: string) => ['agent-run', workspaceId, id] as const,
  agentDefinitions: (workspaceId: string) => ['agent-definitions', workspaceId] as const,
};

export function useWorkspaceId(): string | null {
  return useAuthStore((state) => state.workspaceId);
}
