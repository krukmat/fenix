// Task 4.1 — FR-301: Health endpoint tests (TDD — written before implementation)
import request from 'supertest';

// Mock goClient before importing app to control Go backend reachability
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn(),
}));

import { pingGoBackend } from '../src/services/goClient';
import app from '../src/app';

const mockPingGoBackend = pingGoBackend as jest.MockedFunction<typeof pingGoBackend>;

describe('GET /bff/health', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('returns 200 with status ok when Go backend is reachable', async () => {
    mockPingGoBackend.mockResolvedValue({ reachable: true, latencyMs: 42 });

    const res = await request(app).get('/bff/health');

    expect(res.status).toBe(200);
    expect(res.body).toMatchObject({
      status: 'ok',
      backend: 'reachable',
    });
    expect(typeof res.body.latency_ms).toBe('number');
  });

  it('returns 503 with status degraded when Go backend is unreachable', async () => {
    mockPingGoBackend.mockResolvedValue({ reachable: false, latencyMs: 2001 });

    const res = await request(app).get('/bff/health');

    expect(res.status).toBe(503);
    expect(res.body).toMatchObject({
      status: 'degraded',
      backend: 'unreachable',
    });
  });
});
