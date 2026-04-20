export type CRMEntityType =
  | 'account'
  | 'contact'
  | 'lead'
  | 'pipeline'
  | 'pipeline_stage'
  | 'deal'
  | 'case'
  | 'activity'
  | 'note'
  | 'attachment';

export interface CRMPageMeta {
  total: number;
  limit: number;
  offset: number;
}

export interface CRMListResponse<T> {
  data: T[];
  meta: CRMPageMeta;
}

export interface CRMBaseEntity {
  id: string;
  workspaceId?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CRMOwnedEntity extends CRMBaseEntity {
  ownerId?: string;
}

export interface CRMAccount extends CRMOwnedEntity {
  name: string;
  industry?: string;
  website?: string;
  phone?: string;
  email?: string;
  description?: string;
  activeSignalCount: number;
}

export interface CRMContact extends CRMOwnedEntity {
  accountId?: string;
  firstName?: string;
  lastName?: string;
  email?: string;
  phone?: string;
  title?: string;
  activeSignalCount: number;
}

export interface CRMLead extends CRMOwnedEntity {
  accountId?: string;
  contactId?: string;
  source?: string;
  status?: string;
  score?: number;
  metadata: Record<string, unknown>;
}

export interface CRMPipeline extends CRMBaseEntity {
  name: string;
  entityType?: string;
  isDefault: boolean;
}

export interface CRMPipelineStage extends CRMBaseEntity {
  pipelineId: string;
  name: string;
  position?: number;
  probability?: number;
}

export interface CRMDeal extends CRMOwnedEntity {
  accountId: string;
  contactId?: string;
  pipelineId: string;
  stageId: string;
  title: string;
  amount?: number;
  currency?: string;
  expectedClose?: string;
  status?: string;
  metadata: Record<string, unknown>;
  accountName?: string;
  activeSignalCount: number;
}

export interface CRMCase extends CRMOwnedEntity {
  accountId?: string;
  contactId?: string;
  pipelineId?: string;
  stageId?: string;
  subject: string;
  description?: string;
  priority?: string;
  status?: string;
  channel?: string;
  slaConfig?: string;
  slaDeadline?: string;
  metadata: Record<string, unknown>;
  accountName?: string;
  activeSignalCount: number;
}

export interface CRMActivity extends CRMBaseEntity {
  entityType: CRMEntityType | string;
  entityId: string;
  ownerId?: string;
  assignedTo?: string;
  type?: string;
  subject?: string;
  description?: string;
  status?: string;
  dueAt?: string;
  completedAt?: string;
  metadata: Record<string, unknown>;
}

export interface CRMNote extends CRMBaseEntity {
  entityType: CRMEntityType | string;
  entityId: string;
  authorId?: string;
  content: string;
  isInternal: boolean;
  metadata: Record<string, unknown>;
}

export interface CRMAttachment extends CRMBaseEntity {
  entityType: CRMEntityType | string;
  entityId: string;
  uploaderId?: string;
  fileName?: string;
  storagePath?: string;
  contentType?: string;
  sizeBytes?: number;
  metadata: Record<string, unknown>;
}

export interface CRMTimelineEvent extends CRMBaseEntity {
  entityType: CRMEntityType | string;
  entityId: string;
  actorId?: string;
  eventType?: string;
  action?: string;
  title: string;
  description?: string;
  timestamp: string;
  metadata: Record<string, unknown>;
}
