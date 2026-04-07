// W1-T2 (mobile_wedge_harmonization_plan): BFF approval alias routes tests
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = {
  put: jest.fn(),
};

jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

describe('BFF approval alias routes', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('POST /bff/api/v1/approvals/:id/approve', () => {
    it('calls Go PUT /api/v1/approvals/:id with decision=approve and returns 204', async () => {
      mockGoClient.put.mockResolvedValue({ data: null, status: 204 });

      const res = await request(app)
        .post('/bff/api/v1/approvals/approval-123/approve')
        .set('Authorization', 'Bearer test-token')
        .send({ reason: 'looks good' });

      expect(res.status).toBe(204);
      expect(mockGoClient.put).toHaveBeenCalledWith(
        '/api/v1/approvals/approval-123',
        { decision: 'approve', reason: 'looks good' }
      );
    });

    it('works without a reason body', async () => {
      mockGoClient.put.mockResolvedValue({ data: null, status: 204 });

      const res = await request(app)
        .post('/bff/api/v1/approvals/approval-456/approve')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(204);
      expect(mockGoClient.put).toHaveBeenCalledWith(
        '/api/v1/approvals/approval-456',
        { decision: 'approve', reason: undefined }
      );
    });
  });

  describe('POST /bff/api/v1/approvals/:id/reject', () => {
    it('calls Go PUT /api/v1/approvals/:id with decision=reject and returns 204', async () => {
      mockGoClient.put.mockResolvedValue({ data: null, status: 204 });

      const res = await request(app)
        .post('/bff/api/v1/approvals/approval-789/reject')
        .set('Authorization', 'Bearer test-token')
        .send({ reason: 'not authorized' });

      expect(res.status).toBe(204);
      expect(mockGoClient.put).toHaveBeenCalledWith(
        '/api/v1/approvals/approval-789',
        { decision: 'reject', reason: 'not authorized' }
      );
    });

    it('propagates Go errors via next()', async () => {
      mockGoClient.put.mockRejectedValue(new Error('backend down'));

      const res = await request(app)
        .post('/bff/api/v1/approvals/approval-000/reject')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});
