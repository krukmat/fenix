// bff-http-snapshots T6: SSE capture tests — written before implementation (TDD)
import http from 'http';
import { captureSSE, type SSEEvent } from '../../scripts/snapshots/sse-capture';

function listen(server: http.Server): Promise<number> {
  return new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      resolve((server.address() as { port: number }).port);
    });
  });
}

function close(server: http.Server): Promise<void> {
  return new Promise((resolve) => server.close(() => resolve()));
}

/** Builds an SSE server that sends `events` then optionally hangs forever. */
function makeSseServer(events: string[], hangAfter = false): http.Server {
  return http.createServer((_req, res) => {
    res.writeHead(200, {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    });

    let i = 0;
    function send() {
      if (i >= events.length) {
        if (!hangAfter) res.end();
        return;
      }
      res.write(`data: ${events[i]}\n\n`);
      i++;
      setImmediate(send);
    }
    send();
  });
}

// ── Core capture behaviour ───────────────────────────────────────────────────

describe('captureSSE — basic capture', () => {
  let server: http.Server;
  let baseUrl: string;

  beforeEach(async () => {
    server = makeSseServer(['{"type":"chunk","text":"hello"}', '{"type":"done"}', '{"type":"end"}']);
    const port = await listen(server);
    baseUrl = `http://127.0.0.1:${port}/sse`;
  });

  afterEach(() => close(server));

  it('captures all events when server closes before maxEvents', async () => {
    const events = await captureSSE(baseUrl, {}, { maxEvents: 10, timeoutMs: 2000 });

    expect(events).toHaveLength(3);
    expect(events[0]).toMatchObject({ data: '{"type":"chunk","text":"hello"}' });
    expect(events[1]).toMatchObject({ data: '{"type":"done"}' });
    expect(events[2]).toMatchObject({ data: '{"type":"end"}' });
  });

  it('stops at maxEvents when server keeps streaming', async () => {
    const infiniteServer = makeSseServer(
      Array.from({ length: 20 }, (_, i) => `{"n":${i}}`),
      true, // hang after sending all 20 — simulates real SSE that never closes
    );
    const port2 = await listen(infiniteServer);

    const events = await captureSSE(`http://127.0.0.1:${port2}/sse`, {}, { maxEvents: 5, timeoutMs: 3000 });

    await close(infiniteServer);

    expect(events).toHaveLength(5);
  });
});

describe('captureSSE — timeout', () => {
  it('stops after timeoutMs when server hangs without events', async () => {
    const hangingServer = http.createServer((_req, res) => {
      res.writeHead(200, { 'Content-Type': 'text/event-stream' });
      res.write(''); // flush headers so fetch() completes and reader.read() blocks
      // then hang — simulates stalled upstream
    });
    const port = await listen(hangingServer);

    const start = Date.now();
    const events = await captureSSE(
      `http://127.0.0.1:${port}/sse`,
      {},
      { maxEvents: 10, timeoutMs: 300 },
    );
    const elapsed = Date.now() - start;

    await close(hangingServer);

    expect(events).toHaveLength(0);
    // should resolve close to the timeout, not sooner
    expect(elapsed).toBeGreaterThanOrEqual(250);
  });

  it('returns partial events captured before timeout', async () => {
    // sends 1 event then hangs
    const partialServer = http.createServer((_req, res) => {
      res.writeHead(200, { 'Content-Type': 'text/event-stream' });
      res.write('data: {"type":"partial"}\n\n');
      // hangs after first event
    });
    const port = await listen(partialServer);

    const events = await captureSSE(
      `http://127.0.0.1:${port}/sse`,
      {},
      { maxEvents: 10, timeoutMs: 400 },
    );

    await close(partialServer);

    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({ data: '{"type":"partial"}' });
  });
});

describe('captureSSE — auth header forwarding', () => {
  it('forwards Authorization header to SSE endpoint', async () => {
    let receivedAuth = '';
    const authServer = http.createServer((req, res) => {
      receivedAuth = req.headers['authorization'] ?? '';
      res.writeHead(200, { 'Content-Type': 'text/event-stream' });
      res.write('data: {"ok":true}\n\n');
      res.end();
    });
    const port = await listen(authServer);

    await captureSSE(
      `http://127.0.0.1:${port}/sse`,
      { authorization: 'Bearer test-token-xyz' },
      { maxEvents: 5, timeoutMs: 1000 },
    );

    await close(authServer);

    expect(receivedAuth).toBe('Bearer test-token-xyz');
  });
});

describe('captureSSE — multi-line data fields', () => {
  it('parses data: lines arriving in the same chunk', async () => {
    // Two events sent as a single write (simulates chunked delivery)
    const batchServer = http.createServer((_req, res) => {
      res.writeHead(200, { 'Content-Type': 'text/event-stream' });
      // Two events in one write
      res.write('data: {"n":1}\n\ndata: {"n":2}\n\n');
      res.end();
    });
    const port = await listen(batchServer);

    const events = await captureSSE(
      `http://127.0.0.1:${port}/sse`,
      {},
      { maxEvents: 10, timeoutMs: 1000 },
    );

    await close(batchServer);

    expect(events).toHaveLength(2);
    expect(events[0]).toMatchObject({ data: '{"n":1}' });
    expect(events[1]).toMatchObject({ data: '{"n":2}' });
  });
});

describe('captureSSE — non-200 response', () => {
  it('returns empty array when server responds with non-200 status', async () => {
    const errorServer = http.createServer((_req, res) => {
      res.writeHead(401, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'unauthorized' }));
    });
    const port = await listen(errorServer);

    const events = await captureSSE(
      `http://127.0.0.1:${port}/sse`,
      {},
      { maxEvents: 5, timeoutMs: 1000 },
    );

    await close(errorServer);

    expect(events).toHaveLength(0);
  });
});

describe('captureSSE — SSEEvent shape', () => {
  it('each captured event has data and index fields', async () => {
    const simpleServer = makeSseServer(['{"type":"hello"}', '{"type":"world"}']);
    const port = await listen(simpleServer);

    const events: SSEEvent[] = await captureSSE(
      `http://127.0.0.1:${port}/sse`,
      {},
      { maxEvents: 10, timeoutMs: 1000 },
    );

    await close(simpleServer);

    expect(events[0]).toHaveProperty('data');
    expect(events[0]).toHaveProperty('index', 0);
    expect(events[1]).toHaveProperty('index', 1);
  });
});
