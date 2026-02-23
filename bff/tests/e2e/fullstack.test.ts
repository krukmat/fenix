import request from 'supertest';

// Mock http-proxy-middleware before importing app (proxy.ts imports at module-level)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyHandlerFn = jest.fn((_req: any, _res: any, next: any) => next());
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyStub = Object.assign(proxyHandlerFn, { upgrade: () => {} }) as any;
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

jest.mock('../../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import { createGoClient } from '../../src/services/goClient';
import app from '../../src/app';

const mockCreateGoClient = createGoClient as jest.MockedFunction<typeof createGoClient>;

describe('BFF fullstack integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Auth flow: login → protected endpoint', () => {
    it('login returns JWT and token can be sent to a protected proxy route', async () => {
      const mockToken = 'eyJhbGciOiJIUzI1NiJ9.fullstack-token';
      const mockUser = { id: 'user-1', email: 'test@example.com', workspace_id: 'ws-1' };

      mockCreateGoClient.mockReturnValueOnce({
        post: jest.fn().mockResolvedValue({
          data: { token: mockToken, user: mockUser },
          status: 200,
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const loginRes = await request(app)
        .post('/bff/auth/login')
        .send({ email: 'test@example.com', password: 'password123' });

      expect(loginRes.status).toBe(200);
      expect(loginRes.body.token).toBeDefined();

      const protectedRes = await request(app)
        .get('/bff/api/v1/accounts')
        .set('Authorization', `Bearer ${loginRes.body.token}`);

      // mocked proxy middleware calls next(), app then ends in 404
      expect([200, 404]).toContain(protectedRes.status);
    });
  });

  describe('Aggregated endpoint: GET /bff/accounts/:id/full', () => {
    it('returns merged account + contacts + deals + timeline', async () => {
      const accountID = 'acc-fullstack-test';

      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path === `/api/v1/accounts/${accountID}`) {
            return Promise.resolve({ data: { id: accountID, name: 'Acme Corp' } });
          }
          if (path.includes('/contacts')) {
            return Promise.resolve({ data: { items: [{ id: 'c-1', name: 'John' }] } });
          }
          if (path.includes('/deals')) {
            return Promise.resolve({ data: { items: [] } });
          }
          if (path.includes('timeline')) {
            return Promise.resolve({ data: { items: [] } });
          }
          return Promise.resolve({ data: null });
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/accounts/${accountID}/full`)
        .set('Authorization', 'Bearer mock-token');

      expect(res.status).toBe(200);
      expect(res.body.account).toMatchObject({ id: accountID, name: 'Acme Corp' });
      expect(res.body.contacts).toMatchObject({ items: expect.any(Array) });
      expect(res.body.deals).toBeDefined();
      expect(res.body.timeline).toBeDefined();
    });

    it('returns null for failed sub-call but still responds 200', async () => {
      const accountID = 'acc-partial-test';

      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path === `/api/v1/accounts/${accountID}`) {
            return Promise.resolve({ data: { id: accountID, name: 'Partial Corp' } });
          }
          return Promise.reject(new Error('service unavailable'));
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/accounts/${accountID}/full`)
        .set('Authorization', 'Bearer mock-token');

      expect(res.status).toBe(200);
      expect(res.body.account).toMatchObject({ id: accountID });
      expect(res.body.contacts).toBeNull();
    });
  });

  describe('SSE copilot proxy: POST /bff/copilot/chat', () => {
    it('sets SSE headers and relays stream from Go backend', async () => {
      const nock = require('nock');
      const backendURL = process.env.BACKEND_URL || 'http://localhost:8080';

      nock(backendURL)
        .post('/api/v1/copilot/chat')
        .reply(200, 'data: {"token":"hello"}\n\ndata: [DONE]\n\n', {
          'Content-Type': 'text/event-stream',
        });

      const res = await request(app)
        .post('/bff/copilot/chat')
        .set('Authorization', 'Bearer mock-token')
        .send({ case_id: 'case-1', message: 'test question' });

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/event-stream/);
      nock.cleanAll();
    });
  });

  describe('Error handling: Go backend unavailable', () => {
    it('returns 502 when Go backend rejects connection', async () => {
      const axiosError = Object.assign(new Error('ECONNREFUSED'), {
        code: 'ECONNREFUSED',
        isAxiosError: true,
        response: undefined,
      });

      mockCreateGoClient.mockReturnValue({
        post: jest.fn().mockRejectedValue(axiosError),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .post('/bff/auth/login')
        .send({ email: 'test@example.com', password: 'pass' });

      expect(res.status).toBe(502);
    });
  });
});
