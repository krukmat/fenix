// BFF-ADMIN-90: end-to-end navigation smoke test — walks all 8 admin routes,
// asserts HTTP 200 and presence of the shared layout landmark (page-title class)
// and a route-specific heading on each page. Catches regressions in the shared
// chrome without duplicating the per-route unit tests.
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = { get: jest.fn(), put: jest.fn() };
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

// ─── Minimal stubs — enough for each route to reach its render path ──────────

// Workflows handler expects data to be WorkflowRow[] directly (not wrapped in { data: [] })
const WORKFLOW_STUB = [
  { id: 'wf-1', name: 'smoke_wf', status: 'active', version: 1, description: null, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
];

// Agent runs handler expects { data: AgentRunRow[], meta: {...} }
const AGENT_RUNS_STUB = {
  data: [{ id: 'run-1', agentDefinitionId: 'agent-1', status: 'success', triggerType: 'manual', startedAt: '2026-01-01T00:00:00Z', createdAt: '2026-01-01T00:00:00Z' }],
  meta: { total: 1, limit: 50, offset: 0 },
};

const APPROVALS_STUB = {
  data: [{ id: 'appr-1', toolName: 'send_email', requestedBy: 'user-1', status: 'pending', createdAt: '' }],
  meta: { total: 1 },
};

const AUDIT_STUB = {
  data: [{ id: 'evt-1', actor: 'user-1', action: 'read', resourceType: 'case', resourceId: 'c-1', createdAt: '' }],
  meta: { total: 1, limit: 50, offset: 0 },
};

const GOVERNANCE_STUB = {
  quotaStates: [{ agentId: 'agent-1', metric: 'tokens_per_day', used: 100, limit: 1000, status: 'ok' }],
  recentUsage: [{ agentId: 'agent-1', tokensUsed: 100, costEuros: 0.01, createdAt: '' }],
};

const POLICY_SETS_STUB = {
  data: [{ id: 'ps-1', name: 'Default Policy', version: 1, active: true, createdAt: '' }],
};

// Tools handler expects { data: Tool[] } where Tool has id, name, description, active, createdAt
const TOOLS_STUB = {
  data: [{ id: 'tool-1', name: 'send_email', description: 'Send an email', active: true, createdAt: '2026-01-01T00:00:00Z' }],
};

const PROMETHEUS_STUB =
  '# HELP fenixcrm_requests_total\nfenixcrm_requests_total 42\n' +
  '# HELP fenixcrm_request_errors_total\nfenixcrm_request_errors_total 1\n' +
  '# HELP fenixcrm_uptime_seconds\nfenixcrm_uptime_seconds 3600\n';

// ─── Smoke test suite ────────────────────────────────────────────────────────

describe('BFF admin navigation smoke test — BFF-ADMIN-90', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin (dashboard)', () => {
    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains the dashboard page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      expect(res.text).toContain('Dashboard');
    });
  });

  describe('GET /bff/admin/workflows', () => {
    beforeEach(() => {
      mockGoClient.get.mockResolvedValue({ data: WORKFLOW_STUB, status: 200 });
    });

    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains the Workflows page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      expect(res.text).toContain('Workflows');
    });
  });

  describe('GET /bff/admin/agent-runs', () => {
    beforeEach(() => {
      mockGoClient.get.mockResolvedValue({ data: AGENT_RUNS_STUB, status: 200 });
    });

    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains the Agent Runs page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      expect(res.text).toContain('Agent Runs');
    });
  });

  describe('GET /bff/admin/approvals', () => {
    beforeEach(() => {
      mockGoClient.get.mockResolvedValue({ data: APPROVALS_STUB, status: 200 });
    });

    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains the Approvals page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      expect(res.text).toContain('Approvals');
    });
  });

  describe('GET /bff/admin/audit', () => {
    beforeEach(() => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_STUB, status: 200 });
    });

    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains the Audit Trail page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      expect(res.text).toContain('Audit Trail');
    });
  });

  describe('GET /bff/admin/policy', () => {
    beforeEach(() => {
      // policy route calls governance summary first, then policy sets
      mockGoClient.get
        .mockResolvedValueOnce({ data: GOVERNANCE_STUB, status: 200 })
        .mockResolvedValueOnce({ data: POLICY_SETS_STUB, status: 200 });
    });

    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains a policy page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin/policy')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      // Governance summary page uses "Quota States" as its first h2
      expect(res.text).toMatch(/Quota States|Policy Sets|Governance/i);
    });
  });

  describe('GET /bff/admin/tools', () => {
    beforeEach(() => {
      mockGoClient.get.mockResolvedValue({ data: TOOLS_STUB, status: 200 });
    });

    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains the Tools page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin/tools')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      expect(res.text).toContain('Tools');
    });
  });

  describe('GET /bff/admin/metrics', () => {
    beforeEach(() => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_STUB, status: 200 });
    });

    it('returns 200 with HTML content-type', async () => {
      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains the Metrics page-title landmark', async () => {
      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('page-title');
      expect(res.text).toContain('Metrics');
    });
  });
});
