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

const mockCreateGoClient = createGoClient as jest.MockedFunction<typeof createGoClient>;

describe('POST /bff/copilot/chat (SSE relay)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
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

    // Emit SSE data after a tick (simulates Go streaming)
    setImmediate(() => {
      mockStream.write('data: {"type":"token","content":"Hello"}\n\n');
      mockStream.write('data: {"type":"token","content":" world"}\n\n');
      mockStream.end();
    });

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
});

describe('POST /bff/api/v1/copilot/sales-brief', () => {
  beforeEach(() => {
    jest.clearAllMocks();
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
});
