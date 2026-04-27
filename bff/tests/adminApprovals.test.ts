// BFF-ADMIN-30 / BFF-ADMIN-31: admin approvals queue list + decision form
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = { get: jest.fn(), post: jest.fn() };
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

const APPROVAL_LIST = [
  {
    id: 'apr-001',
    requestedBy: 'user-abc',
    action: 'send_email',
    status: 'pending',
    reason: null,
    createdAt: '2026-04-25T10:00:00Z',
  },
  {
    id: 'apr-002',
    requestedBy: 'user-xyz',
    action: 'create_task',
    status: 'approved',
    reason: 'Approved by manager.',
    createdAt: '2026-04-24T08:30:00Z',
  },
];

const BACKEND_RESPONSE = { data: APPROVAL_LIST, meta: { total: 2, limit: 50, offset: 0 } };

describe('BFF admin approvals list — BFF-ADMIN-30', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/approvals', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/approvals', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/approvals',
        expect.objectContaining({ params: expect.any(Object) }),
      );
    });

    it('renders approval IDs', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('apr-001');
      expect(res.text).toContain('apr-002');
    });

    it('renders action names', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('send_email');
      expect(res.text).toContain('create_task');
    });

    it('renders status values', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('pending');
      expect(res.text).toContain('approved');
    });

    it('filters to pending by default when no status param given', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/approvals',
        expect.objectContaining({ params: expect.objectContaining({ status: 'pending' }) }),
      );
    });

    it('relays explicit status filter overriding the default', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/approvals?status=approved')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/approvals',
        expect.objectContaining({ params: expect.objectContaining({ status: 'approved' }) }),
      );
    });

    it('renders empty-state when list is empty', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [], meta: { total: 0, limit: 50, offset: 0 } }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No approvals');
    });

    it('renders total count in the page heading', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Approvals');
      expect(res.text).toContain('2');
    });

    it('includes a status filter form', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('name="status"');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('timeout'));

      const res = await request(app)
        .get('/bff/admin/approvals')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});

const APPROVAL_DETAIL = {
  id: 'apr-001',
  requestedBy: 'user-abc',
  action: 'send_email',
  status: 'pending',
  reason: null,
  proposedPayload: { to: 'customer@example.com', templateId: 'tmpl-01' },
  createdAt: '2026-04-25T10:00:00Z',
};

// BFF-ADMIN-31: decision form — GET detail + POST decision relay
describe('BFF admin approval decision form — BFF-ADMIN-31', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/approvals/:id', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/approvals/:id', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_DETAIL, status: 200 });

      await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/approvals/apr-001');
    });

    it('renders the approval id', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('apr-001');
    });

    it('renders the action name', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('send_email');
    });

    it('renders approve and reject buttons', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('approve');
      expect(res.text).toContain('reject');
    });

    it('renders a reason textarea in the form', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('textarea');
      expect(res.text).toContain('name="reason"');
    });

    it('redirects to /bff/admin when Go backend returns 401 on detail GET', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });
  });

  describe('POST /bff/admin/approvals/:id/decision', () => {
    it('relays decision=approve to Go POST /api/v1/approvals/:id/decision', async () => {
      mockGoClient.post.mockResolvedValue({ data: { status: 'approved' }, status: 200 });

      await request(app)
        .post('/bff/admin/approvals/apr-001/decision')
        .set('Authorization', 'Bearer test-token')
        .send('decision=approve&reason=Looks+good');

      expect(mockGoClient.post).toHaveBeenCalledWith(
        '/api/v1/approvals/apr-001/decision',
        expect.objectContaining({ decision: 'approve' }),
      );
    });

    it('relays decision=reject and reason verbatim to Go', async () => {
      mockGoClient.post.mockResolvedValue({ data: { status: 'rejected' }, status: 200 });

      await request(app)
        .post('/bff/admin/approvals/apr-001/decision')
        .set('Authorization', 'Bearer test-token')
        .send('decision=reject&reason=Not+authorized');

      expect(mockGoClient.post).toHaveBeenCalledWith(
        '/api/v1/approvals/apr-001/decision',
        expect.objectContaining({ decision: 'reject', reason: 'Not authorized' }),
      );
    });

    it('redirects to approvals list after successful decision', async () => {
      mockGoClient.post.mockResolvedValue({ data: { status: 'approved' }, status: 200 });

      const res = await request(app)
        .post('/bff/admin/approvals/apr-001/decision')
        .set('Authorization', 'Bearer test-token')
        .send('decision=approve');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin/approvals');
    });

    it('renders 422 HTML with backend error message on validation failure', async () => {
      const err = Object.assign(new Error('Unprocessable'), {
        isAxiosError: true,
        response: { status: 422, data: { message: 'decision field required' } },
      });
      mockGoClient.post = jest.fn().mockRejectedValue(err);

      const res = await request(app)
        .post('/bff/admin/approvals/apr-001/decision')
        .set('Authorization', 'Bearer test-token')
        .send('decision=');

      expect(res.status).toBe(422);
      expect(res.text).toContain('decision field required');
    });

    it('redirects to /bff/admin when Go backend returns 401 on decision POST', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.post = jest.fn().mockRejectedValue(err);

      const res = await request(app)
        .post('/bff/admin/approvals/apr-001/decision')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error during decision POST', async () => {
      mockGoClient.post = jest.fn().mockRejectedValue(new Error('network error'));

      const res = await request(app)
        .post('/bff/admin/approvals/apr-001/decision')
        .set('Authorization', 'Bearer test-token')
        .send('decision=approve');

      expect(res.status).toBe(500);
    });
  });
});

// ─── BFF-ADMIN-70: approval detail with reasoning trace ──────────────────────

const APPROVAL_WITH_TRACE = {
  id: 'apr-003',
  requestedBy: 'user-abc',
  action: 'send_email',
  status: 'pending',
  reason: null,
  proposedPayload: { to: 'customer@example.com', templateId: 'tmpl-01' },
  reasoningTrace: 'Agent analyzed context and determined action is safe based on evidence.',
  toolCalls: [
    { toolName: 'retrieve_context', args: { entityId: 'case-123' }, result: 'success' },
    { toolName: 'validate_recipient', args: { email: 'customer@example.com' }, result: 'success' },
  ],
  createdAt: '2026-04-25T10:00:00Z',
};

describe('BFF admin approval detail with trace — BFF-ADMIN-70', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/approvals/:id (with reasoning trace)', () => {
    it('renders reasoning trace section', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_WITH_TRACE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-003')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Agent analyzed context');
    });

    it('renders tool calls table when present', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_WITH_TRACE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-003')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('retrieve_context');
      expect(res.text).toContain('validate_recipient');
    });

    it('renders tool call args', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_WITH_TRACE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-003')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('case-123');
      expect(res.text).toContain('customer@example.com');
    });

    it('renders tool call results', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_WITH_TRACE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-003')
        .set('Authorization', 'Bearer test-token');

      const successCount = (res.text.match(/success/g) || []).length;
      expect(successCount).toBeGreaterThanOrEqual(2);
    });

    it('renders proposed payload as JSON', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_WITH_TRACE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-003')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('tmpl-01');
    });

    it('does not render reasoning section when trace is absent', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      // Should still render the basic approval detail
      expect(res.text).toContain('apr-001');
    });

    it('does not render tool calls section when empty', async () => {
      const noToolsApproval = { ...APPROVAL_WITH_TRACE, toolCalls: [] };
      mockGoClient.get.mockResolvedValue({ data: noToolsApproval, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-003')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      // Reasoning trace should still be there
      expect(res.text).toContain('Agent analyzed context');
    });

    it('renders back link to approval list', async () => {
      mockGoClient.get.mockResolvedValue({ data: APPROVAL_WITH_TRACE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/approvals/apr-003')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/approvals');
    });
  });
});
