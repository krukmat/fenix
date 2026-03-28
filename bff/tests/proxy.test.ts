// Task 4.1 — FR-301: Proxy pass-through tests (TDD — written before implementation)
// Strategy: mock http-proxy-middleware at module level (factory mock).
// The proxy middleware is created at module import time, so the mock must be set up
// via jest.mock factory function (hoisted before imports).
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyHandlerFn = jest.fn((req, res, _next) => {
  // Default: simulate auth-required endpoint
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    res.status(401).json({ message: 'Unauthorized' });
    return;
  }
  res.status(200).json([{ id: 'acc-1', name: 'Acme Corp' }]);
});
const proxyHandler = makeProxyStub();

jest.mock('http-proxy-middleware', () => ({
  // createProxyMiddleware is called at module import time in routes/proxy.ts
  // Return the proxyHandler stub immediately (factory mock)
  createProxyMiddleware: jest.fn(() => proxyHandler),
}));

// Mock goClient for health/other routes
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 10 }),
}));

import app from '../src/app';

describe('Proxy pass-through /bff/api/v1/*', () => {
  afterEach(() => {
    proxyHandlerFn.mockClear();
  });

  it('forwards GET /bff/api/v1/accounts with Bearer token to Go and returns same response', async () => {
    const res = await request(app)
      .get('/bff/api/v1/accounts')
      .set('Authorization', 'Bearer test-token-123');

    expect(res.status).toBe(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('passes through 401 from Go when no auth token provided', async () => {
    const res = await request(app).get('/bff/api/v1/accounts');

    expect(res.status).toBe(401);
  });
});
