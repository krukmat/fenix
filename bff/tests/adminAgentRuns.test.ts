// BFF-ADMIN-20 / BFF-ADMIN-21a: admin agent runs list + detail header tests
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

const RUN_LIST = [
  {
    id: 'run-001',
    agentDefinitionId: 'agent-abc',
    triggerType: 'manual',
    status: 'success',
    totalTokens: 1200,
    totalCost: 0.04,
    latencyMs: 1850,
    startedAt: '2026-04-20T10:00:00Z',
    completedAt: '2026-04-20T10:00:01Z',
    createdAt: '2026-04-20T10:00:00Z',
  },
  {
    id: 'run-002',
    agentDefinitionId: 'agent-xyz',
    triggerType: 'event',
    status: 'failed',
    totalTokens: null,
    totalCost: null,
    latencyMs: null,
    startedAt: '2026-04-21T09:00:00Z',
    completedAt: null,
    createdAt: '2026-04-21T09:00:00Z',
  },
];

const BACKEND_RESPONSE = { data: RUN_LIST, meta: { total: 2, limit: 50, offset: 0 } };

describe('BFF admin agent runs list — BFF-ADMIN-20', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /bff/admin/agent-runs', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/agents/runs', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/agents/runs',
        expect.objectContaining({ params: expect.any(Object) }),
      );
    });

    it('renders run IDs in the table', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('run-001');
      expect(res.text).toContain('run-002');
    });

    it('renders status values', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('success');
      expect(res.text).toContain('failed');
    });

    it('renders a link to each run detail', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/agent-runs/run-001');
      expect(res.text).toContain('/bff/admin/agent-runs/run-002');
    });

    it('relays status filter to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [RUN_LIST[0]], meta: { total: 1, limit: 50, offset: 0 } }, status: 200 });

      await request(app)
        .get('/bff/admin/agent-runs?status=success')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/agents/runs',
        expect.objectContaining({ params: expect.objectContaining({ status: 'success' }) }),
      );
    });

    it('relays agent filter (workflow_id) to Go backend', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/agent-runs?workflow_id=wf-001')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/agents/runs',
        expect.objectContaining({ params: expect.objectContaining({ workflow_id: 'wf-001' }) }),
      );
    });

    it('relays date_from and date_to as offset/limit proxies are absent — passes them through as-is', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      await request(app)
        .get('/bff/admin/agent-runs?date_from=2026-04-01&date_to=2026-04-30')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith(
        '/api/v1/agents/runs',
        expect.objectContaining({ params: expect.objectContaining({ date_from: '2026-04-01', date_to: '2026-04-30' }) }),
      );
    });

    it('renders empty-state when list is empty', async () => {
      mockGoClient.get.mockResolvedValue({ data: { data: [], meta: { total: 0, limit: 50, offset: 0 } }, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('No agent runs found');
    });

    it('includes filter form with status and workflow_id inputs', async () => {
      mockGoClient.get.mockResolvedValue({ data: BACKEND_RESPONSE, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('name="status"');
      expect(res.text).toContain('name="workflow_id"');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('timeout'));

      const res = await request(app)
        .get('/bff/admin/agent-runs')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});

// BFF-ADMIN-21a / BFF-ADMIN-21b / BFF-ADMIN-21c: agent run detail — header, outcome, trace, evidence
const RUN_DETAIL = {
  id: 'run-001',
  agentDefinitionId: 'agent-abc',
  triggerType: 'manual',
  status: 'success',
  outcome: 'Case resolved via KB lookup.',
  abstainReason: null,
  totalTokens: 1200,
  totalCost: 0.04,
  latencyMs: 1850,
  startedAt: '2026-04-20T10:00:00Z',
  completedAt: '2026-04-20T10:00:01Z',
  createdAt: '2026-04-20T10:00:00Z',
  reasoningTrace: [
    { step: 0, thought: 'Retrieving relevant knowledge items.' },
    { step: 1, thought: 'Evaluating evidence confidence.' },
    { step: 2, thought: 'Generating response.' },
  ],
  retrievedEvidence: [
    {
      id: 'ev-001',
      sourceId: 'email_123',
      snippet: 'Customer reported issue with login timeout.',
      score: 0.95,
      confidence: 'high',
      timestamp: '2026-04-19T08:00:00Z',
    },
    {
      id: 'ev-002',
      sourceId: 'case_456',
      snippet: 'Similar case resolved by resetting auth token.',
      score: 0.81,
      confidence: 'medium',
      timestamp: '2026-04-18T14:30:00Z',
    },
  ],
};

const RUN_DETAIL_ABSTAINED = {
  id: 'run-003',
  agentDefinitionId: 'agent-xyz',
  triggerType: 'event',
  status: 'abstained',
  outcome: null,
  abstainReason: 'Insufficient evidence to proceed.',
  totalTokens: 300,
  totalCost: 0.01,
  latencyMs: 400,
  startedAt: '2026-04-22T08:00:00Z',
  completedAt: '2026-04-22T08:00:00Z',
  createdAt: '2026-04-22T08:00:00Z',
};

describe('BFF admin agent run detail header — BFF-ADMIN-21a', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /bff/admin/agent-runs/:id', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /api/v1/agents/runs/:id', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/api/v1/agents/runs/run-001');
    });

    it('renders the run id in the header', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('run-001');
    });

    it('renders agent definition id', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('agent-abc');
    });

    it('renders status badge', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('success');
    });

    it('renders outcome text', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Case resolved via KB lookup.');
    });

    it('renders abstain reason when present', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_ABSTAINED, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-003')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Insufficient evidence to proceed.');
    });

    it('does not render abstain section when abstainReason is null', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).not.toContain('Abstain reason');
    });

    it('renders startedAt timestamp', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('2026-04-20');
    });

    it('renders completedAt timestamp when present', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('2026-04-20T10:00:01');
    });

    it('renders back link to the list', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/agent-runs');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: { message: 'Unauthorized' } },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('network timeout'));

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});

// BFF-ADMIN-21b: reasoning trace fragment
describe('BFF admin agent run detail trace — BFF-ADMIN-21b', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /bff/admin/agent-runs/:id (trace section)', () => {
    it('renders Reasoning Trace section heading', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Reasoning Trace');
    });

    it('renders all trace step thoughts', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('Retrieving relevant knowledge items.');
      expect(res.text).toContain('Evaluating evidence confidence.');
      expect(res.text).toContain('Generating response.');
    });

    it('renders step numbers', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      // step numbers 1, 2, 3 rendered (1-indexed display)
      expect(res.text).toContain('Step 1');
      expect(res.text).toContain('Step 2');
      expect(res.text).toContain('Step 3');
    });

    it('renders empty-state when trace is absent', async () => {
      const runNoTrace = { ...RUN_DETAIL, reasoningTrace: undefined };
      mockGoClient.get.mockResolvedValue({ data: runNoTrace, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('No trace available');
    });

    it('renders empty-state when trace is empty array', async () => {
      const runEmptyTrace = { ...RUN_DETAIL, reasoningTrace: [] };
      mockGoClient.get.mockResolvedValue({ data: runEmptyTrace, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('No trace available');
    });

    it('paginates trace by trace_offset — shows only PAGE_SIZE steps from offset', async () => {
      const manySteps = Array.from({ length: 15 }, (_, i) => ({
        step: i,
        thought: `Thought number ${i}`,
      }));
      const runMany = { ...RUN_DETAIL, reasoningTrace: manySteps };
      mockGoClient.get.mockResolvedValue({ data: runMany, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001?trace_offset=10')
        .set('Authorization', 'Bearer test-token');

      // steps 10-14 visible
      expect(res.text).toContain('Thought number 10');
      expect(res.text).toContain('Thought number 14');
      // steps 0-9 NOT visible
      expect(res.text).not.toContain('Thought number 0');
      expect(res.text).not.toContain('Thought number 9');
    });

    it('renders next-page link when more steps remain', async () => {
      const manySteps = Array.from({ length: 15 }, (_, i) => ({
        step: i,
        thought: `Thought number ${i}`,
      }));
      const runMany = { ...RUN_DETAIL, reasoningTrace: manySteps };
      mockGoClient.get.mockResolvedValue({ data: runMany, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001?trace_offset=0')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('trace_offset=10');
    });

    it('renders prev-page link when offset > 0', async () => {
      const manySteps = Array.from({ length: 15 }, (_, i) => ({
        step: i,
        thought: `Thought number ${i}`,
      }));
      const runMany = { ...RUN_DETAIL, reasoningTrace: manySteps };
      mockGoClient.get.mockResolvedValue({ data: runMany, status: 200 });

      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001?trace_offset=10')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('trace_offset=0');
    });

    it('does not render next-page link on last page', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });

      // only 3 steps total, page size 10 → no next page
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001?trace_offset=0')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).not.toContain('trace_offset=10');
    });
  });
});

// BFF-ADMIN-21c: evidence pack fragment
describe('BFF admin agent run detail evidence — BFF-ADMIN-21c', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/agent-runs/:id (evidence section)', () => {
    it('renders Evidence section heading', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('Evidence');
    });

    it('renders source ids for each evidence item', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('email_123');
      expect(res.text).toContain('case_456');
    });

    it('renders snippets', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('Customer reported issue with login timeout.');
      expect(res.text).toContain('Similar case resolved by resetting auth token.');
    });

    it('renders scores', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('0.95');
      expect(res.text).toContain('0.81');
    });

    it('renders confidence tiers', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('high');
      expect(res.text).toContain('medium');
    });

    it('renders timestamps', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('2026-04-19');
      expect(res.text).toContain('2026-04-18');
    });

    it('renders empty-state when retrievedEvidence is absent', async () => {
      const runNoEvidence = { ...RUN_DETAIL, retrievedEvidence: undefined };
      mockGoClient.get.mockResolvedValue({ data: runNoEvidence, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('No evidence');
    });

    it('renders empty-state when retrievedEvidence is empty array', async () => {
      const runEmpty = { ...RUN_DETAIL, retrievedEvidence: [] };
      mockGoClient.get.mockResolvedValue({ data: runEmpty, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('No evidence');
    });
  });
});

// BFF-ADMIN-21d: tool calls and cost panel
const RUN_DETAIL_WITH_TOOLS = {
  ...RUN_DETAIL,
  toolCalls: [
    {
      toolName: 'create_task',
      status: 'success',
      latencyMs: 320,
      idempotencyKey: 'idem-key-001',
      input: { owner: 'user_123', title: 'Follow up' },
      output: { taskId: 'task-999' },
    },
    {
      toolName: 'send_email',
      status: 'denied',
      latencyMs: 12,
      idempotencyKey: null,
      input: { to: 'customer@example.com', templateId: 'tmpl-01' },
      output: null,
    },
  ],
  costTokens: 1200,
  costEuros: 0.04,
};

describe('BFF admin agent run detail tool calls and cost — BFF-ADMIN-21d', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/agent-runs/:id (tool calls section)', () => {
    it('renders Tool Calls section heading', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('Tool Calls');
    });

    it('renders tool names', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('create_task');
      expect(res.text).toContain('send_email');
    });

    it('renders status for each tool call', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('success');
      expect(res.text).toContain('denied');
    });

    it('renders latency for each tool call', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('320');
      expect(res.text).toContain('12');
    });

    it('renders idempotency key when present', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('idem-key-001');
    });

    it('renders dash when idempotency key is null', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      // second tool call has no idempotency key — dash rendered
      expect(res.text).toMatch(/—/);
    });

    it('renders empty-state when toolCalls is absent', async () => {
      const runNoTools = { ...RUN_DETAIL, toolCalls: undefined };
      mockGoClient.get.mockResolvedValue({ data: runNoTools, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('No tool calls');
    });

    it('renders empty-state when toolCalls is empty array', async () => {
      const runEmpty = { ...RUN_DETAIL, toolCalls: [] };
      mockGoClient.get.mockResolvedValue({ data: runEmpty, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('No tool calls');
    });
  });

  describe('GET /bff/admin/agent-runs/:id (cost panel)', () => {
    it('renders Cost section heading', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('Cost');
    });

    it('renders token count from backend payload', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('1200');
    });

    it('renders cost in euros from backend payload', async () => {
      mockGoClient.get.mockResolvedValue({ data: RUN_DETAIL_WITH_TOOLS, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('0.04');
    });

    it('renders dash for tokens when costTokens is absent', async () => {
      const runNoCost = { ...RUN_DETAIL, costTokens: undefined, costEuros: undefined };
      mockGoClient.get.mockResolvedValue({ data: runNoCost, status: 200 });
      const res = await request(app)
        .get('/bff/admin/agent-runs/run-001')
        .set('Authorization', 'Bearer test-token');
      expect(res.text).toContain('Cost');
      // values absent — dash rendered for both
      const costMatches = (res.text.match(/—/g) ?? []).length;
      expect(costMatches).toBeGreaterThanOrEqual(2);
    });
  });
});
