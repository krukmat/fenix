import type {
  ApprovalLinkage,
  NormalizedToolCall,
  RunEvidenceItem,
  RunEvidenceMeta,
  RunInspectorDetail,
} from './runInspector.types';
import { asRecord, readNumber, readString, readStringArray } from '../../utils/recordReaders';

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
