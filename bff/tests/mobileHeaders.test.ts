// Task 4.1 — FR-301: mobileHeaders middleware tests
import request from 'supertest';
import express, { Request, Response } from 'express';
import { mobileHeaders } from '../src/middleware/mobileHeaders';

// Mock http-proxy-middleware
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyHandlerFn = jest.fn((_req: any, _res: any, next: any) => next());
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyStub = Object.assign(proxyHandlerFn, { upgrade: () => {} }) as any;
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 10 }),
}));

type BffReq = Request & { mobileHeaders?: Record<string, string> };

function buildHeaderTestApp() {
  const app = express();
  app.use(mobileHeaders);
  app.get('/test', (req: BffReq, res: Response) => {
    res.json({ headers: (req as BffReq).mobileHeaders ?? {} });
  });
  return app;
}

describe('mobileHeaders middleware', () => {
  it('extracts string mobile headers from request', async () => {
    const app = buildHeaderTestApp();
    const res = await request(app)
      .get('/test')
      .set('x-device-id', 'device-abc')
      .set('x-app-version', '1.2.3');

    expect(res.status).toBe(200);
    expect(res.body.headers['x-device-id']).toBe('device-abc');
    expect(res.body.headers['x-app-version']).toBe('1.2.3');
  });

  it('ignores non-string header values (does not add arrays)', async () => {
    // When a header value is not a string (e.g. not present), it should not appear
    const app = buildHeaderTestApp();
    const res = await request(app).get('/test');

    expect(res.status).toBe(200);
    // No mobile headers set → empty object
    expect(res.body.headers).toEqual({});
  });
});
