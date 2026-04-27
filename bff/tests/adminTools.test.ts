// BFF-ADMIN-60: tools list
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = { get: jest.fn() };
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

const TOOLS_LIST = {
  data: [
    { id: 'tool-001', name: 'create_task', description: 'Create a task', active: true, createdAt: '2026-04-20T10:00:00Z' },
    { id: 'tool-002', name: 'send_email', description: 'Send an email', active: true, createdAt: '2026-04-18T08:30:00Z' },
    { id: 'tool-003', name: 'old_tool', description: 'Deprecated', active: false, createdAt: '2026-01-01T00:00:00Z' },
  ],
};

describe('BFF admin tools — BFF-ADMIN-60', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/tools', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/admin/tools', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/admin/tools');
    });

    it('renders tool names', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('create_task');
      expect(res.text).toContain('send_email');
      expect(res.text).toContain('old_tool');
    });

    it('renders tool IDs', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('tool-001');
      expect(res.text).toContain('tool-002');
      expect(res.text).toContain('tool-003');
    });

    it('renders tool descriptions', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Create a task');
      expect(res.text).toContain('Send an email');
      expect(res.text).toContain('Deprecated');
    });

    it('renders active badge for active tools', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('active');
    });

    it('renders inactive badge for inactive tools', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      const inactiveCount = (res.text.match(/inactive/g) || []).length;
      expect(inactiveCount).toBeGreaterThan(0);
    });

    it('renders timestamps', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('2026-04-20');
      expect(res.text).toContain('2026-04-18');
    });

    it('renders empty-state when list is empty', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [] }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No tools');
    });

    it('renders no mutation affordance (read-only — no POST form in content)', async () => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_LIST, status: 200 });

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).not.toContain('method="POST"');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('timeout'));

      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});
