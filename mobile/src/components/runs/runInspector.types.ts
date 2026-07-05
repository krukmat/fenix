import type { HandoffPackage, UsageEvent } from '../../services/api.types';
import type { EvidenceSource as CopilotEvidenceSource } from '../../services/sse';

export interface RunInspectorDetail {
  id: string;
  agent_name: string;
  status: string;
  runtime_status?: string;
  entity_type?: string;
  entity_id?: string;
  triggered_by?: string;
  trigger_type?: string;
  inputs?: Record<string, unknown> | null;
  evidence_retrieved?: unknown[] | null;
  reasoning_trace?: unknown[] | null;
  tool_calls?: unknown[] | null;
  output?: unknown;
  audit_events?: unknown[] | null;
  latency_ms?: number;
  cost_euros?: number;
  rejection_reason?: string;
  trace_id?: string;
  traceId?: string;
  created_at?: string;
  started_at?: string;
  completed_at?: string;
}

export type RunEvidenceMeta = {
  schemaVersion?: string;
  confidence?: 'high' | 'medium' | 'low';
  warnings?: string[];
  retrievalMethodsUsed?: string[];
  builtAt?: string;
};

export type RunEvidenceItem = CopilotEvidenceSource;

export type NormalizedToolCall = {
  toolName: string;
  status?: string;
  latencyMs?: number;
  idempotencyKey?: string;
  input?: unknown;
  output?: unknown;
};

export type ApprovalLinkage = {
  approvalId?: string;
  action?: string;
};

export type RunInspectorProps = {
  run: RunInspectorDetail;
  usage: UsageEvent[] | undefined;
  handoff?: HandoffPackage;
  handoffLoading: boolean;
  onOpenAuditTrail: () => void;
  onOpenInbox: () => void;
  onOpenHandoffDestination: () => void;
  onViewFullUsage: () => void;
};
