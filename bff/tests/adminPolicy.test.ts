// BFF-ADMIN-50 / BFF-ADMIN-51: governance summary + policy sets list + versions drill-down
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

// ─── Fixtures ────────────────────────────────────────────────────────────────

const GOVERNANCE_SUMMARY = {
  quotaStates: [
    { agentId: 'agent-001', metric: 'tokens_per_day', used: 4200, limit: 10000, status: 'ok' },
    { agentId: 'agent-002', metric: 'cost_per_day',   used: 8.5,  limit: 10.0,  status: 'warning' },
  ],
  recentUsage: [
    { agentId: 'agent-001', tokensUsed: 1200, costEuros: 0.06, createdAt: '2026-04-25T10:00:00Z' },
    { agentId: 'agent-002', tokensUsed: 800,  costEuros: 0.04, createdAt: '2026-04-25T09:30:00Z' },
  ],
};

const POLICY_SETS = {
  data: [
    { id: 'ps-001', name: 'Default Policy', version: 2, active: true,  createdAt: '2026-01-10T00:00:00Z' },
    { id: 'ps-002', name: 'No-Cloud Policy', version: 1, active: false, createdAt: '2026-02-15T00:00:00Z' },
  ],
};

const POLICY_VERSIONS = {
  data: [
    { id: 'pv-001', policySetId: 'ps-001', version: 2, active: true,  createdAt: '2026-03-01T00:00:00Z' },
    { id: 'pv-002', policySetId: 'ps-001', version: 1, active: false, createdAt: '2026-01-10T00:00:00Z' },
  ],
};

// ─── BFF-ADMIN-50: governance summary page ───────────────────────────────────

describe('BFF admin policy — governance summary — BFF-ADMIN-50', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/policy', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/governance/summary', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/governance/summary');
    });

    it('calls Go GET /api/v1/policy/sets', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/policy/sets');
    });

    it('renders quota agent IDs', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('agent-001');
      expect(res.text).toContain('agent-002');
    });

    it('renders quota metric names', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('tokens_per_day');
      expect(res.text).toContain('cost_per_day');
    });

    it('renders quota status badges', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('ok');
      expect(res.text).toContain('warning');
    });

    it('renders policy set names', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Default Policy');
      expect(res.text).toContain('No-Cloud Policy');
    });

    it('renders policy set IDs', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('ps-001');
      expect(res.text).toContain('ps-002');
    });

    it('renders versions drill-down link for each policy set', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_SUMMARY, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS,        status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/policy/ps-001/versions');
      expect(res.text).toContain('/bff/admin/policy/ps-002/versions');
    });

    it('renders empty-state when quota list is empty', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: { quotaStates: [], recentUsage: [] }, status: 200 })
        .mockResolvedValueOnce({ data: { data: [] },                         status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No quota states');
    });

    it('renders empty-state when policy set list is empty', async () => {
      mockGoClient.get
        .mockResolvedValueOnce({ data: { quotaStates: [], recentUsage: [] }, status: 200 })
        .mockResolvedValueOnce({ data: { data: [] },                         status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No policy sets');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('timeout'));

      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});

// ─── BFF-ADMIN-51: policy set versions drill-down ────────────────────────────

describe('BFF admin policy — versions drill-down — BFF-ADMIN-51', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/policy/:id/versions', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: POLICY_VERSIONS, status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/policy/sets/:id/versions', async () => {
      mockGoClient.get.mockResolvedValue({ data: POLICY_VERSIONS, status: 200 });

      await request(app)
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/policy/sets/ps-001/versions');
    });

    it('renders version numbers', async () => {
      mockGoClient.get.mockResolvedValue({ data: POLICY_VERSIONS, status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('pv-001');
      expect(res.text).toContain('pv-002');
    });

    it('renders active badge on active version', async () => {
      mockGoClient.get.mockResolvedValue({ data: POLICY_VERSIONS, status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('active');
    });

    it('renders back link to policy list', async () => {
      mockGoClient.get.mockResolvedValue({ data: POLICY_VERSIONS, status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/policy');
    });

    it('renders empty-state when no versions exist', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [] }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No versions');
    });

    it('renders no mutation affordance (read-only — no POST form in content)', async () => {
      mockGoClient.get.mockResolvedValue({ data: POLICY_VERSIONS, status: 200 });

      const res = await request(app)
        .get('/bff/admin/policy/ps-001/versions')
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
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('network failure'));

      const res = await request(app)
        .get('/bff/admin/policy/ps-001/versions')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});
