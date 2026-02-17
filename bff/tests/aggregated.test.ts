// Task 4.1 — FR-301: Aggregated routes tests (TDD — written before implementation)
import request from 'supertest';

// Mock http-proxy-middleware (required at module import in routes/proxy.ts)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyHandlerFn = jest.fn((_req: any, _res: any, next: any) => next());
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const proxyStub = Object.assign(proxyHandlerFn, { upgrade: () => {} }) as any;
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

// Mock goClient to control Go backend responses
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 10 }),
}));

import { createGoClient } from '../src/services/goClient';
import app from '../src/app';

const mockCreateGoClient = createGoClient as jest.MockedFunction<typeof createGoClient>;

const ACCOUNT_ID = 'acc-abc123';
const DEAL_ID = 'deal-xyz789';
const CASE_ID = 'case-pqr456';

const mockAccount = { id: ACCOUNT_ID, name: 'Acme Corp', workspace_id: 'ws-1' };
const mockContacts = { items: [{ id: 'contact-1', name: 'John Doe' }] };
const mockDeals = { items: [{ id: DEAL_ID, title: 'Big Deal', stage: 'proposal' }] };
const mockTimeline = { items: [{ id: 'event-1', type: 'note', created_at: '2026-01-01T00:00:00Z' }] };
const mockDeal = { id: DEAL_ID, title: 'Big Deal', account_id: ACCOUNT_ID, contact_id: 'contact-1', stage: 'proposal' };
const mockActivities = { items: [{ id: 'act-1', type: 'call' }] };
const mockContact = { id: 'contact-1', name: 'John Doe' };
const mockCase = { id: CASE_ID, title: 'Support Issue', account_id: ACCOUNT_ID, contact_id: 'contact-1' };

describe('Aggregated routes', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /bff/accounts/:id/full', () => {
    it('combines account + contacts + deals + timeline into single response', async () => {
      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path.includes(`/accounts/${ACCOUNT_ID}`) && !path.includes('timeline')) {
            return Promise.resolve({ data: mockAccount });
          }
          if (path.includes('/contacts')) {
            return Promise.resolve({ data: mockContacts });
          }
          if (path.includes('/deals')) {
            return Promise.resolve({ data: mockDeals });
          }
          if (path.includes('timeline')) {
            return Promise.resolve({ data: mockTimeline });
          }
          return Promise.resolve({ data: null });
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/accounts/${ACCOUNT_ID}/full`)
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.body).toMatchObject({
        account: { id: ACCOUNT_ID, name: 'Acme Corp' },
        contacts: { items: expect.any(Array) },
        deals: { items: expect.any(Array) },
        timeline: { items: expect.any(Array) },
      });
    });

    it('returns null for failed sub-calls but still returns 200 (partial response)', async () => {
      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path.includes(`/accounts/${ACCOUNT_ID}`) && !path.includes('timeline')) {
            return Promise.resolve({ data: mockAccount });
          }
          if (path.includes('/contacts')) {
            return Promise.reject(new Error('contacts service unavailable'));
          }
          return Promise.resolve({ data: null });
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/accounts/${ACCOUNT_ID}/full`)
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.body.account).toMatchObject({ id: ACCOUNT_ID });
      expect(res.body.contacts).toBeNull();
    });
  });

  describe('GET /bff/deals/:id/full', () => {
    it('combines deal + account + contact + activities into single response', async () => {
      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path === `/api/v1/deals/${DEAL_ID}`) {
            return Promise.resolve({ data: mockDeal });
          }
          if (path.includes(`/accounts/${ACCOUNT_ID}`)) {
            return Promise.resolve({ data: mockAccount });
          }
          if (path.includes('/contacts/contact-1')) {
            return Promise.resolve({ data: mockContact });
          }
          if (path.includes('/activities')) {
            return Promise.resolve({ data: mockActivities });
          }
          return Promise.resolve({ data: null });
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/deals/${DEAL_ID}/full`)
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.body).toMatchObject({
        deal: { id: DEAL_ID },
        account: { id: ACCOUNT_ID },
        contact: { id: 'contact-1' },
        activities: { items: expect.any(Array) },
      });
    });
  });

  describe('GET /bff/deals/:id/full (no account/contact)', () => {
    it('returns null for account and contact when deal has no account_id/contact_id', async () => {
      const dealWithNoLinks = { id: DEAL_ID, title: 'Minimal Deal' }; // no account_id or contact_id

      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path === `/api/v1/deals/${DEAL_ID}`) {
            return Promise.resolve({ data: dealWithNoLinks });
          }
          if (path.includes('/activities')) {
            return Promise.resolve({ data: mockActivities });
          }
          return Promise.resolve({ data: null });
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/deals/${DEAL_ID}/full`)
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.body.deal).toMatchObject({ id: DEAL_ID });
      // No account_id/contact_id → null
      expect(res.body.account).toBeNull();
      expect(res.body.contact).toBeNull();
    });
  });

  describe('GET /bff/cases/:id/full', () => {
    it('combines case + account + contact + activities into single response', async () => {
      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path === `/api/v1/cases/${CASE_ID}`) {
            return Promise.resolve({ data: mockCase });
          }
          if (path.includes(`/accounts/${ACCOUNT_ID}`)) {
            return Promise.resolve({ data: mockAccount });
          }
          if (path.includes('/contacts/contact-1')) {
            return Promise.resolve({ data: mockContact });
          }
          if (path.includes('/activities')) {
            return Promise.resolve({ data: mockActivities });
          }
          return Promise.resolve({ data: null });
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/cases/${CASE_ID}/full`)
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.body).toMatchObject({
        case: { id: CASE_ID },
        account: { id: ACCOUNT_ID },
        contact: { id: 'contact-1' },
        activities: { items: expect.any(Array) },
        handoff: null, // no handoff_id in mockCase → null
      });
    });

    it('includes handoff when case has handoff_id', async () => {
      const HANDOFF_ID = 'handoff-xyz';
      const caseWithHandoff = { id: CASE_ID, title: 'Escalated Case', account_id: ACCOUNT_ID, contact_id: 'contact-1', handoff_id: HANDOFF_ID };
      const mockHandoff = { id: HANDOFF_ID, status: 'pending', reason: 'needs manager' };

      mockCreateGoClient.mockReturnValue({
        get: jest.fn().mockImplementation((path: string) => {
          if (path === `/api/v1/cases/${CASE_ID}`) return Promise.resolve({ data: caseWithHandoff });
          if (path.includes(`/accounts/${ACCOUNT_ID}`)) return Promise.resolve({ data: mockAccount });
          if (path.includes('/contacts/contact-1')) return Promise.resolve({ data: mockContact });
          if (path.includes('/activities')) return Promise.resolve({ data: mockActivities });
          if (path.includes(`/handoffs/${HANDOFF_ID}`)) return Promise.resolve({ data: mockHandoff });
          return Promise.resolve({ data: null });
        }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .get(`/bff/cases/${CASE_ID}/full`)
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.body.handoff).toMatchObject({ id: HANDOFF_ID, status: 'pending' });
    });
  });
});
