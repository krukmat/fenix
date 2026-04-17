// Task 4.2 — FR-300: Axios API Client hacia BFF
// W6-T1: removed workflowApi and workflow types (wedge cleanup)
// Task Mobile P1.4 — T2: re-added WorkflowStatus, Workflow, workflowApi
//
// Public API surface for the services layer.
// Implementation split across api.client.ts, api.types.ts, api.agents.ts,
// api.secondary.ts, api.crm.ts, api.workflows.ts, api.copilot.ts
// to keep each file under the 300-line architecture gate.

export { apiClient, BFF_URL } from './api.client';

// ─── Workflow types (Task Mobile P1.4 — T2) ───────────────────────────────────

export type WorkflowStatus = 'draft' | 'testing' | 'active' | 'archived';

export interface Workflow {
  id: string;
  workspace_id?: string;
  name: string;
  description?: string;
  status: WorkflowStatus;
  version: number;
  dsl_source: string;
  created_at: string;
  updated_at: string;
}

// ─── Re-export all types so existing imports keep working ────────────────────
export type {
  SignalStatus,
  Signal,
  AgentRunPublicStatus,
  AgentRunRuntimeStatus,
  AgentRunStatus,
  AgentRun,
  AgentRunListResponse,
  AgentRunResponse,
  ApprovalStatus,
  ApprovalDecision,
  ApprovalRequest,
  QueuedAgentTriggerResponse,
  HandoffPackage,
  EvidenceSource,
  EvidencePack,
  SalesBriefOutcome,
  SalesBriefAction,
  SalesBrief,
  UsageEvent,
  AuditEvent,
  AuditFilters,
  AuditOutcome,
  AuditActorType,
  PaginatedResponse,
  QuotaStateItem,
  GovernanceSummary,
  UsageFilters,
  UsageCostSummary,
  InboxItem,
  InboxApprovalItem,
  InboxHandoffItem,
  InboxSignalItem,
  InboxRejectedItem,
  InboxResponse,
} from './api.types';

export { agentApi, salesBriefApi } from './api.agents';
export { signalApi, toolApi, approvalApi, inboxApi, governanceApi } from './api.secondary';
export { authApi, crmApi } from './api.crm';
export { workflowApi } from './api.workflows';
export { copilotApi } from './api.copilot';
