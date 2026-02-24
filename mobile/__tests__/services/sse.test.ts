// Task 4.4 — SSE client tests using XHR mock (React Native transport).
// XHR is used instead of fetch+ReadableStream because React Native Hermes does not expose
// response.body as a ReadableStream.
import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { createSSEClient, type SSEMessage } from '../../src/services/sse';

// Minimal XHR mock that simulates the onprogress + onload lifecycle
class FakeXHR {
  // Public fields set by open/setRequestHeader/send
  method = '';
  url = '';
  headers: Record<string, string> = {};
  sentBody: string | null = null;
  responseText = '';
  status = 200;

  // Callbacks assigned by the code under test
  onprogress: (() => void) | null = null;
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onabort: (() => void) | null = null;

  open(method: string, url: string) {
    this.method = method;
    this.url = url;
  }
  setRequestHeader(name: string, value: string) {
    this.headers[name] = value;
  }
  send(body: string) {
    this.sentBody = body;
  }
  abort() {
    this.onabort?.();
  }

  // Test helpers — simulate server pushing chunks
  pushChunk(text: string) {
    this.responseText += text;
    this.onprogress?.();
  }
  complete(status = 200) {
    this.status = status;
    this.onload?.();
  }
  fail() {
    this.onerror?.();
  }
}

let fakeXhr: FakeXHR;

beforeEach(() => {
  fakeXhr = new FakeXHR();
  // Replace global XMLHttpRequest with our fake
  (global as unknown as Record<string, unknown>).XMLHttpRequest = jest.fn(() => fakeXhr);
});

afterEach(() => {
  jest.clearAllMocks();
});

describe('createSSEClient (XHR transport)', () => {
  it('emits token, evidence and done from streamed frames', () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);

    fakeXhr.pushChunk('data: {"type":"token","delta":"hola "}\n\n');
    fakeXhr.pushChunk('data: {"type":"evidence","sources":[{"id":"e1","snippet":"s","score":0.9,"timestamp":"2026-01-01T00:00:00Z"}]}\n\n');
    fakeXhr.complete(200);

    expect(onMessage).toHaveBeenCalledWith({ type: 'token', delta: 'hola ' });
    expect(onMessage).toHaveBeenCalledWith({
      type: 'evidence',
      sources: [{ id: 'e1', snippet: 's', score: 0.9, timestamp: '2026-01-01T00:00:00Z' }],
    });
    expect(onMessage).toHaveBeenCalledWith({ type: 'done' });
  });

  it('handles a chunk split across two pushes (partial frame)', () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);

    // Split the SSE frame across two network chunks
    fakeXhr.pushChunk('data: {"type":"token","delta"');
    fakeXhr.pushChunk(':"split "}\n\n');
    fakeXhr.complete(200);

    expect(onMessage).toHaveBeenCalledWith({ type: 'token', delta: 'split ' });
    expect(onMessage).toHaveBeenCalledWith({ type: 'done' });
  });

  it('emits error when http status is not ok', () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);
    fakeXhr.complete(500);

    expect(onMessage).toHaveBeenCalledWith({ type: 'error', message: 'SSE request failed (500)' });
  });

  it('emits error on network failure', () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);
    fakeXhr.fail();

    expect(onMessage).toHaveBeenCalledWith({ type: 'error', message: 'SSE network error' });
  });

  it('does not emit error when response.body would be null (React Native fetch behaviour — XHR handles this natively)', () => {
    // This test verifies that switching to XHR resolves the "SSE failed (200)" error that occurred
    // when fetch returned response.body=null on React Native. With XHR, body is never an issue.
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);
    // No pushChunk = body is empty, complete = success
    fakeXhr.complete(200);

    expect(onMessage).not.toHaveBeenCalledWith(
      expect.objectContaining({ type: 'error', message: 'SSE request failed (200)' }),
    );
    expect(onMessage).toHaveBeenCalledWith({ type: 'done' });
  });

  it('ignores abort errors after close()', () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    const client = createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);
    client.close();
    // onabort is called by xhr.abort() — should NOT emit error
    expect(onMessage).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'error' }));
  });

  it('does not emit done after close()', () => {
    const onMessage = jest.fn<(msg: SSEMessage) => void>();

    const client = createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, onMessage);
    client.close();
    // onload fires after abort in some runtimes — aborted flag prevents done emission
    fakeXhr.complete(200);

    expect(onMessage).not.toHaveBeenCalledWith({ type: 'done' });
  });

  it('sets correct request headers', () => {
    createSSEClient('http://localhost:3000/bff/copilot/chat', 'my-token', { query: 'x' }, () => undefined);

    expect(fakeXhr.headers['Authorization']).toBe('Bearer my-token');
    expect(fakeXhr.headers['Content-Type']).toBe('application/json');
    expect(fakeXhr.headers['Accept']).toBe('text/event-stream');
  });

  it('returns a close function', () => {
    const client = createSSEClient('http://localhost:3000/bff/copilot/chat', 'token', { query: 'x' }, () => undefined);
    expect(typeof client.close).toBe('function');
  });
});
