import type { CopilotAbstentionReason, CopilotAnswerType, CopilotEvidenceMeta, EvidenceSource } from './sse.types';
import { readBoolean, readNumber, readString } from '../utils/recordReaders';

export function normalizeEvidenceSource(source: unknown): EvidenceSource {
  const record = source && typeof source === 'object' ? (source as Record<string, unknown>) : {};
  return {
    id: readString(record, 'id', 'ID') ?? '',
    snippet: readString(record, 'snippet', 'Snippet') ?? '',
    score: readNumber(record, 'score', 'Score') ?? 0,
    timestamp: readString(record, 'timestamp', 'created_at', 'createdAt', 'CreatedAt') ?? '',
    title: readString(record, 'title', 'Title'),
    knowledge_item_id: readString(record, 'knowledge_item_id', 'knowledgeItemID', 'knowledgeItemId', 'KnowledgeItemID'),
    retrieval_method: readString(record, 'retrieval_method', 'method', 'Method'),
    pii_redacted: readBoolean(record, 'pii_redacted', 'piiRedacted', 'PiiRedacted'),
  };
}

function normalizeConfidence(value: unknown) {
  return value === 'high' || value === 'medium' || value === 'low' ? value : undefined;
}

function normalizeStringArray(value: unknown): string[] | undefined {
  if (!Array.isArray(value)) return undefined;
  const items = value.filter((item): item is string => typeof item === 'string' && item.trim() !== '');
  return items.length > 0 ? items : undefined;
}

export function normalizeEvidenceMeta(meta: unknown): CopilotEvidenceMeta | undefined {
  if (!meta || typeof meta !== 'object') return undefined;
  const record = meta as Record<string, unknown>;

  const normalized: CopilotEvidenceMeta = {
    schemaVersion: readString(record, 'schema_version', 'schemaVersion'),
    sourceCount: readNumber(record, 'source_count', 'sourceCount'),
    dedupCount: readNumber(record, 'dedup_count', 'dedupCount'),
    filteredCount: readNumber(record, 'filtered_count', 'filteredCount'),
    confidence: normalizeConfidence(record.confidence),
    warnings: normalizeStringArray(record.warnings),
    retrievalMethodsUsed: normalizeStringArray(record.retrieval_methods_used ?? record.retrievalMethodsUsed),
    builtAt: readString(record, 'built_at', 'builtAt'),
  };

  return Object.values(normalized).some((value) => value !== undefined) ? normalized : undefined;
}

export function normalizeDoneMeta(meta: unknown): { answerType?: CopilotAnswerType; abstentionReason?: CopilotAbstentionReason } {
  if (!meta || typeof meta !== 'object') return {};
  const record = meta as Record<string, unknown>;
  const rawAnswerType = record.answer_type ?? record.answerType;
  const rawAbstentionReason = record.abstention_reason ?? record.abstentionReason;
  const answerType = rawAnswerType === 'grounded_answer' || rawAnswerType === 'abstention'
    ? rawAnswerType
    : undefined;
  const abstentionReason = rawAbstentionReason === 'insufficient_evidence' || rawAbstentionReason === 'irrelevant_evidence'
    ? rawAbstentionReason
    : undefined;

  return { answerType, abstentionReason };
}
