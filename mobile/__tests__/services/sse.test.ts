import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { createSSEClient, type SSEMessage } from '../../src/services/sse';

const originalFetch = global.fetch;

function responseWithFrames(frames: string[]): Response {
  const encoder = new TextEncoder();
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      frames.forEach((f) => controller.enqueue(encoder.encode(f)));
      controller.close();
    },
  });

  return {
    ok: true,
    status: 200,
    body: stream,
  } as Response;
}

describe('createSSEClient', () => {
  beforeEach(() => {
    global.fetch = jest.fn<typeof fetch>();
  });

  afterEach(() => {
    global.fetch = originalFetch;
    jest.clearAllMocks();
  });

  it('emits token, evidence and done from streamed frames', async () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    (global.fetch as jest.MockedFunction<typeof fetch>).mockResolvedValueOnce(
      responseWithFrames([
        'data: {"type":"token","delta":"hola "}\n\n',
        'data: {"type":"evidence","sources":[{"id":"e1","snippet":"s","score":0.9,"timestamp":"2026-01-01T00:00:00Z"}]}\n\n',
      ]),
    );

    createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);

    await new Promise((r) => setTimeout(r, 0));

    expect(onMessage).toHaveBeenCalledWith({ type: 'token', delta: 'hola ' });
    expect(onMessage).toHaveBeenCalledWith({
      type: 'evidence',
      sources: [{ id: 'e1', snippet: 's', score: 0.9, timestamp: '2026-01-01T00:00:00Z' }],
    });
    expect(onMessage).toHaveBeenCalledWith({ type: 'done' });
  });

  it('emits error when http status is not ok', async () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    (global.fetch as jest.MockedFunction<typeof fetch>).mockResolvedValueOnce({
      ok: false,
      status: 500,
      body: null,
    } as Response);

    createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);
    await new Promise((r) => setTimeout(r, 0));

    expect(onMessage).toHaveBeenCalledWith({ type: 'error', message: 'SSE request failed (500)' });
  });

  it('ignores abort errors after close()', async () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    (global.fetch as jest.MockedFunction<typeof fetch>).mockRejectedValueOnce(
      new DOMException('Aborted', 'AbortError'),
    );

    const client = createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);
    client.close();
    await new Promise((r) => setTimeout(r, 0));

    expect(onMessage).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'error' }));
  });

  it('returns a close function', () => {
    const client = createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, () => undefined);
    expect(typeof client.close).toBe('function');
  });
});
