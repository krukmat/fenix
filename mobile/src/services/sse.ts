// Task 4.4 — FR-200/FR-092: SSE client via fetch + ReadableStream (POST)

export interface EvidenceSource {
  id: string;
  snippet: string;
  score: number;
  timestamp: string;
  title?: string;
}

export type SSEMessage =
  | { type: 'token'; delta: string }
  | { type: 'evidence'; sources: EvidenceSource[] }
  | { type: 'done' }
  | { type: 'error'; message: string };

export interface SSEClient {
  close: () => void;
}

type RawStreamEvent = {
  type?: string;
  delta?: string;
  sources?: EvidenceSource[];
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
    return { type: 'evidence', sources: msg.sources };
  }
  if (isDoneEvent(msg)) {
    return { type: 'done' };
  }
  if (isErrorEvent(msg)) {
    return { type: 'error', message: msg.error ?? msg.message ?? 'SSE error' };
  }

  return null;
}

function isTokenEvent(msg: RawStreamEvent): msg is RawStreamEvent & { type: 'token'; delta: string } {
  return msg.type === 'token' && typeof msg.delta === 'string';
}

function isEvidenceEvent(msg: RawStreamEvent): msg is RawStreamEvent & { type: 'evidence'; sources: EvidenceSource[] } {
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

function handleTransportError(err: unknown, onMessage: (msg: SSEMessage) => void, onError?: (err: Error) => void): void {
  const aborted = err instanceof DOMException && err.name === 'AbortError';
  if (aborted) return;

  const message = err instanceof Error ? err.message : 'Unknown SSE error';
  onMessage({ type: 'error', message });
  onError?.(err instanceof Error ? err : new Error(message));
}

async function streamLoop(
  response: Response,
  onMessage: (msg: SSEMessage) => void,
): Promise<void> {
  if (!response.ok || !response.body) {
    throw new Error(`SSE request failed (${response.status})`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const frames = buffer.split('\n\n');
    buffer = frames.pop() ?? '';
    mapFrames(frames, onMessage);
  }

  if (buffer.trim()) {
    mapFrames([buffer], onMessage);
  }
}

export function createSSEClient(
  url: string,
  token: string,
  body: Record<string, unknown> = {},
  onMessage: (msg: SSEMessage) => void = () => undefined,
  onError?: (err: Error) => void,
): SSEClient {
  const controller = new AbortController();

  void (async () => {
    try {
      const response = await fetch(url, {
        method: 'POST',
        signal: controller.signal,
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
        },
        body: JSON.stringify(body),
      });
      await streamLoop(response, onMessage);
      onMessage({ type: 'done' });
    } catch (err) {
      handleTransportError(err, onMessage, onError);
    }
  })();

  return {
    close: () => controller.abort(),
  };
}
