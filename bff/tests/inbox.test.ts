// W1-T3 (mobile_wedge_harmonization_plan): BFF inbox aggregation route tests
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = {
  get: jest.fn(),
};

jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

const sampleApproval = { id: 'appr-1', status: 'pending', action: 'send_email' };
const sampleSignal = { id: 'sig-1', status: 'active', signal_type: 'deal_risk' };
const sampleRun = { id: 'run-1', status: 'handed_off' };
const sampleHandoff = {
  run_id: 'run-1',
  reason: 'low confidence',
  conversation_context: 'ctx',
  evidence_count: 3,
  created_at: '2026-04-07T10:00:00Z',
};

describe('GET /bff/api/v1/mobile/inbox', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('returns aggregated approvals, handoffs, and signals', async () => {
    mockGoClient.get.mockImplementation((url: string) => {
      if (url === '/api/v1/approvals') return Promise.resolve({ data: { data: [sampleApproval] } });
      if (url === '/api/v1/signals') return Promise.resolve({ data: [sampleSignal] });
      if (url === '/api/v1/agents/runs') return Promise.resolve({ data: { data: [sampleRun] } });
      if (url === '/api/v1/agents/runs/run-1/handoff') return Promise.resolve({ data: sampleHandoff });
      return Promise.reject(new Error(`unexpected url: ${url}`));
    });

    const res = await request(app)
      .get('/bff/api/v1/mobile/inbox')
      .set('Authorization', 'Bearer test-token');

    expect(res.status).toBe(200);
    expect(res.body.approvals).toHaveLength(1);
    expect(res.body.approvals[0].id).toBe('appr-1');
    expect(res.body.signals).toHaveLength(1);
    expect(res.body.signals[0].id).toBe('sig-1');
    expect(res.body.handoffs).toHaveLength(1);
    expect(res.body.handoffs[0].run_id).toBe('run-1');
    expect(res.body.handoffs[0].handoff.reason).toBe('low confidence');
  });

  it('omits a handoff item when its enrichment fails — does not fail the whole response', async () => {
    mockGoClient.get.mockImplementation((url: string) => {
      if (url === '/api/v1/approvals') return Promise.resolve({ data: { data: [] } });
      if (url === '/api/v1/signals') return Promise.resolve({ data: [] });
      if (url === '/api/v1/agents/runs') return Promise.resolve({ data: { data: [sampleRun] } });
      if (url === '/api/v1/agents/runs/run-1/handoff') return Promise.reject(new Error('handoff not found'));
      return Promise.reject(new Error(`unexpected url: ${url}`));
    });

    const res = await request(app)
      .get('/bff/api/v1/mobile/inbox')
      .set('Authorization', 'Bearer test-token');

    expect(res.status).toBe(200);
    expect(res.body.handoffs).toHaveLength(0);
    expect(res.body.approvals).toHaveLength(0);
    expect(res.body.signals).toHaveLength(0);
  });

  it('returns empty arrays when Go calls fail', async () => {
    mockGoClient.get.mockRejectedValue(new Error('backend down'));

    const res = await request(app)
      .get('/bff/api/v1/mobile/inbox')
      .set('Authorization', 'Bearer test-token');

    expect(res.status).toBe(200);
    expect(res.body.approvals).toHaveLength(0);
    expect(res.body.signals).toHaveLength(0);
    expect(res.body.handoffs).toHaveLength(0);
  });
});
