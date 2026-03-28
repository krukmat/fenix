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

  describe('Auth flow: login -> protected endpoint', () => {
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

  describe('Removed custom BFF CRM routes', () => {
    it('returns 404 for legacy aggregated account route', async () => {
      const res = await request(app)
        .get('/bff/accounts/acc-fullstack-test/full')
        .set('Authorization', 'Bearer mock-token');

      expect(res.status).toBe(404);
    });
  });
});
