// Shared types for the API layer — extracted from api.ts to keep it under 300 lines
// W1-T1 (mobile_wedge_harmonization_plan): frozen public contracts for wedge surfaces

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

// W1-T1: normalized public outcome set — used in all user-facing lists and badges
export type AgentRunPublicStatus =
  | 'completed'
  | 'completed_with_warnings'
  | 'abstained'
  | 'awaiting_approval'
  | 'handed_off'
  | 'denied_by_policy'
  | 'failed';

// W1-T1: raw runtime diagnostic set — used only in run detail diagnostics section
export type AgentRunRuntimeStatus =
  | 'running'
  | 'success'
  | 'partial'
  | 'abstained'
  | 'failed'
  | 'escalated'
  | 'accepted'
  | 'rejected'
  | 'delegated';

// Legacy alias — kept so existing code that imports AgentRunStatus keeps compiling.
// Migrate call sites to AgentRunPublicStatus or AgentRunRuntimeStatus as each surface is rewritten.
export type AgentRunStatus = AgentRunRuntimeStatus;

export interface AgentRun {
  id: string;
  workspaceId: string;
  agentDefinitionId: string;
  triggeredByUserId?: string;
  triggerType: string;
  // W1-T1: normalized public outcome — render this in all user-facing lists and badges
  status: AgentRunPublicStatus;
  // W1-T1: raw runtime diagnostic — render only in run detail diagnostics section
  runtime_status?: AgentRunRuntimeStatus;
  inputs?: Record<string, unknown> | null;
  output?: Record<string, unknown> | null;
  toolCalls?: unknown[] | Record<string, unknown> | null;
  reasoningTrace?: unknown[] | Record<string, unknown> | null;
  totalTokens?: number;
  totalCost?: number;
  latencyMs?: number;
  traceId?: string;
  entity_type?: string;
  entity_id?: string;
  // W1-T1: only present when status == 'denied_by_policy'
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

// W1-T1: normalized approval state model — 'denied' is legacy, do not use in new code
export type ApprovalStatus = 'pending' | 'approved' | 'rejected' | 'expired' | 'cancelled';

// W1-T1: decisions the mobile client may send
export type ApprovalDecision = 'approve' | 'reject';

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
  expiresAt: string;
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
  caseId?: string;
  triggerContext?: {
    entity_type?: string;
    entity_id?: string;
  };
  finalOutput?: {
    entity_type?: string;
    entity_id?: string;
  };
}

// W1-T1: Evidence source item within an evidence pack
export interface EvidenceSource {
  id: string;
  snippet: string;
  score: number;
  timestamp: string;
  source_type?: string;
}

// W1-T1: Evidence Pack v1 — frozen contract for all retrieval-backed responses
export interface EvidencePack {
  schema_version: string;
  query: string;
  sources: EvidenceSource[];
  source_count: number;
  dedup_count: number;
  filtered_count: number;
  confidence: 'high' | 'medium' | 'low';
  warnings: string[];
  retrieval_methods_used: string[];
  built_at: string;
}

// W1-T1: Sales Brief contract — frozen for POST /api/v1/copilot/sales-brief
export type SalesBriefOutcome = 'completed' | 'abstained';

export interface SalesBriefAction {
  title: string;
  description: string;
  tool: string;
  params: Record<string, unknown>;
  confidence_score?: number;
  confidence_level?: 'high' | 'medium' | 'low';
}

export interface SalesBrief {
  outcome: SalesBriefOutcome;
  entityType: string;
  entityId: string;
  // present when outcome == 'completed'
  summary?: string;
  risks?: string[];
  nextBestActions?: SalesBriefAction[];
  // present when outcome == 'abstained'
  abstentionReason?: string;
  confidence: 'high' | 'medium' | 'low';
  evidencePack: EvidencePack;
}

// W1-T1: Usage event — single metered interaction record
export interface UsageEvent {
  id: string;
  workspace_id: string;
  actor_id?: string;
  actor_type?: string;
  run_id?: string;
  tool_name?: string;
  model_name?: string;
  input_units?: number;
  output_units?: number;
  estimated_cost?: number;
  latency_ms?: number;
  created_at: string;
}

// W1-T1: Quota state item — enriched with policy metadata for mobile rendering
export interface QuotaStateItem {
  policyId: string;
  policyType: string;
  metricName?: string;
  limitValue: number;
  resetPeriod: string;
  enforcementMode: string;
  currentValue: number;
  periodStart: string;
  periodEnd: string;
  lastEventAt?: string;
  // false when no state row exists yet for the current period (currentValue is 0)
  statePresent: boolean;
}

// W1-T1: Governance summary — single endpoint response for the governance screen
export interface GovernanceSummary {
  recentUsage: UsageEvent[];
  quotaStates: QuotaStateItem[];
}

// W1-T1: Inbox item types
export interface InboxApprovalItem {
  type: 'approval';
  approval: ApprovalRequest;
}

export interface InboxHandoffItem {
  type: 'handoff';
  run_id: string;
  handoff: HandoffPackage;
}

export interface InboxSignalItem {
  type: 'signal';
  signal: Signal;
}

export interface InboxRejectedItem {
  type: 'rejected';
  run: AgentRun;
}

export type InboxItem = InboxApprovalItem | InboxHandoffItem | InboxSignalItem | InboxRejectedItem;

export interface InboxResponse {
  approvals: ApprovalRequest[];
  handoffs: InboxHandoffItem[];
  signals: Signal[];
  rejected: AgentRun[];
}
