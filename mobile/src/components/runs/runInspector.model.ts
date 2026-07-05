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

function asRecord(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}

function readString(record: Record<string, unknown> | null, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value.trim();
    }
  }
  return undefined;
}

function readNumber(record: Record<string, unknown> | null, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

function readStringArray(record: Record<string, unknown> | null, ...keys: string[]): string[] | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (!Array.isArray(value)) continue;
    const items = value.filter((item): item is string => typeof item === 'string' && item.trim() !== '');
    if (items.length > 0) {
      return items.map((item) => item.trim());
    }
  }
  return undefined;
}

export function formatLabel(value: string): string {
  return value
    .split(/[_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function formatMoney(value?: number): string {
  return typeof value === 'number' ? `€${value.toFixed(4)}` : '—';
}

export function formatMs(value?: number): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '—';
  if (value < 1000) return `${value} ms`;
  return `${(value / 1000).toFixed(1)} s`;
}

export function formatTimestamp(value?: string): string {
  if (!value) return '—';
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toISOString();
}

export function formatJson(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'string') return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function normalizeEvidenceItems(run: RunInspectorDetail): RunEvidenceItem[] {
  return (run.evidence_retrieved ?? []).map((source, index) => {
    const record = asRecord(source);
    return {
      id: readString(record, 'id', 'source_id', 'sourceId') ?? `evidence-${index}`,
      snippet: readString(record, 'snippet') ?? '',
      score: readNumber(record, 'score', 'relevance_score', 'relevanceScore') ?? 0,
      timestamp: readString(record, 'timestamp', 'source_timestamp', 'sourceTimestamp', 'created_at', 'createdAt') ?? '',
      title: readString(record, 'title'),
      knowledge_item_id: readString(record, 'knowledge_item_id', 'knowledgeItemId', 'knowledgeItemID'),
      retrieval_method: readString(record, 'retrieval_method', 'method', 'retrievalMethod'),
      pii_redacted: record?.pii_redacted === true || record?.piiRedacted === true,
    };
  });
}

export function extractEvidenceMeta(run: RunInspectorDetail): RunEvidenceMeta | undefined {
  const runRecord = run as unknown as Record<string, unknown>;
  const output = asRecord(run.output);
  const directMeta = asRecord(runRecord.evidence_pack ?? runRecord.evidencePack);
  const nestedMeta = asRecord(output?.evidence_pack ?? output?.evidencePack);
  const record = directMeta ?? nestedMeta;

  if (!record) return undefined;

  const meta: RunEvidenceMeta = {
    schemaVersion: readString(record, 'schema_version', 'schemaVersion'),
    confidence: readString(record, 'confidence') as RunEvidenceMeta['confidence'],
    warnings: readStringArray(record, 'warnings'),
    retrievalMethodsUsed: readStringArray(record, 'retrieval_methods_used', 'retrievalMethodsUsed'),
    builtAt: readString(record, 'built_at', 'builtAt'),
  };

  return Object.values(meta).some((value) => value !== undefined) ? meta : undefined;
}

export function normalizeToolCalls(run: RunInspectorDetail): NormalizedToolCall[] {
  const calls: (NormalizedToolCall | null)[] = (run.tool_calls ?? [])
    .map((call) => {
      const record = asRecord(call);
      if (!record) return null;

      const input = record.params ?? record.input;
      const output = record.result ?? record.output;
      const outputRecord = asRecord(output);

      return {
        toolName: readString(record, 'tool_name', 'toolName', 'name') ?? 'Tool call',
        status: readString(record, 'status') ?? readString(outputRecord, 'status', 'outcome'),
        latencyMs: readNumber(record, 'latency_ms', 'latencyMs'),
        idempotencyKey: readString(record, 'idempotency_key', 'idempotencyKey'),
        input,
        output,
      };
    });

  return calls.filter((call): call is NormalizedToolCall => call !== null);
}

export function extractApprovalLinkage(run: RunInspectorDetail): ApprovalLinkage {
  const output = asRecord(run.output);
  const inputs = asRecord(run.inputs);
  const toolCalls = Array.isArray(run.tool_calls) ? run.tool_calls : [];

  for (const call of toolCalls) {
    const record = asRecord(call);
    const result = asRecord(record?.result ?? record?.output);
    const approvalId = readString(result, 'approval_id', 'approvalId');
    if (approvalId) {
      return {
        approvalId,
        action: readString(result, 'action'),
      };
    }
  }

  return {
    approvalId:
      readString(output, 'approval_id', 'approvalId') ??
      readString(inputs, 'approval_id', 'approvalId'),
    action:
      readString(output, 'action') ??
      readString(inputs, 'action'),
  };
}

export function normalizeReasoningTrace(run: RunInspectorDetail): string[] {
  if (!Array.isArray(run.reasoning_trace)) return [];
  return run.reasoning_trace.filter((step): step is string => typeof step === 'string' && step.trim() !== '');
}

export function sumUsage(events: UsageEvent[] | undefined) {
  return (events ?? []).reduce(
    (acc, event) => ({
      inputUnits: acc.inputUnits + (event.inputUnits ?? 0),
      outputUnits: acc.outputUnits + (event.outputUnits ?? 0),
      estimatedCost: acc.estimatedCost + (event.estimatedCost ?? 0),
      latencyMs: acc.latencyMs + (event.latencyMs ?? 0),
    }),
    { inputUnits: 0, outputUnits: 0, estimatedCost: 0, latencyMs: 0 },
  );
}
