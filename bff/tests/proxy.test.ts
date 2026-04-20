// Task 4.1 — FR-301: Proxy pass-through tests (TDD — written before implementation)
// Strategy: mock http-proxy-middleware at module level (factory mock).
// The proxy middleware is created at module import time, so the mock must be set up
// via jest.mock factory function (hoisted before imports).
import http from 'http';
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
const proxyHandler = makeProxyStub(proxyHandlerFn);

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

// Import after mock setup to avoid module init order issues
// eslint-disable-next-line @typescript-eslint/no-require-imports
const { hasParsedJsonBody, restreamParsedJsonBody } = require('../src/routes/proxy') as typeof import('../src/routes/proxy');

describe('hasParsedJsonBody', () => {
  it('returns true when body has keys', () => {
    expect(hasParsedJsonBody({ body: { name: 'Acme' } } as unknown as import('express').Request)).toBe(true);
  });

  it('returns false when body is undefined', () => {
    expect(hasParsedJsonBody({ body: undefined } as unknown as import('express').Request)).toBe(false);
  });

  it('returns false when body is null', () => {
    expect(hasParsedJsonBody({ body: null } as unknown as import('express').Request)).toBe(false);
  });

  it('returns false when body is empty object', () => {
    expect(hasParsedJsonBody({ body: {} } as unknown as import('express').Request)).toBe(false);
  });
});

describe('restreamParsedJsonBody', () => {
  function makeProxyReq() {
    return {
      setHeader: jest.fn(),
      write: jest.fn(),
    } as unknown as http.ClientRequest;
  }

  it('writes body to proxyReq when body has content', () => {
    const proxyReq = makeProxyReq();
    const req = { body: { subject: 'Test' } } as unknown as import('express').Request;
    restreamParsedJsonBody(proxyReq, req);
    expect(proxyReq.setHeader).toHaveBeenCalledWith('Content-Type', 'application/json');
    expect(proxyReq.write).toHaveBeenCalledWith(JSON.stringify({ subject: 'Test' }));
  });

  it('does nothing when body is empty', () => {
    const proxyReq = makeProxyReq();
    const req = { body: undefined } as unknown as import('express').Request;
    restreamParsedJsonBody(proxyReq, req);
    expect(proxyReq.setHeader).not.toHaveBeenCalled();
    expect(proxyReq.write).not.toHaveBeenCalled();
  });
});
