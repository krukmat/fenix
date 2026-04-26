// bff-http-snapshots T6: SSE stream capture via native fetch + ReadableStream
export type SSEEvent = {
  data: string;
  index: number;
};

type SSEOptions = {
  maxEvents: number;
  timeoutMs: number;
};

/** Sentinel returned by the timeout race branch. */
const TIMEOUT_SENTINEL = Symbol('timeout');

/**
 * Races a single reader.read() call against an overall deadline.
 *
 * Returns the chunk result or TIMEOUT_SENTINEL.
 * Does NOT use AbortController — Jest intercepts AbortError from setTimeout
 * callbacks even inside try/catch. Instead we use Promise.race with a sentinel
 * value; the pending reader.read() is cleaned up by reader.cancel() in the caller.
 */
async function readWithDeadline(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  deadlineMs: number,
): Promise<{ done: true; value?: undefined } | { done: false; value: Uint8Array } | typeof TIMEOUT_SENTINEL> {
  const deadline = new Promise<typeof TIMEOUT_SENTINEL>((resolve) =>
    setTimeout(() => resolve(TIMEOUT_SENTINEL), deadlineMs),
  );
  // Cast through unknown to avoid the ReadableStreamReadDoneResult optional-value mismatch
  // between DOM typings and the narrower shape we declare here.
  return Promise.race([
    reader.read() as Promise<{ done: true; value?: undefined } | { done: false; value: Uint8Array }>,
    deadline,
  ]);
}

/**
 * Opens an SSE stream and captures up to `maxEvents` events or until `timeoutMs`.
 * Returns an empty array on non-200 response or network failure.
 */
export async function captureSSE(
  url: string,
  headers: Record<string, string>,
  options: SSEOptions,
): Promise<SSEEvent[]> {
  const { maxEvents, timeoutMs } = options;

  const deadline = Date.now() + timeoutMs;
  const events: SSEEvent[] = [];

  let res: Response;
  try {
    res = await fetch(url, {
      method: 'GET',
      headers: { accept: 'text/event-stream', ...headers },
    });
  } catch {
    return events;
  }

  if (!res.ok || !res.body) return events;

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (events.length < maxEvents) {
      const remainingMs = deadline - Date.now();
      if (remainingMs <= 0) break;

      const result = await readWithDeadline(reader, remainingMs);
      if (result === TIMEOUT_SENTINEL) break;

      const { done, value } = result;
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // SSE events are delimited by blank lines (\n\n)
      const parts = buffer.split('\n\n');
      // Last element may be an incomplete event — keep it in buffer
      buffer = parts.pop() ?? '';

      for (const part of parts) {
        if (events.length >= maxEvents) break;
        const dataLine = extractDataLine(part);
        if (dataLine !== null) {
          events.push({ data: dataLine, index: events.length });
        }
      }
    }
  } finally {
    try { reader.cancel(); } catch { /* ignore */ }
  }

  return events;
}

/** Extracts the value from the first `data:` line in an SSE event block. */
function extractDataLine(eventBlock: string): string | null {
  for (const line of eventBlock.split('\n')) {
    const trimmed = line.trimStart();
    if (trimmed.startsWith('data:')) {
      // data: value  (single leading space is conventional but optional)
      return trimmed.slice(5).replace(/^ /, '');
    }
  }
  return null;
}
