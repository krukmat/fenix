// Task 4.1 — FR-301: Error handler middleware tests
import request from 'supertest';
import express, { Request, Response, NextFunction } from 'express';
import { errorHandler } from '../src/middleware/errorHandler';
import { makeProxyStub } from './helpers/proxyStub';

// Mock http-proxy-middleware
const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 10 }),
}));

function buildErrorApp(next_err: unknown) {
  const app = express();
  app.get('/test', (_req: Request, _res: Response, next: NextFunction) => {
    next(next_err);
  });
  app.use(errorHandler);
  return app;
}

describe('errorHandler middleware', () => {
  it('returns 500 with INTERNAL_ERROR for generic Error instances', async () => {
    const app = buildErrorApp(new Error('something broke'));
    const res = await request(app).get('/test');
    expect(res.status).toBe(500);
    expect(res.body.error.code).toBe('INTERNAL_ERROR');
    expect(res.body.error.message).toBe('something broke');
  });

  it('returns 500 with UNKNOWN_ERROR for non-Error thrown values (string)', async () => {
    const app = buildErrorApp('just a string error');
    const res = await request(app).get('/test');
    expect(res.status).toBe(500);
    expect(res.body.error.code).toBe('UNKNOWN_ERROR');
  });

  it('returns BACKEND_ERROR with status from Axios-like error', async () => {
    const axiosLikeError = Object.assign(new Error('Not Found'), {
      isAxiosError: true,
      response: { status: 404, data: { message: 'Resource not found' } },
    });
    const app = buildErrorApp(axiosLikeError);
    const res = await request(app).get('/test');
    expect(res.status).toBe(404);
    expect(res.body.error.code).toBe('BACKEND_ERROR');
    expect(res.body.error.message).toBe('Resource not found');
  });

  it('returns 502 when Axios error has no response', async () => {
    const axiosLikeError = Object.assign(new Error('Network Error'), {
      isAxiosError: true,
      response: undefined,
    });
    const app = buildErrorApp(axiosLikeError);
    const res = await request(app).get('/test');
    expect(res.status).toBe(502);
    expect(res.body.error.code).toBe('BACKEND_ERROR');
  });
});
