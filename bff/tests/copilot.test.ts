// Task 4.1 — FR-301: SSE proxy tests (TDD — written before implementation)
import request from 'supertest';
import { PassThrough } from 'stream';
import { makeProxyStub } from './helpers/proxyStub';

// Mock http-proxy-middleware (required at module import in routes/proxy.ts)
const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

// Mock goClient ping
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 10 }),
}));

// Mock axios to control SSE stream
jest.mock('axios');
import axios from 'axios';
const mockAxios = axios as jest.Mocked<typeof axios>;

import app from '../src/app';
import { createGoClient } from '../src/services/goClient';
import { resetScreenshotModeStateForTests } from '../src/routes/copilot';

const mockCreateGoClient = createGoClient as jest.MockedFunction<typeof createGoClient>;
const ORIGINAL_NODE_ENV = process.env['NODE_ENV'];
const ORIGINAL_SCREENSHOT_MODE = process.env['SCREENSHOT_MODE'];
const ORIGINAL_ENABLE_SCREENSHOT_FIXTURES = process.env['ENABLE_SCREENSHOT_FIXTURES'];

function resetCopilotFixtureEnv(): void {
  process.env['NODE_ENV'] = 'test';
  delete process.env['SCREENSHOT_MODE'];
  delete process.env['ENABLE_SCREENSHOT_FIXTURES'];
  resetScreenshotModeStateForTests();
}

describe('POST /bff/copilot/chat (SSE relay)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    resetCopilotFixtureEnv();
  });

  it('sets SSE headers and relays stream chunks from Go', (done) => {
    // Create a controllable stream to simulate Go SSE output
    const mockStream = new PassThrough();

    // Mock axios.post to return our controllable stream
    mockAxios.post.mockResolvedValue({
      data: mockStream,
      status: 200,
      headers: { 'content-type': 'text/event-stream' },
    });

    const chunks: string[] = [];

    // Use raw http request to receive SSE chunks without buffering
    const req = request(app)
      .post('/bff/copilot/chat')
      .set('Authorization', 'Bearer test-token')
      .set('Accept', 'text/event-stream')
      .send({ message: 'What are the latest cases?', entity_id: 'case-123' })
      .buffer(false)
      .parse((res, callback) => {
        res.on('data', (chunk: Buffer) => {
          chunks.push(chunk.toString());
        });
        res.on('end', () => callback(null, chunks.join('')));
        res.on('error', callback);
      });

    // Emit SSE data after a short delay — setImmediate is too tight under parallel test load
    setTimeout(() => {
      mockStream.write('data: {"type":"token","content":"Hello"}\n\n');
      mockStream.write('data: {"type":"token","content":" world"}\n\n');
      mockStream.end();
    }, 20);

    req.then((res) => {
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/event-stream/);
      const fullBody = chunks.join('');
      expect(fullBody).toContain('data: {"type":"token","content":"Hello"}');
      done();
    }).catch(done);
  });
});

describe('GET /bff/copilot/events (EventSource relay)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    resetCopilotFixtureEnv();
  });

  it('relays SSE chunks from Go through a browser-compatible GET endpoint', (done) => {
    const mockStream = new PassThrough();
    mockAxios.post.mockResolvedValue({
      data: mockStream,
      status: 200,
      headers: { 'content-type': 'text/event-stream' },
    });

    const chunks: string[] = [];
    const req = request(app)
      .get('/bff/copilot/events?message=Hello&entity_id=case-123&entity_type=case')
      .set('Authorization', 'Bearer browser-token')
      .set('Accept', 'text/event-stream')
      .buffer(false)
      .parse((res, callback) => {
        res.on('data', (chunk: Buffer) => chunks.push(chunk.toString()));
        res.on('end', () => callback(null, chunks.join('')));
        res.on('error', callback);
      });

    setImmediate(() => {
      mockStream.write('data: {"type":"token","content":"Browser"}\n\n');
      mockStream.end();
    });

    req.then((res) => {
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/event-stream/);
      expect(chunks.join('')).toContain('data: {"type":"token","content":"Browser"}');
      expect(mockAxios.post).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/copilot/chat'),
        { message: 'Hello', entity_id: 'case-123', entity_type: 'case' },
        expect.objectContaining({ responseType: 'stream' }),
      );
      done();
    }).catch(done);
  });

  it('returns a terminal SSE error event for persistent upstream failures', async () => {
    mockAxios.post.mockRejectedValue(new Error('Unauthorized'));

    const res = await request(app)
      .get('/bff/copilot/events?message=Hello')
      .set('Accept', 'text/event-stream');

    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/event-stream/);
    expect(res.text).toContain('retry: 0');
    expect(res.text).toContain('event: error');
    expect(res.text).toContain('sse_upstream_error');
  });

  it('relays real SSE upstream data even when SCREENSHOT_MODE=true without the explicit guard', async () => {
    process.env['SCREENSHOT_MODE'] = 'true';
    resetScreenshotModeStateForTests();

    const mockStream = new PassThrough();
    mockAxios.post.mockResolvedValue({
      data: mockStream,
      status: 200,
      headers: { 'content-type': 'text/event-stream' },
    });

    setImmediate(() => {
      mockStream.write('data: {"type":"token","content":"Real upstream"}\n\n');
      mockStream.end();
    });

    const res = await request(app)
      .get('/bff/copilot/events?message=Hello')
      .set('Accept', 'text/event-stream');

    expect(res.status).toBe(200);
    expect(res.text).toContain('Real upstream');
    expect(res.text).not.toContain('Snapshot fixture response.');
    expect(mockAxios.post).toHaveBeenCalled();
  });
});

describe('POST /bff/api/v1/copilot/sales-brief', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    resetCopilotFixtureEnv();
  });

  it('bypasses the transparent proxy, relays to Go, and unwraps the data envelope', async () => {
    const post = jest.fn().mockResolvedValue({
      data: {
        data: {
          outcome: 'completed',
          summary: 'Healthy pipeline',
        },
      },
      status: 200,
    });

    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/api/v1/copilot/sales-brief')
      .set('Authorization', 'Bearer brief-token')
      .send({ entityType: 'account', entityId: 'acc-1' });

    expect(res.status).toBe(200);
    expect(res.body).toEqual({
      outcome: 'completed',
      summary: 'Healthy pipeline',
    });
    expect(post).toHaveBeenCalledWith('/api/v1/copilot/sales-brief', {
      entityType: 'account',
      entityId: 'acc-1',
    });
    expect(proxyStub).not.toHaveBeenCalled();
  });

  it('relays to Go when SCREENSHOT_MODE=true but the explicit fixture guard is unset', async () => {
    process.env['SCREENSHOT_MODE'] = 'true';
    resetScreenshotModeStateForTests();

    const post = jest.fn().mockResolvedValue({
      data: {
        data: {
          outcome: 'completed',
          summary: 'Real upstream response',
        },
      },
      status: 200,
    });

    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/api/v1/copilot/sales-brief')
      .send({ entityType: 'account', entityId: 'acc-2' });

    expect(res.status).toBe(200);
    expect(res.body).toEqual({
      outcome: 'completed',
      summary: 'Real upstream response',
    });
    expect(post).toHaveBeenCalledTimes(1);
  });
});

describe('POST /bff/api/v1/copilot/internal/screenshot-mode', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    resetCopilotFixtureEnv();
  });

  afterAll(() => {
    process.env['NODE_ENV'] = ORIGINAL_NODE_ENV;
    if (ORIGINAL_SCREENSHOT_MODE === undefined) {
      delete process.env['SCREENSHOT_MODE'];
    } else {
      process.env['SCREENSHOT_MODE'] = ORIGINAL_SCREENSHOT_MODE;
    }
    if (ORIGINAL_ENABLE_SCREENSHOT_FIXTURES === undefined) {
      delete process.env['ENABLE_SCREENSHOT_FIXTURES'];
    } else {
      process.env['ENABLE_SCREENSHOT_FIXTURES'] = ORIGINAL_ENABLE_SCREENSHOT_FIXTURES;
    }
    resetScreenshotModeStateForTests();
  });

  it('returns 404 when the explicit fixture guard is unset', async () => {
    const res = await request(app)
      .post('/bff/api/v1/copilot/internal/screenshot-mode')
      .send({ enabled: true });

    expect(res.status).toBe(404);
    expect(res.body).toEqual({ error: 'Not Found' });
  });

  it('allows screenshot fixtures in local test mode when the explicit guard is enabled', async () => {
    process.env['ENABLE_SCREENSHOT_FIXTURES'] = 'true';
    process.env['NODE_ENV'] = 'test';
    resetScreenshotModeStateForTests();

    const enableRes = await request(app)
      .post('/bff/api/v1/copilot/internal/screenshot-mode')
      .send({ enabled: true });

    expect(enableRes.status).toBe(200);
    expect(enableRes.body).toEqual({ screenshotMode: true });

    const salesBriefRes = await request(app)
      .post('/bff/api/v1/copilot/sales-brief')
      .send({ entityType: 'deal', entityId: 'deal-1' });

    expect(salesBriefRes.status).toBe(200);
    expect(salesBriefRes.body).toMatchObject({
      outcome: 'completed',
      entityId: 'fixture',
      summary: expect.stringContaining('Champion confirmed budget approval'),
    });
    expect(mockCreateGoClient).not.toHaveBeenCalled();
  });

  it('returns 404 in production even if the explicit fixture guard is enabled', async () => {
    process.env['ENABLE_SCREENSHOT_FIXTURES'] = 'true';
    process.env['NODE_ENV'] = 'production';
    resetScreenshotModeStateForTests();

    const res = await request(app)
      .post('/bff/api/v1/copilot/internal/screenshot-mode')
      .send({ enabled: true });

    expect(res.status).toBe(404);
    expect(res.body).toEqual({ error: 'Not Found' });
  });
});
