// Task 4.4 — FR-200/FR-092: SSE client via XMLHttpRequest (POST)
// Uses XHR instead of fetch+ReadableStream because React Native / Hermes does not expose
// response.body as a ReadableStream — XHR onprogress delivers incremental chunks natively.

export interface EvidenceSource {
  id: string;
  snippet: string;
  score: number;
  timestamp: string;
  title?: string;
  knowledge_item_id?: string;
  retrieval_method?: string;
  pii_redacted?: boolean;
}

export type CopilotConfidence = 'high' | 'medium' | 'low';
export type CopilotAnswerType = 'grounded_answer' | 'abstention';
export type CopilotAbstentionReason = 'insufficient_evidence' | 'irrelevant_evidence';

export interface CopilotEvidenceMeta {
  schemaVersion?: string;
  sourceCount?: number;
  dedupCount?: number;
  filteredCount?: number;
  confidence?: CopilotConfidence;
  warnings?: string[];
  retrievalMethodsUsed?: string[];
  builtAt?: string;
}

export type SSEMessage =
  | { type: 'token'; delta: string }
  | { type: 'evidence'; sources: EvidenceSource[]; meta?: CopilotEvidenceMeta }
  | { type: 'done'; answerType?: CopilotAnswerType; abstentionReason?: CopilotAbstentionReason }
  | { type: 'error'; message: string };

export interface SSEClient {
  close: () => void;
}

type RawStreamEvent = {
  type?: string;
  delta?: string;
  sources?: unknown[];
  meta?: unknown;
  error?: string;
  message?: string;
};

function parseFrame(frame: string): unknown | null {
  const lines = frame.split('\n');
  const dataLines = lines
    .map((line) => line.trim())
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trim());

  if (dataLines.length === 0) return null;
  const payload = dataLines.join('\n');
  if (!payload) return null;

  try {
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

function toClientMessage(evt: unknown): SSEMessage | null {
  if (!evt || typeof evt !== 'object') return null;
  const msg = evt as RawStreamEvent;

  if (isTokenEvent(msg)) {
    return { type: 'token', delta: msg.delta };
  }
  if (isEvidenceEvent(msg)) {
    return {
      type: 'evidence',
      sources: msg.sources.map(normalizeEvidenceSource),
      meta: normalizeEvidenceMeta(msg.meta),
    };
  }
  if (isDoneEvent(msg)) {
    const meta = normalizeDoneMeta(msg.meta);
    return { type: 'done', answerType: meta.answerType, abstentionReason: meta.abstentionReason };
  }
  if (isErrorEvent(msg)) {
    return { type: 'error', message: msg.error ?? msg.message ?? 'SSE error' };
  }

  return null;
}

function isTokenEvent(msg: RawStreamEvent): msg is RawStreamEvent & { type: 'token'; delta: string } {
  return msg.type === 'token' && typeof msg.delta === 'string';
}

function isEvidenceEvent(msg: RawStreamEvent): msg is RawStreamEvent & { type: 'evidence'; sources: unknown[] } {
  return msg.type === 'evidence' && Array.isArray(msg.sources);
}

function isDoneEvent(msg: RawStreamEvent): msg is RawStreamEvent & { type: 'done' } {
  return msg.type === 'done';
}

function isErrorEvent(msg: RawStreamEvent): msg is RawStreamEvent & { type: 'error' } {
  return msg.type === 'error';
}

function mapFrames(frames: string[], onMessage: (msg: SSEMessage) => void): void {
  for (const frame of frames) {
    const parsed = parseFrame(frame);
    const mapped = toClientMessage(parsed);
    if (mapped) onMessage(mapped);
  }
}

function readString(record: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value;
    }
  }
  return undefined;
}

function readNumber(record: Record<string, unknown>, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

function readBoolean(record: Record<string, unknown>, ...keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'boolean') {
      return value;
    }
  }
  return undefined;
}

function normalizeEvidenceSource(source: unknown): EvidenceSource {
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

function normalizeConfidence(value: unknown): CopilotConfidence | undefined {
  return value === 'high' || value === 'medium' || value === 'low' ? value : undefined;
}

function normalizeStringArray(value: unknown): string[] | undefined {
  if (!Array.isArray(value)) return undefined;
  const items = value.filter((item): item is string => typeof item === 'string' && item.trim() !== '');
  return items.length > 0 ? items : undefined;
}

function normalizeEvidenceMeta(meta: unknown): CopilotEvidenceMeta | undefined {
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

function normalizeDoneMeta(meta: unknown): { answerType?: CopilotAnswerType; abstentionReason?: CopilotAbstentionReason } {
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

// Task 4.4 UAT fix: XHR-based SSE transport.
// React Native Hermes/JSC does not expose response.body as ReadableStream, so fetch-based
// streaming fails with "SSE failed (200)". XMLHttpRequest.onprogress delivers incremental
// responseText chunks natively on both Android and iOS.
export function createSSEClient(
  url: string,
  token: string,
  body: Record<string, unknown> = {},
  onMessage: (msg: SSEMessage) => void = () => undefined,
  onError?: (err: Error) => void,
): SSEClient {
  const xhr = new XMLHttpRequest();
  let offset = 0; // track how many chars of responseText we've already processed
  let buffer = '';
  let aborted = false;

  xhr.open('POST', url, true);
  xhr.setRequestHeader('Authorization', `Bearer ${token}`);
  xhr.setRequestHeader('Content-Type', 'application/json');
  xhr.setRequestHeader('Accept', 'text/event-stream');

  xhr.onprogress = () => {
    // responseText grows with each chunk; slice only the new bytes
    const newText = xhr.responseText.slice(offset);
    offset = xhr.responseText.length;

    buffer += newText;
    const frames = buffer.split('\n\n');
    buffer = frames.pop() ?? '';
    mapFrames(frames, onMessage);
  };

  xhr.onload = () => {
    if (aborted) return;

    // Flush any remaining buffer after stream ends
    if (buffer.trim()) {
      mapFrames([buffer], onMessage);
      buffer = '';
    }

    if (xhr.status < 200 || xhr.status >= 300) {
      const err = new Error(`SSE request failed (${xhr.status})`);
      onMessage({ type: 'error', message: err.message });
      onError?.(err);
      return;
    }

    onMessage({ type: 'done' });
  };

  xhr.onerror = () => {
    if (aborted) return;
    const err = new Error('SSE network error');
    onMessage({ type: 'error', message: err.message });
    onError?.(err);
  };

  xhr.onabort = () => {
    // Abort is intentional (close() called) — do not emit error
  };

  xhr.send(JSON.stringify(body));

  return {
    close: () => {
      aborted = true;
      xhr.abort();
    },
  };
}
