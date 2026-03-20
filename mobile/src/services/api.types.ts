// Shared types for the API layer — extracted from api.ts to keep it under 300 lines

export type SignalStatus = 'active' | 'dismissed' | 'expired';

export interface Signal {
  id: string;
  workspace_id: string;
  entity_type: string;
  entity_id: string;
  signal_type: string;
  confidence: number;
  evidence_ids: string[];
  source_type: string;
  source_id: string;
  metadata: Record<string, unknown>;
  status: SignalStatus;
  dismissed_by?: string;
  dismissed_at?: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export type WorkflowStatus = 'draft' | 'testing' | 'active' | 'archived';

export interface Workflow {
  id: string;
  workspace_id: string;
  agent_definition_id?: string;
  parent_version_id?: string;
  name: string;
  description?: string;
  dsl_source: string;
  spec_source?: string;
  version: number;
  status: WorkflowStatus;
  created_by_user_id?: string;
  archived_at?: string;
  created_at: string;
  updated_at: string;
}

export type WorkflowVersion = Workflow;

export interface CreateWorkflowInput {
  agent_definition_id?: string;
  name: string;
  description?: string;
  dsl_source: string;
  spec_source?: string;
}

export interface UpdateWorkflowInput {
  agent_definition_id?: string;
  description?: string;
  dsl_source: string;
  spec_source?: string;
}

export type AgentRunStatus =
  | 'running'
  | 'success'
  | 'failed'
  | 'abstained'
  | 'partial'
  | 'escalated'
  | 'accepted'
  | 'rejected'
  | 'delegated';

export interface AgentRun {
  id: string;
  workspaceId: string;
  agentDefinitionId: string;
  triggeredByUserId?: string;
  triggerType: string;
  status: AgentRunStatus;
  inputs?: Record<string, unknown> | null;
  output?: Record<string, unknown> | null;
  toolCalls?: unknown[] | Record<string, unknown> | null;
  reasoningTrace?: unknown[] | Record<string, unknown> | null;
  totalTokens?: number;
  totalCost?: number;
  latencyMs?: number;
  traceId?: string;
  workflow_id?: string;
  entity_type?: string;
  entity_id?: string;
  rejection_reason?: string;
  startedAt: string;
  completedAt?: string;
  createdAt: string;
}

export interface AgentRunListResponse {
  data: AgentRun[];
  meta?: {
    total?: number;
    limit?: number;
    offset?: number;
  };
}

export interface AgentRunResponse {
  data: AgentRun;
}

export type ApprovalStatus = 'pending' | 'approved' | 'denied' | 'expired';

export interface ApprovalRequest {
  id: string;
  workspace_id: string;
  requested_by: string;
  approver_id: string;
  decided_by?: string;
  action: string;
  resource_type?: string;
  resource_id?: string;
  payload: Record<string, unknown>;
  reason?: string;
  status: ApprovalStatus;
  expires_at: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
}

// Task Mobile P1.8 — FR-232/UC-A7: handoff package type
export interface HandoffPackage {
  run_id: string;
  reason: string;
  conversation_context: string;
  evidence_count: number;
  entity_type?: string;
  entity_id?: string;
  created_at: string;
}
