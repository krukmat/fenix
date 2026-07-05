// Task 4.4 — FR-200/FR-092: SSE client via XMLHttpRequest (POST)
// Uses XHR instead of fetch+ReadableStream because React Native / Hermes does not expose
// response.body as a ReadableStream — XHR onprogress delivers incremental chunks natively.

import type { SSEClient, SSEMessage } from './sse.types';
import { normalizeDoneMeta, normalizeEvidenceMeta, normalizeEvidenceSource } from './sse.normalize';

export type {
  CopilotAbstentionReason,
  CopilotAnswerType,
  CopilotConfidence,
  CopilotEvidenceMeta,
  EvidenceSource,
  SSEClient,
  SSEMessage,
} from './sse.types';

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

function buildTokenMessage(msg: RawStreamEvent): SSEMessage | null {
  return typeof msg.delta === 'string' ? { type: 'token', delta: msg.delta } : null;
}

function buildEvidenceMessage(msg: RawStreamEvent): SSEMessage | null {
  if (!Array.isArray(msg.sources)) return null;
  return {
    type: 'evidence',
    sources: msg.sources.map(normalizeEvidenceSource),
    meta: normalizeEvidenceMeta(msg.meta),
  };
}

function buildDoneMessage(msg: RawStreamEvent): SSEMessage {
  const meta = normalizeDoneMeta(msg.meta);
  return { type: 'done', answerType: meta.answerType, abstentionReason: meta.abstentionReason };
}

function buildErrorMessage(msg: RawStreamEvent): SSEMessage {
  return { type: 'error', message: msg.error ?? msg.message ?? 'SSE error' };
}

const MESSAGE_BUILDERS: Record<string, (msg: RawStreamEvent) => SSEMessage | null> = {
  token: buildTokenMessage,
  evidence: buildEvidenceMessage,
  done: buildDoneMessage,
  error: buildErrorMessage,
};

function toClientMessage(evt: unknown): SSEMessage | null {
  if (!evt || typeof evt !== 'object') return null;
  const msg = evt as RawStreamEvent;
  const builder = msg.type ? MESSAGE_BUILDERS[msg.type] : undefined;
  return builder ? builder(msg) : null;
}

function mapFrames(frames: string[], onMessage: (msg: SSEMessage) => void): void {
  for (const frame of frames) {
    const parsed = parseFrame(frame);
    const mapped = toClientMessage(parsed);
    if (mapped) onMessage(mapped);
  }
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
