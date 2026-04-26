// bff-http-snapshots T1+T5: health gate + REST runner tests
import http from 'http';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { checkHealth } from '../../scripts/snapshots/health';
import { runEntries } from '../../scripts/snapshots/runner';
import type { CatalogEntry, SeederOutput } from '../../scripts/snapshots/types';

// ── T1: health gate tests ────────────────────────────────────────────────────

type FetchMock = jest.MockedFunction<typeof fetch>;

const originalFetch = global.fetch;

afterEach(() => {
  global.fetch = originalFetch;
});

describe('checkHealth', () => {
  it('returns ok=true when endpoint responds with 2xx', async () => {
    (global as unknown as { fetch: FetchMock }).fetch = jest.fn().mockResolvedValue(
      { ok: true, status: 200 }
    );

    const result = await checkHealth('http://localhost:8080/health', 'Go backend');

    expect(result.ok).toBe(true);
    expect(result.url).toBe('http://localhost:8080/health');
    expect(result.error).toBeUndefined();
  });

  it('returns ok=false with error message when endpoint returns non-2xx', async () => {
    (global as unknown as { fetch: FetchMock }).fetch = jest.fn().mockResolvedValue(
      { ok: false, status: 503 }
    );

    const result = await checkHealth('http://localhost:8080/health', 'Go backend');

    expect(result.ok).toBe(false);
    expect(result.error).toMatch(/Go backend.*503/);
  });

  it('returns ok=false with error message when fetch throws (unreachable)', async () => {
    (global as unknown as { fetch: FetchMock }).fetch = jest.fn().mockRejectedValue(
      new Error('ECONNREFUSED')
    );

    const result = await checkHealth('http://localhost:3000/bff/health', 'BFF');

    expect(result.ok).toBe(false);
    expect(result.error).toMatch(/BFF.*unreachable/i);
  });

  it('returns ok=false when fetch times out', async () => {
    const timeoutErr = Object.assign(new Error('TimeoutError'), { name: 'TimeoutError' });
    (global as unknown as { fetch: FetchMock }).fetch = jest.fn().mockRejectedValue(timeoutErr);

    const result = await checkHealth('http://localhost:8080/health', 'Go backend');

    expect(result.ok).toBe(false);
    expect(result.error).toMatch(/Go backend.*unreachable/i);
  });
});

describe('health gate — abort on failure', () => {
  it('reports which dependency is unreachable', async () => {
    const mockFetch = jest.fn()
      .mockResolvedValueOnce({ ok: true, status: 200 })
      .mockRejectedValueOnce(new Error('ECONNREFUSED'));
    (global as unknown as { fetch: FetchMock }).fetch = mockFetch;

    const goResult = await checkHealth('http://localhost:8080/health', 'Go backend');
    const bffResult = await checkHealth('http://localhost:3000/bff/health', 'BFF');

    expect(goResult.ok).toBe(true);
    expect(bffResult.ok).toBe(false);
    expect(bffResult.error).toMatch(/BFF/);
  });
});

// ── T5: REST runner tests ────────────────────────────────────────────────────

const UUID_USER = '550e8400-e29b-41d4-a716-446655440001';
const UUID_WS   = '550e8400-e29b-41d4-a716-446655440002';

const SEED: SeederOutput = {
  credentials: { email: 'e2e@test.com', password: 'pass' },
  auth: { token: 'tok-abc', userId: UUID_USER, workspaceId: UUID_WS },
  account: { id: '550e8400-e29b-41d4-a716-446655440003' },
  contact: { id: '550e8400-e29b-41d4-a716-446655440004', email: 'c@t.com' },
  lead: { id: '550e8400-e29b-41d4-a716-446655440005' },
  deal: { id: '550e8400-e29b-41d4-a716-446655440006' },
  pipeline: { id: '550e8400-e29b-41d4-a716-446655440007' },
  stage: { id: '550e8400-e29b-41d4-a716-446655440008' },
  staleDeal: { id: '550e8400-e29b-41d4-a716-446655440009' },
  case: { id: '550e8400-e29b-41d4-a716-44665544000a', subject: 'Test' },
  resolvedCase: { id: '550e8400-e29b-41d4-a716-44665544000b', subject: 'Resolved' },
  agentRuns: {
    completedId: '550e8400-e29b-41d4-a716-44665544000c',
    handoffId: '550e8400-e29b-41d4-a716-44665544000d',
    deniedByPolicyId: '550e8400-e29b-41d4-a716-44665544000e',
  },
  inbox: {
    approvalId: '550e8400-e29b-41d4-a716-44665544000f',
    rejectApprovalId: '550e8400-e29b-41d4-a716-446655440012',
    signalId: '550e8400-e29b-41d4-a716-446655440010',
  },
  workflow: { id: '550e8400-e29b-41d4-a716-446655440011' },
};

function makeServer(statusCode: number, body: unknown): http.Server {
  return http.createServer((_req, res) => {
    res.writeHead(statusCode, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(body));
  });
}

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

describe('runEntries — REST runner', () => {
  let server: http.Server;
  let baseUrl: string;
  let outputDir: string;

  beforeEach(async () => {
    server = makeServer(200, { ok: true, userId: UUID_USER });
    const port = await listen(server);
    baseUrl = `http://127.0.0.1:${port}`;
    outputDir = fs.mkdtempSync(path.join(os.tmpdir(), 'fenix-snapshots-'));
  });

  afterEach(async () => {
    await close(server);
    fs.rmSync(outputDir, { recursive: true, force: true });
  });

  const makeEntry = (overrides: Partial<CatalogEntry> = {}): CatalogEntry => ({
    name: 'test-entry',
    group: 'test',
    method: 'GET',
    path: '/bff/health',
    auth: false,
    expectedStatus: 200,
    ...overrides,
  });

  it('writes a JSON artifact for each non-SSE entry', async () => {
    const entries: CatalogEntry[] = [
      makeEntry({ name: 'entry-one', group: 'alpha' }),
      makeEntry({ name: 'entry-two', group: 'beta' }),
    ];

    await runEntries(entries, SEED, baseUrl, outputDir);

    expect(fs.existsSync(path.join(outputDir, 'alpha', 'entry-one.json'))).toBe(true);
    expect(fs.existsSync(path.join(outputDir, 'beta', 'entry-two.json'))).toBe(true);
  });

  it('artifact contains redacted request and response fields', async () => {
    const entries: CatalogEntry[] = [
      makeEntry({ name: 'auth-entry', group: 'auth', auth: true }),
    ];

    await runEntries(entries, SEED, baseUrl, outputDir);

    const raw = fs.readFileSync(path.join(outputDir, 'auth', 'auth-entry.json'), 'utf8');
    const artifact = JSON.parse(raw);

    expect(artifact.name).toBe('auth-entry');
    expect(artifact.method).toBe('GET');
    expect(artifact.response.status).toBe(200);
    expect(artifact.latencyMs).toBe('<duration>');
    // token must be redacted
    expect(artifact.request.headers['authorization']).toBe('Bearer <REDACTED>');
    expect(artifact.request.headers['x-real-ip']).toBe('<snapshot-ip>');
    // volatile UUIDs in response body must be redacted
    expect(artifact.response.body?.userId).toBe('<uuid:1>');
    // timestamp must be redacted
    expect(artifact.capturedAt).toBe('<timestamp>');
  });

  it('resolves body function with seed before sending request', async () => {
    let receivedBody: unknown;
    server.on('request', async (req, _res) => {
      const chunks: Buffer[] = [];
      for await (const chunk of req) chunks.push(chunk as Buffer);
      receivedBody = JSON.parse(Buffer.concat(chunks).toString() || '{}');
    });

    const entries: CatalogEntry[] = [
      makeEntry({
        name: 'body-entry',
        group: 'test',
        method: 'POST',
        body: (seed: SeederOutput) => ({ dealId: seed.deal.id }),
      }),
    ];

    await runEntries(entries, SEED, baseUrl, outputDir);

    expect(receivedBody).toEqual({ dealId: '550e8400-e29b-41d4-a716-446655440006' });
  });

  it('resolves pathParams and substitutes :param in path', async () => {
    let receivedPath = '';
    const customServer = makeServer(200, { ok: true });
    const port = await listen(customServer);
    const customBase = `http://127.0.0.1:${port}`;
    customServer.on('request', (req, _res) => { receivedPath = req.url ?? ''; });

    const entries: CatalogEntry[] = [
      makeEntry({
        name: 'param-entry',
        group: 'approvals',
        method: 'POST',
        path: '/bff/api/v1/approvals/:id/approve',
        pathParams: (seed: SeederOutput) => ({ id: seed.inbox.approvalId }),
      }),
    ];

    await runEntries(entries, SEED, customBase, outputDir);

    expect(receivedPath).toBe('/bff/api/v1/approvals/550e8400-e29b-41d4-a716-44665544000f/approve');
    await close(customServer);
  });

  it('routes SSE entries to .sse.json artifact (T6)', async () => {
    // Build an in-process SSE server that sends 2 events then closes
    const sseServer = http.createServer((_req, res) => {
      res.writeHead(200, { 'Content-Type': 'text/event-stream' });
      res.write('data: {"type":"chunk"}\n\n');
      res.write('data: {"type":"done"}\n\n');
      res.end();
    });
    const ssePort = await listen(sseServer);
    const sseBase = `http://127.0.0.1:${ssePort}`;

    const entries: CatalogEntry[] = [
      makeEntry({
        name: 'sse-entry',
        group: 'copilot',
        path: '/sse',
        sse: { maxEvents: 5, timeoutMs: 2000 },
      }),
      makeEntry({ name: 'rest-entry', group: 'rest' }),
    ];

    // SSE entry uses sseBase, REST entry uses the regular server baseUrl
    // We run them separately to control which base URL each entry sees
    const sseEntries = entries.filter((e) => e.sse);
    const restEntries = entries.filter((e) => !e.sse);

    await runEntries(sseEntries, SEED, sseBase, outputDir);
    await runEntries(restEntries, SEED, baseUrl, outputDir);

    await close(sseServer);

    // SSE artifact written as <name>.sse.json
    expect(fs.existsSync(path.join(outputDir, 'copilot', 'sse-entry.sse.json'))).toBe(true);
    // REST artifact written as <name>.json
    expect(fs.existsSync(path.join(outputDir, 'rest', 'rest-entry.json'))).toBe(true);

    const raw = fs.readFileSync(path.join(outputDir, 'copilot', 'sse-entry.sse.json'), 'utf8');
    const artifact = JSON.parse(raw);
    expect(Array.isArray(artifact.response.body)).toBe(true);
    expect(artifact.response.body).toHaveLength(2);
    expect(artifact.capturedAt).toBe('<timestamp>');
  });

  it('records a failed entry when response status does not match expectedStatus', async () => {
    server.removeAllListeners('request');
    server.on('request', (_req, res) => {
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'internal' }));
    });

    const entries: CatalogEntry[] = [
      makeEntry({ name: 'fail-entry', group: 'test', expectedStatus: 200 }),
    ];

    const results = await runEntries(entries, SEED, baseUrl, outputDir);

    expect(results[0]?.pass).toBe(false);
    expect(results[0]?.name).toBe('fail-entry');
  });
});
