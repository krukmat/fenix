// BFF-ADMIN-40 / BFF-ADMIN-41: audit trail paginated list + record detail
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

// BFF-ADMIN-Task6: fields match Go snake_case response shape
const AUDIT_LIST = [
  {
    id: 'evt-001',
    actor_id: 'user-abc',
    actor_type: 'user',
    action: 'case.created',
    entity_type: 'case',
    entity_id: 'case-111',
    outcome: 'success',
    created_at: '2026-04-25T10:00:00Z',
  },
  {
    id: 'evt-002',
    actor_id: 'user-xyz',
    actor_type: 'user',
    action: 'approval.decided',
    entity_type: 'approval',
    entity_id: 'apr-002',
    outcome: 'success',
    created_at: '2026-04-24T08:30:00Z',
  },
];

const BACKEND_LIST_RESPONSE = {
  data: AUDIT_LIST,
  meta: { total: 2, nextCursor: 'cursor-abc' },
};

// ─── BFF-ADMIN-40: list ──────────────────────────────────────────────────────

describe('BFF admin audit list — BFF-ADMIN-40', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/audit', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/audit', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/audit/events',
        expect.objectContaining({ params: expect.any(Object) }),
      );
    });

    it('renders audit event IDs', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('evt-001');
      expect(res.text).toContain('evt-002');
    });

    it('renders actor values', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('user-abc');
      expect(res.text).toContain('user-xyz');
    });

    it('renders action values', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('case.created');
      expect(res.text).toContain('approval.decided');
    });

    it('renders resource type values', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('case');
      expect(res.text).toContain('approval');
    });

    it('relays actor filter to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/audit?actor=user-abc')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/audit/events',
        expect.objectContaining({ params: expect.objectContaining({ actor: 'user-abc' }) }),
      );
    });

    it('relays resource_type filter to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/audit?resource_type=case')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/audit/events',
        expect.objectContaining({ params: expect.objectContaining({ resource_type: 'case' }) }),
      );
    });

    it('relays date_from filter to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/audit?date_from=2026-04-01')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/audit/events',
        expect.objectContaining({ params: expect.objectContaining({ date_from: '2026-04-01' }) }),
      );
    });

    it('relays date_to filter to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/audit?date_to=2026-04-30')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/audit/events',
        expect.objectContaining({ params: expect.objectContaining({ date_to: '2026-04-30' }) }),
      );
    });

    it('relays cursor param to Go backend for pagination', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/audit?cursor=cursor-abc')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/audit/events',
        expect.objectContaining({ params: expect.objectContaining({ cursor: 'cursor-abc' }) }),
      );
    });

    it('renders next-page link when backend returns nextCursor', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('cursor-abc');
    });

    it('does not render next-page link when nextCursor is absent', async () => {
      mockGoClient.get.mockResolvedValue({
        data: { data: AUDIT_LIST, meta: { total: 2 } },
        status: 200,
      });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      // No "Next" pagination link rendered
      expect(res.text).not.toContain('Next');
    });

    it('renders empty-state when list is empty', async () => {
      mockGoClient.get.mockResolvedValue({
        data: { data: [], meta: { total: 0 } },
        status: 200,
      });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No audit events');
    });

    it('includes actor and resource filter form inputs', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_LIST_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('name="actor"');
      expect(res.text).toContain('name="resource_type"');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('timeout'));

      const res = await request(app)
        .get('/bff/admin/audit')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});

// ─── BFF-ADMIN-41: detail ────────────────────────────────────────────────────

const AUDIT_DETAIL = {
  id: 'evt-001',
  actor_id: 'user-abc',
  actor_type: 'user',
  action: 'case.created',
  entity_type: 'case',
  entity_id: 'case-111',
  outcome: 'success',
  permissions_checked: [{ rule: 'rbac:support_agent', result: 'allow' }],
  details: { ip: '10.0.0.1', userAgent: 'FenixMobile/1.0' },
  created_at: '2026-04-25T10:00:00Z',
};

describe('BFF admin audit detail — BFF-ADMIN-41', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/audit/:id', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/audit/:id', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/audit/events/evt-001');
    });

    it('renders the audit event id', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('evt-001');
    });

    it('renders actor and action', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('user-abc');
      expect(res.text).toContain('case.created');
    });

    it('renders resource type and id', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('case');
      expect(res.text).toContain('case-111');
    });

    it('renders outcome', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('success');
    });

    it('renders back link to audit list', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/audit');
    });

    it('renders no mutation affordance (read-only — no POST form in content)', async () => {
      mockGoClient.get.mockResolvedValue({ data: AUDIT_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      // The shared layout auth-bar uses method="GET" (default); no content-area POST form exists
      expect(res.text).not.toContain('method="POST"');
    });

    it('accepts legacy envelope responses without crashing', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: AUDIT_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('evt-001');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('network failure'));

      const res = await request(app)
        .get('/bff/admin/audit/evt-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});
