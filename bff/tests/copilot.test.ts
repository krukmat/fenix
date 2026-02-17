// Task 4.1 — FR-301: SSE proxy tests (TDD — written before implementation)
import request from 'supertest';
import { PassThrough } from 'stream';

// Mock http-proxy-middleware (required at module import in routes/proxy.ts)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyHandlerFn = jest.fn((_req: any, _res: any, next: any) => next());
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyStub = Object.assign(proxyHandlerFn, { upgrade: () => {} }) as any;
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
