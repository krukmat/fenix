import { apiClient } from './api.client';

type PageParams = { page?: number; limit?: number };
type OffsetParams = { limit?: number; offset?: number };
type CRMMetadata = string | Record<string, unknown>;
type EntityRef = { entityType: string; entityId: string };

type AccountBody = {
  name?: string;
  industry?: string;
  website?: string;
  phone?: string;
  email?: string;
  description?: string;
  ownerId?: string;
};

type ContactBody = {
  accountId?: string;
  firstName?: string;
  lastName?: string;
  email?: string;
  phone?: string;
  title?: string;
  ownerId?: string;
};

type LeadBody = {
  accountId?: string;
  contactId?: string;
  ownerId?: string;
  source?: string;
  status?: string;
  score?: number;
  metadata?: CRMMetadata;
};

type PipelineBody = {
  name?: string;
  entityType?: string;
  isDefault?: boolean;
};

type StageBody = {
  name?: string;
  position?: number;
  probability?: number;
};

type ActivityBody = EntityRef & {
  ownerId?: string;
  assignedTo?: string;
  activityType?: string;
  body?: string;
  type?: string;
  subject?: string;
  description?: string;
  status?: string;
  dueAt?: string;
  completedAt?: string;
  metadata?: CRMMetadata;
};

type NoteBody = EntityRef & {
  authorId?: string;
  content?: string;
  isInternal?: boolean;
  metadata?: CRMMetadata;
};

type AttachmentBody = EntityRef & {
  uploaderId?: string;
  filename?: string;
  fileName?: string;
  storagePath?: string;
  contentType?: string;
  sizeBytes?: number;
  metadata?: CRMMetadata;
};

type WireAttachmentBody = Omit<AttachmentBody, 'fileName'>;

function pageParams(workspaceId: string, pagination?: PageParams) {
  return { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 };
}

function offsetParams(workspaceId: string, pagination?: OffsetParams) {
  return { workspace_id: workspaceId, limit: pagination?.limit ?? 50, offset: pagination?.offset ?? 0 };
}

async function getData(path: string, params?: Record<string, string | number | undefined>) {
  const response = await apiClient.get(path, params ? { params } : undefined);
  return response.data;
}

async function postData(path: string, data: Record<string, unknown>) {
  const response = await apiClient.post(path, data);
  return response.data;
}

async function putData(path: string, data: Record<string, unknown>) {
  const response = await apiClient.put(path, data);
  return response.data;
}

async function deleteData(path: string) {
  const response = await apiClient.delete(path);
  return response.data;
}

function activityPayload(data: Partial<ActivityBody>): Record<string, unknown> {
  const { type, description, ...rest } = data;
  const payload = { ...rest };
  const activityType = data.activityType ?? type;
  const body = data.body ?? description;
  if (activityType !== undefined) payload.activityType = activityType;
  if (body !== undefined) payload.body = body;
  return payload;
}

function attachmentPayload(data: AttachmentBody): WireAttachmentBody {
  const { fileName, ...rest } = data;
  const filename = data.filename ?? fileName;
  return filename === undefined ? rest : { ...rest, filename };
}

export const crmEndpointApi = {
  getAccount: (id: string) => getData(`/bff/api/v1/accounts/${id}`),
  updateAccount: (id: string, data: AccountBody) => putData(`/bff/api/v1/accounts/${id}`, data),
  deleteAccount: (id: string) => deleteData(`/bff/api/v1/accounts/${id}`),

  getContactsByAccount: (accountId: string) =>
    getData(`/bff/api/v1/accounts/${accountId}/contacts`),
  createContact: (data: ContactBody) => postData('/bff/api/v1/contacts', data),
  updateContact: (id: string, data: ContactBody) => putData(`/bff/api/v1/contacts/${id}`, data),
  deleteContact: (id: string) => deleteData(`/bff/api/v1/contacts/${id}`),

  createLead: (data: LeadBody) => postData('/bff/api/v1/leads', data),
  updateLead: (id: string, data: LeadBody) => putData(`/bff/api/v1/leads/${id}`, data),
  deleteLead: (id: string) => deleteData(`/bff/api/v1/leads/${id}`),

  deleteDeal: (id: string) => deleteData(`/bff/api/v1/deals/${id}`),
  deleteCase: (id: string) => deleteData(`/bff/api/v1/cases/${id}`),

  getPipelines: (workspaceId: string, pagination?: PageParams) =>
    getData('/bff/api/v1/pipelines', pageParams(workspaceId, pagination)),
  getPipeline: (id: string) => getData(`/bff/api/v1/pipelines/${id}`),
  createPipeline: (data: PipelineBody) => postData('/bff/api/v1/pipelines', data),
  updatePipeline: (id: string, data: PipelineBody) => putData(`/bff/api/v1/pipelines/${id}`, data),
  deletePipeline: (id: string) => deleteData(`/bff/api/v1/pipelines/${id}`),

  getPipelineStages: (pipelineId: string) => getData(`/bff/api/v1/pipelines/${pipelineId}/stages`),
  createPipelineStage: (pipelineId: string, data: StageBody) =>
    postData(`/bff/api/v1/pipelines/${pipelineId}/stages`, data),
  updatePipelineStage: (stageId: string, data: StageBody) =>
    putData(`/bff/api/v1/pipelines/stages/${stageId}`, data),
  deletePipelineStage: (stageId: string) => deleteData(`/bff/api/v1/pipelines/stages/${stageId}`),

  getActivities: (workspaceId: string, pagination?: OffsetParams) =>
    getData('/bff/api/v1/activities', offsetParams(workspaceId, pagination)),
  getActivity: (id: string) => getData(`/bff/api/v1/activities/${id}`),
  createActivity: (data: ActivityBody) => postData('/bff/api/v1/activities', activityPayload(data)),
  updateActivity: (id: string, data: Partial<ActivityBody>) => putData(`/bff/api/v1/activities/${id}`, activityPayload(data)),
  deleteActivity: (id: string) => deleteData(`/bff/api/v1/activities/${id}`),

  getNotes: (workspaceId: string, pagination?: OffsetParams) =>
    getData('/bff/api/v1/notes', offsetParams(workspaceId, pagination)),
  getNote: (id: string) => getData(`/bff/api/v1/notes/${id}`),
  createNote: (data: NoteBody) => postData('/bff/api/v1/notes', data),
  updateNote: (id: string, data: Partial<NoteBody>) => putData(`/bff/api/v1/notes/${id}`, data),
  deleteNote: (id: string) => deleteData(`/bff/api/v1/notes/${id}`),

  getAttachments: (workspaceId: string, pagination?: OffsetParams) =>
    getData('/bff/api/v1/attachments', offsetParams(workspaceId, pagination)),
  getAttachment: (id: string) => getData(`/bff/api/v1/attachments/${id}`),
  createAttachment: (data: AttachmentBody) => postData('/bff/api/v1/attachments', attachmentPayload(data)),
  deleteAttachment: (id: string) => deleteData(`/bff/api/v1/attachments/${id}`),

  getTimeline: (workspaceId: string, pagination?: OffsetParams) =>
    getData('/bff/api/v1/timeline', offsetParams(workspaceId, pagination)),
  getTimelineByEntity: (entityType: string, entityId: string) =>
    getData(`/bff/api/v1/timeline/${entityType}/${entityId}`),
};
