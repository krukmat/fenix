// BFF-ADMIN-10 / BFF-ADMIN-11: admin workflows list and detail page tests
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = { get: jest.fn(), post: jest.fn(), put: jest.fn() };
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

const WORKFLOW_LIST = [
  {
    id: 'wf-001',
    name: 'sales_followup',
    status: 'active',
    version: 3,
    description: 'Follow up on deals',
    created_at: '2026-04-01T10:00:00Z',
    updated_at: '2026-04-20T12:00:00Z',
  },
  {
    id: 'wf-002',
    name: 'triage_case',
    status: 'draft',
    version: 1,
    description: null,
    created_at: '2026-04-10T09:00:00Z',
    updated_at: '2026-04-10T09:00:00Z',
  },
];

describe('BFF admin workflows list — BFF-ADMIN-10', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /bff/admin/workflows', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_LIST }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/workflows', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_LIST }, status: 200 });

      await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/workflows',
        expect.objectContaining({ params: expect.any(Object) }),
      );
    });

    it('renders workflow names in the table', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_LIST }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('sales_followup');
      expect(res.text).toContain('triage_case');
    });

    it('renders workflow status values', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_LIST }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('active');
      expect(res.text).toContain('draft');
    });

    it('renders a link to each workflow detail', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_LIST }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/workflows/wf-001');
      expect(res.text).toContain('/bff/admin/workflows/wf-002');
    });

    it('relays status filter query param to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [WORKFLOW_LIST[0]] }, status: 200 });

      await request(app)
        .get('/bff/admin/workflows?status=active')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/workflows',
        expect.objectContaining({ params: expect.objectContaining({ status: 'active' }) }),
      );
    });

    it('relays name filter query param to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [WORKFLOW_LIST[0]] }, status: 200 });

      await request(app)
        .get('/bff/admin/workflows?name=sales')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/workflows',
        expect.objectContaining({ params: expect.objectContaining({ name: 'sales' }) }),
      );
    });

    it('renders an empty-state message when the list is empty', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [] }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No workflows found');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer expired-token');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('network timeout'));

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });

    it('includes filter form in the page', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_LIST }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('name="status"');
      expect(res.text).toContain('name="name"');
    });

    it('includes a create workflow action', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_LIST }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/workflows/new');
    });
  });
});

describe('BFF admin workflow draft creation — WFA-01', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /bff/admin/workflows/new', () => {
    it('renders the create draft form', async () => {
      const res = await request(app)
        .get('/bff/admin/workflows/new')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
      expect(res.text).toContain('Create workflow draft');
      expect(res.text).toContain('name="name"');
      expect(res.text).toContain('name="description"');
      expect(res.text).toContain('name="authoring_mode"');
    });
  });

  describe('POST /bff/admin/workflows', () => {
    it('calls Go POST /api/v1/workflows with the draft scaffold', async () => {
      mockGoClient.post.mockResolvedValue({ data: { data: { id: 'wf-new', name: 'sales_followup' } }, status: 201 });

      await request(app)
        .post('/bff/admin/workflows')
        .type('form')
        .send({ name: 'sales_followup', description: 'Follow up on deals', authoring_mode: 'visual' })
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.post).toHaveBeenCalledWith('/api/v1/workflows', {
        name: 'sales_followup',
        description: 'Follow up on deals',
        dsl_source: 'WORKFLOW sales_followup\nON case.created',
      });
    });

    it('redirects to the builder with the workflow context after create', async () => {
      mockGoClient.post.mockResolvedValue({ data: { data: { id: 'wf-new', name: 'sales_followup' } }, status: 201 });

      const res = await request(app)
        .post('/bff/admin/workflows')
        .type('form')
        .send({ name: 'sales_followup', description: 'Follow up on deals', authoring_mode: 'visual' })
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/builder?workflowId=wf-new');
    });

    it('re-renders the form with inline error on 422 from backend', async () => {
      const err = Object.assign(new Error('Unprocessable'), {
        isAxiosError: true,
        response: {
          status: 422,
          data: { message: 'workflow name is required' },
        },
      });
      mockGoClient.post.mockRejectedValue(err);

      const res = await request(app)
        .post('/bff/admin/workflows')
        .type('form')
        .send({ name: '', description: 'Follow up on deals', authoring_mode: 'visual' })
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
      expect(res.text).toContain('Create draft failed');
      expect(res.text).toContain('workflow name is required');
    });

    it('re-renders the form with inline error on 403 from backend', async () => {
      const err = Object.assign(new Error('Forbidden'), {
        isAxiosError: true,
        response: {
          status: 403,
          data: { message: 'forbidden' },
        },
      });
      mockGoClient.post.mockRejectedValue(err);

      const res = await request(app)
        .post('/bff/admin/workflows')
        .type('form')
        .send({ name: 'sales_followup', description: 'Follow up on deals', authoring_mode: 'visual' })
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('forbidden');
      expect(res.text).toContain('name="name"');
    });

    it('redirects to /bff/admin on 401 from upstream', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.post.mockRejectedValue(err);

      const res = await request(app)
        .post('/bff/admin/workflows')
        .type('form')
        .send({ name: 'sales_followup', description: 'Follow up on deals', authoring_mode: 'visual' })
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.post.mockRejectedValue(new Error('network timeout'));

      const res = await request(app)
        .post('/bff/admin/workflows')
        .type('form')
        .send({ name: 'sales_followup', description: 'Follow up on deals', authoring_mode: 'visual' })
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});

// BFF-ADMIN-11: workflow detail page (read-only)
const WORKFLOW_DETAIL = {
  id: 'wf-001',
  name: 'sales_followup',
  status: 'active',
  version: 3,
  description: 'Follow up on deals',
  dsl_source: 'WORKFLOW sales_followup\nON deal.updated',
  spec_source: 'CARTA sales_followup\nAGENT sales_assistant',
  created_at: '2026-04-01T10:00:00Z',
  updated_at: '2026-04-20T12:00:00Z',
};

describe('BFF admin workflow detail — BFF-ADMIN-11', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /bff/admin/workflows/:id', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/workflows/:id', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/workflows/wf-001');
    });

    it('renders workflow name and status', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('sales_followup');
      expect(res.text).toContain('active');
    });

    it('renders DSL source in a code block', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('WORKFLOW sales_followup');
    });

    it('renders spec source when present', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('CARTA sales_followup');
    });

    it('renders version number', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('3');
    });

    it('includes a link to the builder for this workflow', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/builder?workflowId=wf-001');
    });

    it('includes a link back to the workflow list from detail actions', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Back to workflow list');
    });

    it('renders a status hint for draft workflows', async () => {
      const draftDetail = { ...WORKFLOW_DETAIL, status: 'draft' };
      mockGoClient.get.mockResolvedValue({ data: { data: draftDetail }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Workflow status');
      expect(res.text).toContain('Draft: editable in builder and not yet ready for activation.');
    });

    it('includes a back link to the workflows list', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/workflows');
    });

    it('renders activation section with activate form wired (BFF-ADMIN-12)', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('id="activation-section"');
    });

    it('handles missing spec_source gracefully', async () => {
      const noSpec = { ...WORKFLOW_DETAIL, spec_source: null };
      mockGoClient.get.mockResolvedValue({ data: { data: noSpec }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No spec source');
    });

    it('redirects to /bff/admin on 401 from upstream', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('backend down'));

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});

// BFF-ADMIN-12: activation form — POST /:id/activate
describe('BFF admin workflow activation — BFF-ADMIN-12', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('POST /bff/admin/workflows/:id/activate', () => {
    it('calls Go PUT /api/v1/workflows/:id/activate with the bearer token', async () => {
      mockGoClient.put = jest.fn().mockResolvedValue({ data: { ...WORKFLOW_DETAIL, status: 'active' }, status: 200 });

      await request(app)
        .post('/bff/admin/workflows/wf-001/activate')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.put).toHaveBeenCalledWith('/api/v1/workflows/wf-001/activate');
    });

    it('redirects to the detail page after successful activation', async () => {
      mockGoClient.put = jest.fn().mockResolvedValue({ data: { ...WORKFLOW_DETAIL, status: 'active' }, status: 200 });

      const res = await request(app)
        .post('/bff/admin/workflows/wf-001/activate')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin/workflows/wf-001');
    });

    it('re-renders the detail page with error message on 422 from backend', async () => {
      const err = Object.assign(new Error('Unprocessable'), {
        isAxiosError: true,
        response: {
          status: 422,
          data: { message: 'workflow must be in testing to activate' },
        },
      });
      mockGoClient.put = jest.fn().mockRejectedValue(err);
      // detail GET needed for re-render
      mockGoClient.get = jest.fn().mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .post('/bff/admin/workflows/wf-001/activate')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
      expect(res.text).toContain('workflow must be in testing to activate');
    });

    it('redirects to /bff/admin on 401 from upstream', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.put = jest.fn().mockRejectedValue(err);

      const res = await request(app)
        .post('/bff/admin/workflows/wf-001/activate')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.put = jest.fn().mockRejectedValue(new Error('network failure'));

      const res = await request(app)
        .post('/bff/admin/workflows/wf-001/activate')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });

  describe('GET /bff/admin/workflows/:id (with BFF-ADMIN-12 form wired)', () => {
    it('renders the activate button in the activation section', async () => {
      mockGoClient.get = jest.fn().mockResolvedValue({ data: { data: WORKFLOW_DETAIL }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/workflows/wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/workflows/wf-001/activate');
    });
  });
});
